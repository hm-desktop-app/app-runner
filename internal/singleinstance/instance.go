package singleinstance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Primary struct {
	file       *os.File
	activation string
}

// Acquire returns nil without an error if another instance owns the lock and
// has been sent a request to show its window.
func Acquire() (*Primary, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	return acquire(filepath.Join(cache, "app-runner"))
}

func acquire(dir string) (*Primary, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(dir, "instance.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	activation := filepath.Join(dir, "instance.activate")
	locked, err := tryLock(file)
	if locked {
		return &Primary{file: file, activation: activation}, nil
	}
	_ = file.Close()
	if err != nil {
		return nil, err
	}
	// A marker can be queued even while the first window is still starting.
	// Exclusive creation coalesces concurrent activation requests.
	signal, err := os.OpenFile(activation, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if errors.Is(err, os.ErrExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return nil, signal.Close()
}

// Close releases the OS lock. Keep the file so every process opens the same
// lock target, even if another process starts during shutdown.
func (p *Primary) Close() error { return p.file.Close() }

func (p *Primary) Watch(ctx context.Context, activate func()) {
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for ctx.Err() == nil {
		if os.Remove(p.activation) == nil {
			activate()
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
