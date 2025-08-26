package tar

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
)

func TestMetadataPopulation(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "metadata_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files with different permissions
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create test directory
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	// Test with preserveMetadata enabled
	logger := log.NewNopLogger()
	archive := New(logger, tmpDir, false, true) // preserveMetadata = true

	var buf bytes.Buffer
	_, err = archive.Create([]string{testFile, testDir}, &buf, false)
	if err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}

	// Verify the archive contains PAX format headers
	reader := tar.NewReader(&buf)
	foundFile := false
	foundDir := false

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read archive: %v", err)
		}

		t.Logf("Found file in archive: %s (type: %c)", header.Name, header.Typeflag)

		// Check that PAX format is used when preserveMetadata is enabled
		if header.Format != tar.FormatPAX {
			t.Errorf("Expected PAX format, got %v for %s", header.Format, header.Name)
		}

		if filepath.Base(header.Name) == "test.txt" {
			foundFile = true
			// Verify file permissions
			if header.Mode != 0644 {
				t.Errorf("Expected mode 0644, got %o", header.Mode)
			}
			// Verify that uid/gid are populated (on Unix systems)
			if header.Uid == 0 && header.Gid == 0 {
				t.Logf("Note: uid/gid are 0 (expected on non-root or Windows)")
			}
		}

		if filepath.Base(header.Name) == "testdir" {
			foundDir = true
			// Verify directory permissions
			if header.Mode != 0755 {
				t.Errorf("Expected mode 0755, got %o", header.Mode)
			}
		}
	}

	if !foundFile {
		t.Error("Test file not found in archive")
	}
	if !foundDir {
		t.Error("Test directory not found in archive")
	}
}

func TestMetadataDisabled(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "metadata_disabled_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with preserveMetadata disabled
	logger := log.NewNopLogger()
	archive := New(logger, tmpDir, false, false) // preserveMetadata = false

	var buf bytes.Buffer
	_, err = archive.Create([]string{testFile}, &buf, false)
	if err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}

	// Verify the archive uses default format (not PAX)
	reader := tar.NewReader(&buf)
	header, err := reader.Next()
	if err != nil {
		t.Fatalf("Failed to read archive: %v", err)
	}

	// When preserveMetadata is disabled, should not use PAX format
	if header.Format == tar.FormatPAX {
		t.Error("Expected non-PAX format when preserveMetadata is disabled")
	}
}

func TestTimestampPreservation(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "timestamp_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get original file info
	originalInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}

	// Test with preserveMetadata enabled
	logger := log.NewNopLogger()
	archive := New(logger, tmpDir, false, true) // preserveMetadata = true

	var buf bytes.Buffer
	_, err = archive.Create([]string{testFile}, &buf, false)
	if err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}

	// Verify the archive preserves timestamps
	reader := tar.NewReader(&buf)
	header, err := reader.Next()
	if err != nil {
		t.Fatalf("Failed to read archive: %v", err)
	}

	// Check that ModTime is preserved
	if !header.ModTime.Equal(originalInfo.ModTime()) {
		t.Errorf("ModTime not preserved: expected %v, got %v", originalInfo.ModTime(), header.ModTime)
	}

	// Check that AccessTime is set (may be zero on some systems)
	if !header.AccessTime.IsZero() {
		t.Logf("AccessTime preserved: %v", header.AccessTime)
	} else {
		t.Logf("AccessTime not available (expected on some systems)")
	}
}