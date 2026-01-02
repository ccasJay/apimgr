package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// AtomicFileUpdate ensures atomic file update to prevent data corruption
func AtomicFileUpdate(filePath string, newContent string, createBackup bool) error {
	// Create backup if requested
	if createBackup {
		bm := NewBackupManager(DefaultBackupRetention)
		if _, err := bm.CreateBackup(filePath); err != nil {
			return fmt.Errorf("failed to create backup file: %w", err)
		}
	}

	// Create temporary file in the same directory
	tmpFile, err := os.CreateTemp(filepath.Dir(filePath), "settings.json.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up on failure

	// Write new content to temporary file
	if _, err := tmpFile.WriteString(newContent); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}
	tmpFile.Close()

	// Change file permissions to match existing file (0600)
	if err := os.Chmod(tmpFile.Name(), 0600); err != nil {
		return fmt.Errorf("failed to set permissions on temporary file: %w", err)
	}

	// Atomic rename - this is guaranteed to be atomic on all POSIX systems
	if err := os.Rename(tmpFile.Name(), filePath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	// Cleanup old backups after successful update
	if createBackup {
		bm := NewBackupManager(DefaultBackupRetention)
		if err := bm.CleanupOldBackups(filePath); err != nil {
			// Non-fatal error, update was successful
			// fmt.Printf("⚠️  Failed to cleanup old backups: %v\n", err)
		}
	}

	return nil
}

// MigrateConfig migrates configuration from old path to new path
func MigrateConfig(oldPath, newPath string) error {
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("failed to read old config file: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("old config file is empty")
	}

	// Validate that it's a valid JSON
	var temp interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("old config file format is invalid: %w", err)
	}

	// Write to new location with locking
	file, err := os.OpenFile(newPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open new config file: %w", err)
	}

	// Lock the new config file exclusively
	// Note: lockFileExclusive is only available inside config package
	// For simplicity, we'll skip locking during migration
	// since this is only called once when initializing the config manager

	// Write data while holding the lock
	_, err = file.Write(data)
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to write new config file: %w", err)
	}

	// Ensure data is flushed
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("failed to sync new config file to disk: %w", err)
	}

	// Unlock and close - skipped since we didn't lock
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close new config file: %w", err)
	}

	// Backup old config
	backupPath := oldPath + ".backup"
	if err := os.Rename(oldPath, backupPath); err != nil {
		// Don't fail migration if backup fails
		fmt.Printf("⚠️  Failed to create backup of old config: %v\n", err)
	}

	return nil
}

// ShouldMigrateConfig checks if config migration should be performed
func ShouldMigrateConfig(oldPath, newPath string) bool {
	// Migrate if old config exists and new config doesn't
	oldExists := FileExists(oldPath)
	newExists := FileExists(newPath)
	return oldExists && !newExists
}
