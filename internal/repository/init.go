package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configBytes        = "repository_format = 5\nobject_format = sha256\nref_format = manifest-v1\n"
	format6ConfigBytes = "repository_format = 6\nobject_format = sha256\nref_format = manifest-v1\n"
	format4ConfigBytes = "repository_format = 4\nobject_format = sha256\nref_format = manifest-v1\n"
)

const recommendedGitignore = `# Local runtime state; keep config, objects and REF manifests tracked.
/index/
/cache/
/locks/
/logs/

# Temporary files used for atomic canonical writes.
/objects/*/.tmp-object-*
/refs/seals/**/.tmp-ref-*
`

const format4MigrationGuide = "FORMAT4_REQUIRES_MIGRATION: ordinary format-5 operations cannot open format-4 repositories; extract read-only with 'sealgraph migrate extract --source-format 4 --format universal-blob-v1 > repository.dump.json', then from an absent target import with 'sealgraph load --format universal-blob-v1 < repository.dump.json'; no in-place migration or general compatibility reader is available"

type InitOutcome string

const (
	InitInitialized         InitOutcome = "initialized"
	InitRuntimeBootstrapped InitOutcome = "runtime_bootstrapped"
	InitAlreadyComplete     InitOutcome = "already_complete"
)

type InitResult struct {
	Outcome            InitOutcome
	RuntimeDirectories []string
}

// InitStandalone initializes only workDir/.sealgraph. It never probes Git or
// searches parent directories.
func InitStandalone(workDir string) (InitResult, error) {
	repositoryDir := filepath.Join(workDir, ".sealgraph")
	info, err := os.Lstat(repositoryDir)
	if err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return InitResult{}, fmt.Errorf("%s exists but is not a standalone sealgraph directory; inspect it and choose a different directory", repositoryDir)
		}
		if err := validateCanonicalLayout(repositoryDir); err != nil {
			return InitResult{}, fmt.Errorf("%s exists but is not a valid standalone repository: %w; repair it explicitly before retrying", repositoryDir, err)
		}
		created, err := bootstrapRuntimeLayout(repositoryDir)
		if err != nil {
			return InitResult{}, fmt.Errorf("%s has unsafe runtime state: %w; inspect it explicitly before retrying", repositoryDir, err)
		}
		if len(created) != 0 {
			return InitResult{Outcome: InitRuntimeBootstrapped, RuntimeDirectories: created}, nil
		}
		return InitResult{Outcome: InitAlreadyComplete, RuntimeDirectories: []string{}}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return InitResult{}, fmt.Errorf("inspect %s: %w", repositoryDir, err)
	}

	staging, err := os.MkdirTemp(workDir, ".sealgraph-init-")
	if err != nil {
		return InitResult{}, fmt.Errorf("create initialization staging directory: %w", err)
	}
	defer os.RemoveAll(staging)
	for _, relative := range []string{"objects", filepath.Join("refs", "seals"), "index", "locks"} {
		if err := os.MkdirAll(filepath.Join(staging, relative), 0o755); err != nil {
			return InitResult{}, fmt.Errorf("prepare repository layout: %w", err)
		}
	}
	if err := writeSyncedFile(filepath.Join(staging, "config"), []byte(configBytes), 0o644); err != nil {
		return InitResult{}, fmt.Errorf("write repository config: %w", err)
	}
	if err := writeSyncedFile(filepath.Join(staging, ".gitignore"), []byte(recommendedGitignore), 0o644); err != nil {
		return InitResult{}, fmt.Errorf("write recommended gitignore: %w", err)
	}
	if err := syncStagingTree(staging); err != nil {
		return InitResult{}, fmt.Errorf("synchronize initialization staging tree: %w", err)
	}
	if err := verifyUniversalLoadModes(staging); err != nil {
		return InitResult{}, fmt.Errorf("verify initialization creation modes: %w", err)
	}
	if err := renameNoReplace(staging, repositoryDir); err != nil {
		if _, statErr := os.Lstat(repositoryDir); statErr == nil {
			return InitResult{}, fmt.Errorf("%s appeared during initialization; retry to validate it: %w", repositoryDir, err)
		}
		return InitResult{}, fmt.Errorf("publish standalone repository atomically: %w", err)
	}
	if err := syncDirectoryForLoad(workDir); err != nil {
		return InitResult{}, fmt.Errorf("standalone repository was published at %s but parent-directory durability is uncertain; inspect it before retrying: %w", repositoryDir, err)
	}
	return InitResult{Outcome: InitInitialized, RuntimeDirectories: []string{"index", "locks"}}, nil
}

