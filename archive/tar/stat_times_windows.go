//go:build windows

package tar

import (
	"os"
	"time"
)

// statTimes extracts access time and change time from os.FileInfo on Windows systems.
// Returns zero times and ok=false since Windows doesn't reliably provide atime/ctime
// in the same way as Unix systems.
func statTimes(fi os.FileInfo) (atime, ctime time.Time, ok bool) {
	// Windows doesn't provide reliable access to atime/ctime through os.FileInfo
	// Return false to indicate these times are not available
	return time.Time{}, time.Time{}, false
}

// getUID extracts the user ID from os.FileInfo on Windows systems.
// Always returns 0 since Windows doesn't use POSIX UIDs.
func getUID(fi os.FileInfo) int {
	return 0
}

// getGID extracts the group ID from os.FileInfo on Windows systems.
// Always returns 0 since Windows doesn't use POSIX GIDs.
func getGID(fi os.FileInfo) int {
	return 0
}