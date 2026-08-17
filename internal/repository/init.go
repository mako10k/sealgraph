package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const configBytes = "repository_format = 4\nobject_format = sha256\n"

// InitStandalone initializes only workDir/.sealgraph. It never probes Git or
// searches parent directories.
func InitStandalone(workDir string) (bool, error) {
	repositoryDir := filepath.Join(workDir, ".sealgraph")
	info, err := os.Lstat(repositoryDir)
	if err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return false, fmt.Errorf("%s exists but is not a standalone sealgraph directory; inspect it and choose a different directory", repositoryDir)
		}
		if err := validateCanonicalLayout(repositoryDir); err != nil {
			return false, fmt.Errorf("%s exists but is not a valid standalone repository: %w; repair it explicitly before retrying", repositoryDir, err)
		}
		if err := bootstrapRuntimeLayout(repositoryDir); err != nil {
			return false, fmt.Errorf("%s has unsafe runtime state: %w; inspect it explicitly before retrying", repositoryDir, err)
		}
		return false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("inspect %s: %w", repositoryDir, err)
	}

	staging, err := os.MkdirTemp(workDir, ".sealgraph-init-")
	if err != nil {
		return false, fmt.Errorf("create initialization staging directory: %w", err)
	}
	defer os.RemoveAll(staging)
	for _, relative := range []string{"objects", filepath.Join("refs", "seals"), filepath.Join("refs", "tags"), "index", "locks"} {
		if err := os.MkdirAll(filepath.Join(staging, relative), 0o755); err != nil {
			return false, fmt.Errorf("prepare repository layout: %w", err)
		}
	}
	if err := os.WriteFile(filepath.Join(staging, "config"), []byte(configBytes), 0o644); err != nil {
		return false, fmt.Errorf("write repository config: %w", err)
	}
	if err := os.Rename(staging, repositoryDir); err != nil {
		if _, statErr := os.Lstat(repositoryDir); statErr == nil {
			return false, fmt.Errorf("%s appeared during initialization; retry to validate it: %w", repositoryDir, err)
		}
		return false, fmt.Errorf("publish standalone repository atomically: %w", err)
	}
	return true, nil
}

func validateLayout(repositoryDir string) error {
	if err := validateCanonicalLayout(repositoryDir); err != nil {
		return err
	}
	for _, relative := range []string{"index", "locks"} {
		if err := validateRealDirectory(filepath.Join(repositoryDir, relative), relative); err != nil {
			return err
		}
	}
	return nil
}

func validateCanonicalLayout(repositoryDir string) error {
	configPath := filepath.Join(repositoryDir, "config")
	configInfo, err := os.Lstat(configPath)
	if err != nil {
		return fmt.Errorf("inspect config: %w", err)
	}
	if !configInfo.Mode().IsRegular() || configInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("config is not a regular file")
	}
	config, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if string(config) != configBytes {
		return fmt.Errorf("unsupported or malformed config")
	}
	for _, relative := range []string{"objects", "refs", filepath.Join("refs", "seals"), filepath.Join("refs", "tags")} {
		if err := validateRealDirectory(filepath.Join(repositoryDir, relative), relative); err != nil {
			return err
		}
	}
	return nil
}

func bootstrapRuntimeLayout(repositoryDir string) error {
	missing := make([]string, 0, 2)
	for _, relative := range []string{"index", "locks"} {
		path := filepath.Join(repositoryDir, relative)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			missing = append(missing, relative)
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect %s: %w", relative, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is not a real directory", relative)
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
			return fmt.Errorf("create %s runtime directory: %w", relative, err)
		}
	}
	return nil
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
