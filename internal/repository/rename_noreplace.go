package repository

import "github.com/mako10k/sealgraph/internal/fsatomic"

func renameNoReplace(oldPath, newPath string) error {
	return fsatomic.RenameNoReplace(oldPath, newPath)
}
