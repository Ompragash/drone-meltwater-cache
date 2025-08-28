//go:build !windows

package tar

import (
	"archive/tar"
	"os"
	"time"
)

// applyFileMetadata applies metadata to files on Unix systems
func applyFileMetadata(target string, h *tar.Header) {
	// Apply file mode
	_ = os.Chmod(target, os.FileMode(h.Mode))
	
	// Apply timestamps - use AccessTime if available, otherwise use ModTime for atime
	atime := h.ModTime
	if !h.AccessTime.IsZero() {
		atime = h.AccessTime
	}
	_ = os.Chtimes(target, atime, h.ModTime)
	
	// Apply ownership (ignore errors - will fail if not root)
	_ = os.Chown(target, h.Uid, h.Gid)
}

// applySymlinkMetadata applies metadata to symlinks on Unix systems
func applySymlinkMetadata(target string, h *tar.Header) {
	// Apply ownership for symlinks (ignore errors - will fail if not root)
	_ = os.Lchown(target, h.Uid, h.Gid)
}

// applyDirMetadata applies metadata to directories on Unix systems
func applyDirMetadata(target string, mode os.FileMode, atime, mtime time.Time, uid, gid int) {
	// Apply directory mode
	_ = os.Chmod(target, mode)
	
	// Apply timestamps
	_ = os.Chtimes(target, atime, mtime)
	
	// Apply ownership (ignore errors - will fail if not root)
	_ = os.Chown(target, uid, gid)
}