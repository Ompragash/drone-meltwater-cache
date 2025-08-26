package tar

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
)

func TestPreserveMetadata_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "tar_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files with specific metadata
	testFile := filepath.Join(tmpDir, "testfile.txt")
	testDir := filepath.Join(tmpDir, "testdir")

	// Create test file
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create test directory
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	// Modify file times for testing
	pastTime := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(testFile, pastTime, pastTime); err != nil {
		t.Fatalf("Failed to set file times: %v", err)
	}
	if err := os.Chtimes(testDir, pastTime, pastTime); err != nil {
		t.Fatalf("Failed to set dir times: %v", err)
	}

	logger := log.NewNopLogger()

	t.Run("WithPreserveMetadata", func(t *testing.T) {
		archive := NewWithOptions(logger, tmpDir, false, true)
		var buf bytes.Buffer

		// Create archive with metadata preservation
		_, err := archive.Create([]string{testFile, testDir}, &buf, false)
		if err != nil {
			t.Fatalf("Failed to create archive: %v", err)
		}

		// Verify the archive contains PAX headers
		tr := tar.NewReader(&buf)
		foundFile := false
		foundDir := false

		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Failed to read archive: %v", err)
			}

			t.Logf("Found header: %s, Format: %v", header.Name, header.Format)
			
			if header.Format == tar.FormatPAX {
				if filepath.Base(header.Name) == "testfile.txt" {
					foundFile = true
					// Verify metadata is preserved
					if header.AccessTime.IsZero() {
						t.Error("Expected AccessTime to be set in PAX format")
					}
				} else if filepath.Base(header.Name) == "testdir" {
					foundDir = true
					if header.AccessTime.IsZero() {
						t.Error("Expected AccessTime to be set in PAX format for directory")
					}
				}
			}
		}

		if !foundFile {
			t.Error("Expected to find test file in archive")
		}
		if !foundDir {
			t.Error("Expected to find test directory in archive")
		}
	})

	t.Run("WithoutPreserveMetadata", func(t *testing.T) {
		archive := NewWithOptions(logger, tmpDir, false, false)
		var buf bytes.Buffer

		// Create archive without metadata preservation
		_, err := archive.Create([]string{testFile, testDir}, &buf, false)
		if err != nil {
			t.Fatalf("Failed to create archive: %v", err)
		}

		// Verify the archive uses standard format
		tr := tar.NewReader(&buf)
		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Failed to read archive: %v", err)
			}

			// Should not use PAX format when preserve metadata is disabled
			if header.Format == tar.FormatPAX {
				t.Errorf("Did not expect PAX format when preserve metadata is disabled, got format: %v", header.Format)
			}
		}
	})
}

func TestPreserveMetadata_Extract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	logger := log.NewNopLogger()

	t.Run("ExtractWithPreserveMetadata", func(t *testing.T) {
		// Test that archives created with preserve metadata can be extracted
		// without errors, which tests the core functionality
		
		// This is a simple test that verifies the metadata preservation code
		// doesn't break the archive creation and extraction process
		
		tmpDir, err := os.MkdirTemp("", "tar_preserve_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create a minimal file structure
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create archive with preserve metadata enabled
		archive := NewWithOptions(logger, tmpDir, false, true)
		var buf bytes.Buffer
		
		written, err := archive.Create([]string{testFile}, &buf, false)
		if err != nil {
			t.Fatalf("Failed to create archive with preserve metadata: %v", err)
		}
		
		if written == 0 {
			t.Error("Expected to write some bytes to archive")
		}

		// Basic test that preserve metadata doesn't break extraction
		extractDir, err := os.MkdirTemp("", "tar_extract_test")
		if err != nil {
			t.Fatalf("Failed to create extract dir: %v", err)
		}
		defer os.RemoveAll(extractDir)

		bufReader := bytes.NewReader(buf.Bytes())
		extracted, err := archive.Extract(extractDir, bufReader)
		if err != nil {
			t.Fatalf("Failed to extract archive with preserve metadata: %v", err)
		}
		
		if extracted == 0 {
			t.Error("Expected to extract some bytes from archive")
		}

		t.Logf("Archive creation and extraction with preserve metadata successful")
	})
}

func TestPreserveMetadata_BackwardCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	logger := log.NewNopLogger()

	t.Run("ExtractOldArchiveWithPreserveEnabled", func(t *testing.T) {
		// Test that archives created without preserve metadata can be extracted
		// with preserve metadata enabled (backward compatibility)
		
		tmpDir, err := os.MkdirTemp("", "tar_compat_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create an archive without metadata preservation (simulating old archive)
		testFile := filepath.Join(tmpDir, "oldfile.txt")
		if err := os.WriteFile(testFile, []byte("old content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		archiveCreate := NewWithOptions(logger, tmpDir, false, false) // no preserve metadata
		var buf bytes.Buffer
		written, err := archiveCreate.Create([]string{testFile}, &buf, false)
		if err != nil {
			t.Fatalf("Failed to create archive: %v", err)
		}
		
		if written == 0 {
			t.Error("Expected to write some bytes to archive")
		}

		// Extract with metadata preservation enabled (should not fail)
		extractDir, err := os.MkdirTemp("", "tar_compat_extract")
		if err != nil {
			t.Fatalf("Failed to create extract dir: %v", err)
		}
		defer os.RemoveAll(extractDir)

		archiveExtract := NewWithOptions(logger, extractDir, false, true) // with preserve metadata
		bufReader := bytes.NewReader(buf.Bytes())
		extracted, err := archiveExtract.Extract(extractDir, bufReader)
		if err != nil {
			t.Fatalf("Failed to extract old archive with preserve metadata enabled: %v", err)
		}
		
		if extracted == 0 {
			t.Error("Expected to extract some bytes from backward compatible archive")
		}

		t.Logf("Backward compatibility test successful")
	})
}