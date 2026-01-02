package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
)

// Backup constants
const (
	// DefaultBackupRetention is the default number of backups to keep
	DefaultBackupRetention = 3
)

// BackupManager manages backup files for configurations
type BackupManager struct {
	// MaxBackups is the maximum number of backups to retain
	MaxBackups int
}

// NewBackupManager creates a new BackupManager with default settings
func NewBackupManager(maxBackups int) *BackupManager {
	if maxBackups <= 0 {
		maxBackups = DefaultBackupRetention
	}
	return &BackupManager{
		MaxBackups: maxBackups,
	}
}

// CreateBackup creates a new backup file with timestamp-PID naming format
func (bm *BackupManager) CreateBackup(filePath string) (string, error) {
	// Get PID for the backup filename
	pid := syscall.Getpid()

	// Create backup filename with pattern: original.backup-YYYYMMDDHHMMSS-PID
	timestamp := time.Now().Format("20060102150405")
	backupPath := fmt.Sprintf("%s.backup-%s-%d", filePath, timestamp, pid)

	// Copy the file to create backup
	if err := copyFile(filePath, backupPath); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	// Preserve file permissions
	srcInfo, err := os.Stat(filePath)
	if err != nil {
		return backupPath, nil // Non-fatal, backup was created
	}
	if err := os.Chmod(backupPath, srcInfo.Mode()); err != nil {
		return backupPath, nil // Non-fatal, backup was created
	}

	return backupPath, nil
}

// ListBackups returns a list of all backup files for the given filePath
func (bm *BackupManager) ListBackups(filePath string) ([]string, error) {
	// Pattern to match backup files
	pattern := fmt.Sprintf("%s.backup-*", filePath)

	// Find all matching backup files
	backupFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	// Sort backup files by modification time (oldest first)
	sort.Slice(backupFiles, func(i, j int) bool {
		iInfo, err1 := os.Stat(backupFiles[i])
		jInfo, err2 := os.Stat(backupFiles[j])
		if err1 != nil || err2 != nil {
			return false
		}
		return iInfo.ModTime().Before(jInfo.ModTime())
	})

	return backupFiles, nil
}

// CleanupOldBackups removes old backup files, retaining only the most recent MaxBackups
func (bm *BackupManager) CleanupOldBackups(filePath string) error {
	// Get all backup files sorted by modification time (oldest first)
	backupFiles, err := bm.ListBackups(filePath)
	if err != nil {
		return err
	}

	// Calculate how many backups to remove
	numToRemove := len(backupFiles) - bm.MaxBackups
	if numToRemove <= 0 {
		return nil // No old backups to remove
	}

	// Remove old backups (from the beginning of the sorted list)
	for _, oldBackup := range backupFiles[:numToRemove] {
		if err := os.Remove(oldBackup); err != nil {
			return fmt.Errorf("failed to remove old backup %s: %w", oldBackup, err)
		}
	}

	return nil
}

// RestoreFromBackup restores the file from a specific backup path
func (bm *BackupManager) RestoreFromBackup(filePath string, backupPath string) error {
	// Validate the backup file path
	pattern := fmt.Sprintf("%s.backup-*", filePath)
	match, err := filepath.Match(pattern, backupPath)
	if err != nil {
		return fmt.Errorf("invalid backup path: %w", err)
	}
	if !match {
		return fmt.Errorf("backup path %s is not a valid backup for %s", backupPath, filePath)
	}

	// Copy the backup to original file
	if err := copyFile(backupPath, filePath); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	// Restore file permissions
	srcInfo, err := os.Stat(backupPath)
	if err != nil {
		return nil // Non-fatal, restore was successful
	}
	return os.Chmod(filePath, srcInfo.Mode())
}

// RestoreFromLatestBackup restores the file from the most recent backup
func (bm *BackupManager) RestoreFromLatestBackup(filePath string) error {
	// Get all backup files sorted by modification time (oldest first)
	backupFiles, err := bm.ListBackups(filePath)
	if err != nil {
		return err
	}

	if len(backupFiles) == 0 {
		return fmt.Errorf("no backup files found for %s", filePath)
	}

	// Get the latest backup (last in sorted list)
	latestBackup := backupFiles[len(backupFiles)-1]

	// Restore from the latest backup
	return bm.RestoreFromBackup(filePath, latestBackup)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Preserve permissions from source file
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// restoreFromBackup restores a file from its most recent backup (legacy wrapper for backward compatibility)
func restoreFromBackup(filePath string) error {
	bm := NewBackupManager(DefaultBackupRetention)
	return bm.RestoreFromLatestBackup(filePath)
}
