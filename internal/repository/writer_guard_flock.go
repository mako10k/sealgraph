//go:build android || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd

package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

const writerLockPollInterval = 10 * time.Millisecond

func acquireProcessWriter(ctx context.Context, lockDir string) (func() error, error) {
	file, err := os.Open(lockDir)
	if err != nil {
		return nil, fmt.Errorf("open runtime lock directory: %w", err)
	}
	ticker := time.NewTicker(writerLockPollInterval)
	defer ticker.Stop()
	for {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			_ = file.Close()
			return nil, fmt.Errorf("lock runtime directory: %w", err)
		}
		select {
		case <-ctx.Done():
			_ = file.Close()
			return nil, fmt.Errorf("wait for active repository mutation: %w", ctx.Err())
		case <-ticker.C:
		}
	}

	released := false
	return func() error {
		if released {
			return nil
		}
		released = true
		unlockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		closeErr := file.Close()
		return errors.Join(unlockErr, closeErr)
	}, nil
}
