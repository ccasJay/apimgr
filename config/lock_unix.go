//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly
// +build linux darwin freebsd netbsd openbsd dragonfly

package config

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
	"os"
)

const (
	lockTimeout     = 5 * time.Second
	lockRetryDelay  = 50 * time.Millisecond
)

// lockFileExclusive acquires an exclusive lock (write lock) with timeout
func lockFileExclusive(f *os.File) error {
	return lockWithTimeout(f, unix.LOCK_EX)
}

// lockFileShared acquires a shared lock (read lock) with timeout
func lockFileShared(f *os.File) error {
	return lockWithTimeout(f, unix.LOCK_SH)
}

// lockWithTimeout attempts to acquire a lock with timeout to prevent blocking
func lockWithTimeout(f *os.File, lockType int) error {
	deadline := time.Now().Add(lockTimeout)
	
	for {
		// Try non-blocking lock first
		err := unix.Flock(int(f.Fd()), lockType|unix.LOCK_NB)
		if err == nil {
			return nil
		}
		
		// If error is not EWOULDBLOCK, return immediately
		if err != unix.EWOULDBLOCK && err != unix.EAGAIN {
			return fmt.Errorf("failed to acquire lock: %w", err)
		}
		
		// Check timeout
		if time.Now().After(deadline) {
			return fmt.Errorf("lock timeout: file is locked by another process")
		}
		
		// Wait before retry
		time.Sleep(lockRetryDelay)
	}
}

// unlockFile releases the file lock
func unlockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
