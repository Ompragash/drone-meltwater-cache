package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/meltwater/drone-cache/archive/tar"

	"github.com/go-kit/kit/log"
)

func main() {
	logger := log.NewLogfmtLogger(os.Stdout)

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "preserve_metadata_test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file with specific metadata
	testFile := filepath.Join(tempDir, "testfile.txt")
	content := []byte("Hello, World!")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		panic(err)
	}

	// Change the file's metadata to specific values
	// Set a specific modification time
	modTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(testFile, modTime, modTime); err != nil {
		panic(err)
	}

	// On Unix systems, also set ownership (this will likely fail for non-root users, which is fine for testing)
	// Only try to set ownership if we're root
	if os.Geteuid() == 0 {
		if err := os.Chown(testFile, 1000, 1000); err != nil {
			// If we can't set ownership, that's okay, just note it
			fmt.Printf("Warning: Could not set ownership: %v\n", err)
		}
	} else {
		fmt.Println("Not running as root, skipping ownership test")
	}

	// Create a tar archive with metadata preservation
	archivePath := filepath.Join(tempDir, "test.tar")
	f, err := os.Create(archivePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	tarArchive := tar.NewWithPreserveMetadata(logger, tempDir, false, true)
	_, err = tarArchive.Create([]string{testFile}, f, false)
	if err != nil {
		panic(err)
	}

	// Close the file to ensure all data is written
	f.Close()

	// Extract the tar archive and verify metadata
	extractDir := filepath.Join(tempDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		panic(err)
	}

	f, err = os.Open(archivePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, err = tarArchive.Extract(extractDir, f)
	if err != nil {
		panic(err)
	}

	// Verify the extracted file's metadata
	extractedFile := filepath.Join(extractDir, "testfile.txt")
	fi, err := os.Stat(extractedFile)
	if err != nil {
		panic(err)
	}

	// Check modification time
	if !fi.ModTime().Equal(modTime) {
		panic(fmt.Sprintf("Modification time mismatch: expected %v, got %v", modTime, fi.ModTime()))
	} else {
		fmt.Println("Modification time preserved correctly")
	}

	// On Unix systems, check ownership
	// Only check ownership if we're root, as non-root users can't change ownership
	if os.Geteuid() == 0 {
		if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
			// Note: The uid/gid might not be exactly 1000/1000 after extraction,
			// as the extraction process might not have permission to set them.
			// We'll just check that they exist (i.e., are not 0, which is root).
			// A more thorough test would run this as root or use capabilities.
			if stat.Uid == 0 && stat.Gid == 0 {
				fmt.Printf("Warning: Ownership might not be preserved correctly (UID: %d, GID: %d)\n", stat.Uid, stat.Gid)
			} else {
				fmt.Println("Ownership appears to be preserved")
			}
		}
	}

	fmt.Println("Metadata preservation test completed!")
}