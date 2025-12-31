//go:build windows
// +build windows

package config

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	lockfileExclusiveLock   = 0x00000002
	lockfileFailImmediately = 0x00000001
	lockTimeout             = 5 * time.Second
	lockRetryDelay          = 50 * time.Millisecond
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

// lockFileExclusive acquires an exclusive lock (write lock) with timeout
func lockFileExclusive(f *os.File) error {
	return lockWithTimeout(f, lockfileExclusiveLock)
}

// lockFileShared acquires a shared lock (read lock) with timeout
func lockFileShared(f *os.File) error {
	return lockWithTimeout(f, 0)
}

// lockWithTimeout attempts to acquire a lock with timeout to prevent blocking
func lockWithTimeout(f *os.File, flags uintptr) error {
	deadline := time.Now().Add(lockTimeout)
	
	for {
		var overlapped syscall.Overlapped
		// Use LOCKFILE_FAIL_IMMEDIATELY for non-blocking attempt
		r1, _, err := procLockFileEx.Call(
			uintptr(f.Fd()),
			flags|lockfileFailImmediately,
			0,
			1,
			0,
			uintptr(unsafe.Pointer(&overlapped)),
		)
		
		if r1 != 0 {
			return nil
		}
		
		// Check timeout
		if time.Now().After(deadline) {
			return fmt.Errorf("lock timeout: file is locked by another process")
		}
		
		// Wait before retry
		time.Sleep(lockRetryDelay)
		
		// Ignore the error from Call, we'll retry
		_ = err
	}
}

// unlockFile releases the file lock
func unlockFile(f *os.File) error {
	var overlapped syscall.Overlapped
	r1, _, err := procUnlockFileEx.Call(
		uintptr(f.Fd()),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if r1 == 0 {
		return err
	}
	return nil
}
