package repository

import (
	"context"
	"fmt"
)

type writerGuard struct {
	lockDir      string
	afterAcquire func()
}

func newWriterGuard(lockDir string) writerGuard {
	return writerGuard{lockDir: lockDir}
}

func withMutation[T any](ctx context.Context, guard writerGuard, operation string, mutate func() (T, error)) (result T, err error) {
	release, err := acquireProcessWriter(ctx, guard.lockDir)
	if err != nil {
		return result, fmt.Errorf("%s: acquire repository writer guard: %w", operation, err)
	}
	defer func() {
		if releaseErr := release(); releaseErr != nil {
			if err == nil {
				err = fmt.Errorf("%s completed but repository writer guard release failed: %w", operation, releaseErr)
				return
			}
			err = fmt.Errorf("%w; repository writer guard release also failed: %v", err, releaseErr)
		}
	}()
	if guard.afterAcquire != nil {
		guard.afterAcquire()
	}
	return mutate()
}
