package tar

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestStatTimes(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_file.txt")
	
	// Write some content to the file
	content := []byte("test content for metadata extraction")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Get file info
	fi, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}
	
	// Test the statTimes function
	atime, ctime, ok := statTimes(fi)
	
	// Platform-specific expectations
	if runtime.GOOS == "windows" {
		// On Windows, we expect ok=false since extended timestamps are not supported
		if ok {
			t.Error("Expected ok=false on Windows platform, but got ok=true")
		}
		
		// atime and ctime should be zero values
		if !atime.IsZero() {
			t.Error("Expected zero atime on Windows, but got non-zero value")
		}
		if !ctime.IsZero() {
			t.Error("Expected zero ctime on Windows, but got non-zero value")
		}
		
		t.Log("Windows platform: statTimes correctly returned ok=false")
	} else {
		// On Unix platforms, we expect ok=true and valid timestamps
		if !ok {
			t.Error("Expected ok=true on Unix platform, but got ok=false")
		}
		
		// atime and ctime should be valid timestamps (not zero)
		if atime.IsZero() {
			t.Error("Expected non-zero atime on Unix, but got zero value")
		}
		if ctime.IsZero() {
			t.Error("Expected non-zero ctime on Unix, but got zero value")
		}
		
		// Timestamps should be reasonable (within the last hour and not in the future)
		now := time.Now()
		oneHourAgo := now.Add(-time.Hour)
		
		if atime.Before(oneHourAgo) || atime.After(now.Add(time.Minute)) {
			t.Errorf("atime seems unreasonable: %v (now: %v)", atime, now)
		}
		if ctime.Before(oneHourAgo) || ctime.After(now.Add(time.Minute)) {
			t.Errorf("ctime seems unreasonable: %v (now: %v)", ctime, now)
		}
		
		t.Logf("Unix platform: statTimes returned atime=%v, ctime=%v", atime, ctime)
	}
}

func TestStatTimesWithDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "test_dir")
	
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	
	// Get directory info
	fi, err := os.Stat(testDir)
	if err != nil {
		t.Fatalf("Failed to stat test directory: %v", err)
	}
	
	// Test the statTimes function on directory
	atime, ctime, ok := statTimes(fi)
	
	// Platform-specific expectations (same as file test)
	if runtime.GOOS == "windows" {
		if ok {
			t.Error("Expected ok=false on Windows platform for directory, but got ok=true")
		}
		if !atime.IsZero() || !ctime.IsZero() {
			t.Error("Expected zero timestamps on Windows for directory")
		}
	} else {
		if !ok {
			t.Error("Expected ok=true on Unix platform for directory, but got ok=false")
		}
		if atime.IsZero() || ctime.IsZero() {
			t.Error("Expected non-zero timestamps on Unix for directory")
		}
		
		t.Logf("Unix platform directory: statTimes returned atime=%v, ctime=%v", atime, ctime)
	}
}

func TestStatTimesWithInvalidFileInfo(t *testing.T) {
	// Test with nil FileInfo (should not panic)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("statTimes panicked with nil FileInfo: %v", r)
		}
	}()
	
	// This will likely cause a panic or return ok=false, but shouldn't crash
	// We can't easily test this without creating a mock FileInfo, so we'll skip this test
	// in favor of testing with real file system objects
	t.Skip("Skipping invalid FileInfo test - requires mock objects")
}

// Benchmark the statTimes function to ensure it's performant
func BenchmarkStatTimes(b *testing.B) {
	// Create a temporary file for benchmarking
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "bench_file.txt")
	
	content := []byte("benchmark test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		b.Fatalf("Failed to create benchmark file: %v", err)
	}
	
	fi, err := os.Stat(testFile)
	if err != nil {
		b.Fatalf("Failed to stat benchmark file: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = statTimes(fi)
	}
}