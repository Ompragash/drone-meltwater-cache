//go:build windows

package tar

import (
	"os"
	"time"
)

// statTimes extracts access time and change time from os.FileInfo on Windows platforms.
// On Windows, extended timestamp information (atime, ctime) is not reliably available
// through the standard os.FileInfo interface, so this function returns ok=false to
// indicate that extended timestamps are not supported.
// The metadata preservation system will fall back to using ModTime for both atime and mtime.
func statTimes(fi os.FileInfo) (atime, ctime time.Time, ok bool) {
	// On Windows, we don't have reliable access to atime and ctime through
	// the standard os.FileInfo interface. The syscall.Win32FileAttributeData
	// structure might contain this information, but it's not consistently
	// available and requires platform-specific handling that's beyond the
	// scope of this implementation.
	//
	// Return ok=false to indicate that extended timestamps are not available.
	// The calling code will fall back to using ModTime for timestamp restoration.
	return time.Time{}, time.Time{}, false
}