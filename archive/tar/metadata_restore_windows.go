//go:build windows

package tar

import (
	"archive/tar"
	"os"
	"time"
)

// applyFileMetadata applies metadata to files on Windows systems
func applyFileMetadata(target string, h *tar.Header) {
	// Apply file mode (best effort - Windows has limited support)
	_ = os.Chmod(target, os.FileMode(h.Mode))
	
	// Apply timestamps - use AccessTime if available, otherwise use ModTime for atime
	atime := h.ModTime
	if !h.AccessTime.IsZero() {
		atime = h.AccessTime
	}
	_ = os.Chtimes(target, atime, h.ModTime)
	
	// Skip ownership operations on Windows (no POSIX UID/GID)
}

// applySymlinkMetadata applies metadata to symlinks on Windows systems
func applySymlinkMetadata(target string, h *tar.Header) {
	// Skip ownership operations on Windows (no POSIX UID/GID)
}

// applyDirMetadata applies metadata to directories on Windows systems
func applyDirMetadata(target string, mode os.FileMode, atime, mtime time.Time, uid, gid int) {
	// Apply directory mode (best effort - Windows has limited support)
	_ = os.Chmod(target, mode)
	
	// Apply timestamps
	_ = os.Chtimes(target, atime, mtime)
	
	// Skip ownership operations on Windows (no POSIX UID/GID)
}