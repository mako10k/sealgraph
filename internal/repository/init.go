package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const configBytes = "repository_format = 1\nobject_format = sha256\n"

// InitStandalone initializes only workDir/.sealgraph. It never probes Git or
// searches parent directories.
func InitStandalone(workDir string) (bool, error) {
	repositoryDir := filepath.Join(workDir, ".sealgraph")
	info, err := os.Lstat(repositoryDir)
	if err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return false, fmt.Errorf("%s exists but is not a standalone sealgraph directory; inspect it and choose a different directory", repositoryDir)
		}
		if err := validateLayout(repositoryDir); err != nil {
			return false, fmt.Errorf("%s exists but is not a valid standalone repository: %w; repair it explicitly before retrying", repositoryDir, err)
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
	for _, relative := range []string{"objects", filepath.Join("refs", "seals"), "index", "locks"} {
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
	for _, relative := range []string{"objects", filepath.Join("refs", "seals"), "index", "locks"} {
		path := filepath.Join(repositoryDir, relative)
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect %s: %w", relative, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is not a directory", relative)
		}
	}
	return nil
}
