//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly
// +build linux darwin freebsd netbsd openbsd dragonfly

package config

import (
	"golang.org/x/sys/unix"
	"os"
)

// lockFileExclusive acquires an exclusive lock (write lock)
func lockFileExclusive(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX)
}

// lockFileShared acquires a shared lock (read lock)
func lockFileShared(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_SH)
}

// unlockFile releases the file lock
func unlockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
