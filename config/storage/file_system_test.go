package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAtomicFileUpdateSuccessPreservesPermissionsAndCreatesBackup(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(filePath, []byte(`{"old":true}`), 0640); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	if err := AtomicFileUpdate(filePath, `{"new":true}`, true); err != nil {
		t.Fatalf("AtomicFileUpdate() error: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != `{"new":true}` {
		t.Fatalf("file content = %q, want updated content", data)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	if got := info.Mode().Perm(); got != 0640 {
		t.Fatalf("file mode = %v, want 0640", got)
	}

	backups, err := NewBackupManager(DefaultBackupRetention).ListBackups(filePath)
	if err != nil {
		t.Fatalf("ListBackups() error: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("backup count = %d, want 1", len(backups))
	}
}

func TestAtomicFileUpdateCreateTempFailureLeavesOriginalIntact(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission behavior is platform-specific")
	}

	dir := t.TempDir()
	filePath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(filePath, []byte(`{"old":true}`), 0600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatalf("Chmod() error: %v", err)
	}
	defer os.Chmod(dir, 0700)

	err := AtomicFileUpdate(filePath, `{"new":true}`, false)
	if err == nil {
		t.Fatal("AtomicFileUpdate() expected error, got nil")
	}

	data, readErr := os.ReadFile(filePath)
	if readErr != nil {
		t.Fatalf("ReadFile() error: %v", readErr)
	}
	if string(data) != `{"old":true}` {
		t.Fatalf("file content = %q, want original content", data)
	}
}
