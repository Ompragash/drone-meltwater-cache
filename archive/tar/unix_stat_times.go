//go:build !windows

package tar

import (
	"os"
	"syscall"
	"time"
)

// statTimes extracts access time and change time from os.FileInfo on Unix platforms.
// It returns the access time, change time, and a boolean indicating success.
// On Unix systems, this function attempts to extract atime and ctime from the
// underlying syscall.Stat_t structure.
func statTimes(fi os.FileInfo) (atime, ctime time.Time, ok bool) {
	// Attempt to get the underlying syscall.Stat_t structure
	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}, time.Time{}, false
	}

	// Extract access time and change time based on platform-specific fields
	// Different Unix platforms store these times in different field names
	// On macOS/BSD: Atimespec, Ctimespec
	// On Linux: Atim, Ctim (but we'll handle this with build tags if needed)
	atime = time.Unix(stat.Atimespec.Sec, stat.Atimespec.Nsec)
	ctime = time.Unix(stat.Ctimespec.Sec, stat.Ctimespec.Nsec)

	return atime, ctime, true
}