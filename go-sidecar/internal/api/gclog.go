package api

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/oscar-wu_pingcorp/profiler-sidecar/internal/logger"
)

const (
	gcLogDir     = "/opt/gc"
	gcLogPattern = "gc.log*"
)

// copyGCLogs copies all gc.log files from source to destination
// Returns (filescopied, filesFailed)
func copyGCLogs(sourceDir, destDir string) (int, int) {
	startTime := time.Now()
	copied := 0
	failed := 0
	var totalBytes int64

	// Find all gc.log files
	pattern := filepath.Join(sourceDir, gcLogPattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to glob gc.log files")
		return 0, 0
	}

	if len(matches) == 0 {
		logger.Log.WithField("source_dir", sourceDir).Debug("No gc.log files found")
		return 0, 0
	}

	logger.Log.WithField("file_count", len(matches)).Info("Starting gc.log copy")

	// Copy each file
	for _, srcPath := range matches {
		filename := filepath.Base(srcPath)
		destPath := filepath.Join(destDir, filename)

		fileInfo, err := os.Stat(srcPath)
		if err != nil {
			logger.Log.WithError(err).WithField("source", srcPath).Warn("Failed to stat gc.log file")
			failed++
			continue
		}

		// Skip directories
		if fileInfo.IsDir() {
			logger.Log.WithField("source", srcPath).Warn("Skipping directory matching gc.log pattern")
			failed++
			continue
		}

		// Copy file
		if err := copyFile(srcPath, destPath); err != nil {
			logger.Log.WithError(err).
				WithField("source", srcPath).
				WithField("destination", destPath).
				Warn("Failed to copy gc.log file")
			failed++
			continue
		}

		size := fileInfo.Size()
		totalBytes += size
		copied++

		logger.Log.WithField("source", srcPath).
			WithField("destination", destPath).
			WithField("size_bytes", size).
			Info("Successfully copied gc.log file")
	}

	duration := time.Since(startTime)
	logger.Log.WithField("files_copied", copied).
		WithField("files_failed", failed).
		WithField("total_bytes", totalBytes).
		WithField("duration_ms", duration.Milliseconds()).
		Info("GC log copy completed")

	return copied, failed
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// Set file permissions
	if err := destFile.Chmod(0644); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}

// copyGCLogsOnShutdown copies gc.log files using default directories
func copyGCLogsOnShutdown() {
	copyGCLogs(gcLogDir, profileDir)
}
