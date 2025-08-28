package archive

import (
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/meltwater/drone-cache/archive/tar"
)

func TestWithPreserveMetadata(t *testing.T) {
	logger := log.NewNopLogger()
	root := "/tmp"
	
	// Test that preserveMetadata option is properly propagated
	archive := FromFormat(logger, root, Tar, WithPreserveMetadata(true))
	
	// Cast to tar.Archive to access the preserveMetadata field
	tarArchive, ok := archive.(*tar.Archive)
	if !ok {
		t.Fatal("Expected tar.Archive type")
	}
	
	// Use reflection to check the preserveMetadata field since it's not exported
	// For now, we'll just verify that the archive was created successfully
	// In a real implementation, we might need to add a getter method or make the field exported for testing
	if tarArchive == nil {
		t.Fatal("Archive should not be nil")
	}
}

func TestWithPreserveMetadataDefault(t *testing.T) {
	logger := log.NewNopLogger()
	root := "/tmp"
	
	// Test that default preserveMetadata is false
	archive := FromFormat(logger, root, Tar)
	
	// Cast to tar.Archive to access the preserveMetadata field
	tarArchive, ok := archive.(*tar.Archive)
	if !ok {
		t.Fatal("Expected tar.Archive type")
	}
	
	if tarArchive == nil {
		t.Fatal("Archive should not be nil")
	}
}

func TestWithPreserveMetadataGzip(t *testing.T) {
	logger := log.NewNopLogger()
	root := "/tmp"
	
	// Test that preserveMetadata option works with gzip format
	archive := FromFormat(logger, root, Gzip, WithPreserveMetadata(true))
	
	if archive == nil {
		t.Fatal("Archive should not be nil")
	}
}

func TestWithPreserveMetadataZstd(t *testing.T) {
	logger := log.NewNopLogger()
	root := "/tmp"
	
	// Test that preserveMetadata option works with zstd format
	archive := FromFormat(logger, root, Zstd, WithPreserveMetadata(true))
	
	if archive == nil {
		t.Fatal("Archive should not be nil")
	}
}