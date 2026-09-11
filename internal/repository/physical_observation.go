package repository

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/mako10k/sealgraph/internal/fsread"
)

type physicalEntryObservation struct {
	Path    string
	Present bool
	Mode    os.FileMode
	Size    int64
	SHA256  [sha256.Size]byte
	Data    []byte
}

type physicalRepositoryObservation struct {
	Entries []physicalEntryObservation
}

func capturePhysicalRepositoryObservation(repositoryDir string) (physicalRepositoryObservation, error) {
	entries := []physicalEntryObservation{}
	root, err := observePhysicalPath(repositoryDir, "ROOT", false)
	if err != nil {
		return physicalRepositoryObservation{}, err
	}
	entries = append(entries, root)
	config, err := observePhysicalPath(filepath.Join(repositoryDir, "config"), "CONFIG", true)
	if err != nil {
		return physicalRepositoryObservation{}, err
	}
	entries = append(entries, config)
	for _, namespace := range []string{"objects", "refs"} {
		captured, err := capturePhysicalNamespace(repositoryDir, namespace)
		if err != nil {
			return physicalRepositoryObservation{}, err
		}
		entries = append(entries, captured...)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return physicalRepositoryObservation{Entries: entries}, nil
}

func capturePhysicalNamespace(repositoryDir, relative string) ([]physicalEntryObservation, error) {
	root := filepath.Join(repositoryDir, relative)
	entries := []physicalEntryObservation{}
	err := filepath.WalkDir(root, func(path string, _ os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name, err := filepath.Rel(repositoryDir, path)
		if err != nil {
			return err
		}
		observed, err := observePhysicalPath(path, filepath.ToSlash(name), false)
		if err != nil {
			return err
		}
		entries = append(entries, observed)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("capture physical namespace %s: %w", relative, err)
	}
	return entries, nil
}

func observePhysicalPath(path, label string, allowAbsent bool) (physicalEntryObservation, error) {
	info, err := os.Lstat(path)
	if allowAbsent && os.IsNotExist(err) {
		return physicalEntryObservation{Path: label}, nil
	}
	if err != nil {
		return physicalEntryObservation{}, fmt.Errorf("inspect physical path %s: %w", label, err)
	}
	result := physicalEntryObservation{Path: label, Present: true, Mode: info.Mode()}
	if !info.Mode().IsRegular() {
		return result, nil
	}
	file, err := openFileNoFollow(path)
	if err != nil {
		return physicalEntryObservation{}, fmt.Errorf("open physical file %s: %w", label, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return physicalEntryObservation{}, fmt.Errorf("physical file %s changed while it was opened", label)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return physicalEntryObservation{}, fmt.Errorf("read physical file %s: %w", label, err)
	}
	result.SHA256 = sha256.Sum256(data)
	result.Size = int64(len(data))
	result.Data = data
	return result, nil
}

func openFileNoFollow(path string) (*os.File, error) {
	return fsread.OpenFileNoFollow(path)
}

func openDirectoryNoFollow(path string) (*os.File, error) {
	return fsread.OpenDirectoryNoFollow(path)
}

func equalPhysicalRepositoryObservations(left, right physicalRepositoryObservation) bool {
	if len(left.Entries) != len(right.Entries) {
		return false
	}
	for index := range left.Entries {
		leftEntry, rightEntry := left.Entries[index], right.Entries[index]
		if leftEntry.Path != rightEntry.Path || leftEntry.Present != rightEntry.Present || leftEntry.Mode != rightEntry.Mode ||
			leftEntry.Size != rightEntry.Size || leftEntry.SHA256 != rightEntry.SHA256 {
			return false
		}
	}
	return true
}