func validateLayout(repositoryDir string) error {
	if _, err := validateFormatLayout(repositoryDir); err != nil {
		return err
	}
	for _, relative := range []string{"index", "locks"} {
		if err := validateRealDirectory(filepath.Join(repositoryDir, relative), relative); err != nil {
			return err
		}
	}
	return nil
}

func repositoryFormat(repositoryDir string) (int, error) {
	configPath := filepath.Join(repositoryDir, "config")
	configInfo, err := os.Lstat(configPath)
	if err != nil {
		return 0, fmt.Errorf("inspect config: %w", err)
	}
	if !configInfo.Mode().IsRegular() || configInfo.Mode()&os.ModeSymlink != 0 {
		return 0, fmt.Errorf("config is not a regular file")
	}
	config, err := os.ReadFile(configPath)
	if err != nil {
		return 0, fmt.Errorf("read config: %w", err)
	}
	switch string(config) {
	case configBytes:
		return 5, nil
	case format6ConfigBytes:
		return 6, nil
	case format4ConfigBytes:
		return 0, errors.New(format4MigrationGuide)
	default:
		return 0, fmt.Errorf("unsupported or malformed config")
	}
}

func validateFormatLayout(repositoryDir string) (int, error) {
	format, err := repositoryFormat(repositoryDir)
	if err != nil {
		return 0, err
	}
	for _, relative := range []string{"objects", "refs", filepath.Join("refs", "seals")} {
		if err := validateRealDirectory(filepath.Join(repositoryDir, relative), relative); err != nil {
			return 0, err
		}
	}
	entries, err := os.ReadDir(filepath.Join(repositoryDir, "refs"))
	if err != nil {
		return 0, fmt.Errorf("list canonical refs directory: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() != "seals" {
			return 0, fmt.Errorf("unexpected canonical refs entry %q; manifest-v1 stores tags inside refs/seals/<REF>/.ref", entry.Name())
		}
	}
	return format, nil
}

func validateCanonicalLayout(repositoryDir string) error {
	format, err := validateFormatLayout(repositoryDir)
	if err != nil {
		return err
	}
	if format != 5 {
		return fmt.Errorf("init supports repository format 5; found format %d", format)
	}
	return nil
}

func bootstrapRuntimeLayout(repositoryDir string) ([]string, error) {
	missing := make([]string, 0, 2)
	for _, relative := range []string{"index", "locks"} {
		path := filepath.Join(repositoryDir, relative)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			missing = append(missing, relative)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", relative, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%s is not a real directory", relative)
		}
	}
	for _, relative := range missing {
		path := filepath.Join(repositoryDir, relative)
		if err := os.Mkdir(path, 0o755); err != nil {
			if errors.Is(err, os.ErrExist) {
				if validateErr := validateRealDirectory(path, relative); validateErr == nil {
					continue
				}
			}
			return nil, fmt.Errorf("create %s runtime directory: %w", relative, err)
		}
		if err := os.Chmod(path, 0o755); err != nil {
			return nil, fmt.Errorf("set %s runtime directory creation mode: %w", relative, err)
		}
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode().Perm() != 0o755 {
			return nil, fmt.Errorf("verify %s runtime directory creation mode: mode=%v err=%v", relative, initMode(info), err)
		}
	}
	return missing, nil
}

func initMode(info os.FileInfo) os.FileMode {
	if info == nil {
		return 0
	}
	return info.Mode()
}

func validateRealDirectory(path, relative string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", relative, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is not a real directory", relative)
	}
	return nil
}
