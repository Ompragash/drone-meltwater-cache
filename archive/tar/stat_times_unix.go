//go:build !windows

package tar

import (
	"os"
	"syscall"
	"time"
)

// statTimes extracts access time and change time from os.FileInfo on Unix systems.
// Returns zero times and ok=false if the underlying system info is not available.
func statTimes(fi os.FileInfo) (atime, ctime time.Time, ok bool) {
	if sys := fi.Sys(); sys != nil {
		if stat, ok := sys.(*syscall.Stat_t); ok {
			// Different Unix systems have different field names for timespec
			// On macOS: Atimespec, Ctimespec
			// On Linux: Atim, Ctim
			// We'll use the stat.Atimespec/Ctimespec which should work on most systems
			atime = time.Unix(stat.Atimespec.Sec, stat.Atimespec.Nsec)
			ctime = time.Unix(stat.Ctimespec.Sec, stat.Ctimespec.Nsec)
			return atime, ctime, true
		}
	}
	return time.Time{}, time.Time{}, false
}

// getUID extracts the user ID from os.FileInfo on Unix systems.
func getUID(fi os.FileInfo) int {
	if sys := fi.Sys(); sys != nil {
		if stat, ok := sys.(*syscall.Stat_t); ok {
			return int(stat.Uid)
		}
	}
	return 0
}

// getGID extracts the group ID from os.FileInfo on Unix systems.
func getGID(fi os.FileInfo) int {
	if sys := fi.Sys(); sys != nil {
		if stat, ok := sys.(*syscall.Stat_t); ok {
			return int(stat.Gid)
		}
	}
	return 0
}