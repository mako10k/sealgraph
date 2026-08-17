//go:build !linux || !amd64

package fsatomic

import "fmt"

func RenameNoReplace(_, _ string) error {
	return fmt.Errorf("atomic no-replace rename is unsupported on this platform")
}
