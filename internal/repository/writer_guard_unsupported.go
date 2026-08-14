//go:build !android && !darwin && !dragonfly && !freebsd && !illumos && !linux && !netbsd && !openbsd

package repository

import (
	"context"
	"fmt"
	"runtime"
)

func acquireProcessWriter(context.Context, string) (func() error, error) {
	return nil, fmt.Errorf("safe process-lifetime repository locking is not implemented on %s; no mutation was performed", runtime.GOOS)
}
