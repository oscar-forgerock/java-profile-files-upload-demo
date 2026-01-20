package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oscar-wu_pingcorp/profiler-sidecar/internal/logger"
)

func init() {
	// Initialize logger for tests
	logger.Init()
}

func TestCopyGCLogs_Success(t *testing.T) {
	// Setup temp directories
	gcDir := t.TempDir()
	jfrDir := t.TempDir()

	// Create sample gc.log files
	files := []string{"gc.log", "gc.log.0", "gc.log.1"}
	for _, file := range files {
		path := filepath.Join(gcDir, file)
		content := []byte("test gc log content for " + file)
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Call function
	copied, failed := copyGCLogs(gcDir, jfrDir)

	// Verify results
	if copied != 3 {
		t.Errorf("Expected 3 files copied, got %d", copied)
	}
	if failed != 0 {
		t.Errorf("Expected 0 failures, got %d", failed)
	}

	// Verify files exist in destination
	for _, file := range files {
		destPath := filepath.Join(jfrDir, file)
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist in destination", file)
		}
	}
}

func TestCopyGCLogs_NoFiles(t *testing.T) {
	gcDir := t.TempDir()
	jfrDir := t.TempDir()

	copied, failed := copyGCLogs(gcDir, jfrDir)

	if copied != 0 {
		t.Errorf("Expected 0 files copied, got %d", copied)
	}
	if failed != 0 {
		t.Errorf("Expected 0 failures, got %d", failed)
	}
}

func TestCopyGCLogs_BestEffort(t *testing.T) {
	gcDir := t.TempDir()
	jfrDir := t.TempDir()

	// Create one valid file
	validFile := filepath.Join(gcDir, "gc.log")
	if err := os.WriteFile(validFile, []byte("valid"), 0644); err != nil {
		t.Fatalf("Failed to create valid file: %v", err)
	}

	// Create a directory that matches pattern (should fail to copy)
	invalidPath := filepath.Join(gcDir, "gc.log.dir")
	if err := os.Mkdir(invalidPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	copied, failed := copyGCLogs(gcDir, jfrDir)

	if copied != 1 {
		t.Errorf("Expected 1 file copied, got %d", copied)
	}
	if failed != 1 {
		t.Errorf("Expected 1 failure (directory), got %d", failed)
	}
}
