package tar

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/meltwater/drone-cache/internal"
)

const defaultDirPermission = 0755

var (
	// ErrSourceNotReachable means that given source is not reachable.
	ErrSourceNotReachable = errors.New("source not reachable")
	// ErrArchiveNotReadable means that given archive not readable/corrupted.
	ErrArchiveNotReadable = errors.New("archive not readable")
)

// Archive implements archive for tar.
type Archive struct {
	logger log.Logger

	root            string
	skipSymlinks    bool
	preserveMetadata bool // Add this line
}

// New creates an archive that uses the .tar file format.
func New(logger log.Logger, root string, skipSymlinks bool) *Archive {
	return &Archive{logger, root, skipSymlinks, false} // Add false for preserveMetadata
}

// NewWithPreserveMetadata creates an archive that uses the .tar file format with metadata preservation.
func NewWithPreserveMetadata(logger log.Logger, root string, skipSymlinks bool, preserveMetadata bool) *Archive {
	return &Archive{logger, root, skipSymlinks, preserveMetadata}
}

// Create writes content of the given source to an archive, returns written bytes.
// If isRelativePath is true, it clones using the path, else it clones using a path
// combining archive's root with the path.
func (a *Archive) Create(srcs []string, w io.Writer, isRelativePath bool) (int64, error) {
	tw := tar.NewWriter(w)
	defer internal.CloseWithErrLogf(a.logger, tw, "tar writer")

	var written int64

	for _, src := range srcs {
		_, err := os.Lstat(src)
		if err != nil {
			return written, fmt.Errorf("make sure file or directory readable <%s>: %v,, %w", src, err, ErrSourceNotReachable)
		}

		if err := filepath.Walk(src, writeToArchive(tw, a.root, a.skipSymlinks, &written, isRelativePath, a.logger, a.preserveMetadata)); err != nil {
			return written, fmt.Errorf("walk, add all files to archive, %w", err)
		}
	}

	return written, nil
}

// nolint: lll
func writeToArchive(tw *tar.Writer, root string, skipSymlinks bool, written *int64, isRelativePath bool, logger log.Logger, preserveMetadata bool) func(string, os.FileInfo, error) error {
	return func(path string, fi os.FileInfo, err error) error {
		level.Debug(logger).Log("path", path, "root", root) //nolint: errcheck

		if err != nil {
			return err
		}

		if fi == nil {
			return errors.New("no file info")
		}

		// Create header for Regular files and Directories
		h, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return fmt.Errorf("create header for <%s>, %w", path, err)
		}

		if preserveMetadata {
			h.Format = tar.FormatPAX
			if runtime.GOOS != "windows" {
				if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
					h.Uid = int(stat.Uid)
					h.Gid = int(stat.Gid)
					// For AccessTime and ChangeTime, we'll need to extract them from Stat_t
					// This will require platform-specific code.
					// For now, we'll just use ModTime for both, which is a safe fallback.
					h.AccessTime = fi.ModTime()
					h.ChangeTime = fi.ModTime()
				}
			}
		}

		if fi.Mode()&os.ModeSymlink != 0 { // isSymbolic
			if skipSymlinks {
				return nil
			}

			var err error
			if h, err = createSymlinkHeader(fi, path); err != nil {
				return fmt.Errorf("create header for symbolic link, %w", err)
			}
		}

		var name string
		if filepath.IsAbs(path) {
			name, err = filepath.Abs(path)
		} else if isRelativePath {
			name = path
		} else {
			name, err = relative(root, path)
		}

		if err != nil {
			return fmt.Errorf("relative name <%s>: <%s>, %w", path, root, err)
		}

		h.Name = name

		if err := tw.WriteHeader(h); err != nil {
			return fmt.Errorf("write header for <%s>, %w", path, err)
		}

		if !fi.Mode().IsRegular() {
			return nil
		}

		n, err := writeFileToArchive(tw, path)
		if err != nil {
			return fmt.Errorf("write file to archive, %w", err)
		}

		*written += n
		// Alternatives:
		// *written += h.FileInfo().Size()
		// *written += fi.Size()

		return nil
	}
}

func relative(parent string, path string) (string, error) {
	name := filepath.Base(path)

	rel, err := filepath.Rel(parent, filepath.Dir(path))
	if err != nil {
		return "", fmt.Errorf("relative path <%s>, base <%s>, %w", rel, name, err)
	}

	// NOTICE: filepath.Rel puts "../" when given path is not under parent.
	for strings.HasPrefix(rel, "../") {
		rel = strings.TrimPrefix(rel, "../")
	}

	rel = filepath.ToSlash(rel)

	return strings.TrimPrefix(filepath.Join(rel, name), "/"), nil
}

func createSymlinkHeader(fi os.FileInfo, path string) (*tar.Header, error) {
	lnk, err := os.Readlink(path)
	if err != nil {
		return nil, fmt.Errorf("read link <%s>, %w", path, err)
	}

	h, err := tar.FileInfoHeader(fi, lnk)
	if err != nil {
		return nil, fmt.Errorf("create symlink header for <%s>, %w", path, err)
	}

	return h, nil
}

func writeFileToArchive(tw io.Writer, path string) (n int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open file <%s>, %w", path, err)
	}

	defer internal.CloseWithErrCapturef(&err, f, "write file to archive <%s>", path)

	written, err := io.Copy(tw, f)
	if err != nil {
		return written, fmt.Errorf("copy the file <%s> data to the tarball, %w", path, err)
	}

	return written, nil
}

// Extract reads content from the given archive reader and restores it to the destination, returns written bytes.
func (a *Archive) Extract(dst string, r io.Reader) (int64, error) {
	var (
		written int64
		tr      = tar.NewReader(r)
	)

	// Map to store directory metadata for delayed restoration
	dirMetadata := make(map[string]struct {
		mode os.FileMode
		at   time.Time
		mt   time.Time
		uid  int
		gid  int
	})

	for {
		h, err := tr.Next()

		switch {
		case err == io.EOF: // if no more files are found return
			// Apply metadata to directories in reverse depth order
			if a.preserveMetadata {
				// Sort directories by depth (deepest first)
				dirs := make([]string, 0, len(dirMetadata))
				for dir := range dirMetadata {
					dirs = append(dirs, dir)
				}
				// Simple sort by length of path (deepest first) - this is a heuristic
				// A more robust solution would use filepath.Walk or build a proper tree
				for i := 0; i < len(dirs); i++ {
					for j := i + 1; j < len(dirs); j++ {
						if len(dirs[i]) < len(dirs[j]) {
							dirs[i], dirs[j] = dirs[j], dirs[i]
						}
					}
				}

				for _, dir := range dirs {
					meta := dirMetadata[dir]
					_ = os.Chmod(dir, meta.mode)
					_ = os.Chtimes(dir, meta.at, meta.mt)
					if runtime.GOOS != "windows" {
						_ = os.Chown(dir, meta.uid, meta.gid) // Ignore errors (e.g., EPERM)
					}
				}
			}
			return written, nil
		case err != nil: // return any other error
			return written, fmt.Errorf("tar reader <%v>, %w", err, ErrArchiveNotReadable)
		case h == nil: // if the header is nil, skip it
			continue
		}

		var target string
		if dst == h.Name || filepath.IsAbs(h.Name) {
			target = h.Name
		} else {
			name, err := relative(dst, h.Name)
			if err != nil {
				return 0, fmt.Errorf("relative name, %w", err)
			}

			target = filepath.Join(dst, name)
		}

		level.Debug(a.logger).Log("msg", "extracting archive", "path", target)

		if err := os.MkdirAll(filepath.Dir(target), defaultDirPermission); err != nil {
			return 0, fmt.Errorf("ensure directory <%s>, %w", target, err)
		}

		switch h.Typeflag {
		case tar.TypeDir:
			if a.preserveMetadata {
				// Store metadata for later application
				dirMetadata[target] = struct {
					mode os.FileMode
					at   time.Time
					mt   time.Time
					uid  int
					gid  int
				}{
					mode: os.FileMode(h.Mode),
					at:   h.AccessTime,
					mt:   h.ModTime,
					uid:  h.Uid,
					gid:  h.Gid,
				}
				// Create the directory with default permissions for now
				if err := os.MkdirAll(target, defaultDirPermission); err != nil {
					return written, fmt.Errorf("create directory <%s>, %w", target, err)
				}
			} else {
				if err := extractDir(h, target); err != nil {
					return written, err
				}
			}
			continue
		case tar.TypeReg, tar.TypeRegA, tar.TypeChar, tar.TypeBlock, tar.TypeFifo:
			n, err := extractRegular(h, tr, target, a.preserveMetadata)
			written += n

			if err != nil {
				return written, fmt.Errorf("extract regular file, %w", err)
			}

			continue
		case tar.TypeSymlink:
			if err := extractSymlink(h, target, a.preserveMetadata); err != nil {
				return written, fmt.Errorf("extract symbolic link, %w", err)
			}

			continue
		case tar.TypeLink:
			if err := extractLink(h, target, a.preserveMetadata); err != nil {
				return written, fmt.Errorf("extract link, %w", err)
			}

			continue
		case tar.TypeXGlobalHeader:
			continue
		default:
			return written, fmt.Errorf("extract %s, unknown type flag: %c", target, h.Typeflag)
		}
	}
}

func extractDir(h *tar.Header, target string) error {
	if err := os.MkdirAll(target, os.FileMode(h.Mode)); err != nil {
		return fmt.Errorf("create directory <%s>, %w", target, err)
	}

	return nil
}

func extractRegular(h *tar.Header, tr io.Reader, target string, preserveMetadata bool) (n int64, err error) {
	f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(h.Mode))
	if err != nil {
		return 0, fmt.Errorf("open extracted file for writing <%s>, %w", target, err)
	}

	defer internal.CloseWithErrCapturef(&err, f, "extract regular <%s>", target)

	written, err := io.Copy(f, tr)
	if err != nil {
		return written, fmt.Errorf("copy extracted file for writing <%s>, %w", target, err)
	}

	// Apply metadata if preserveMetadata is enabled
	if preserveMetadata {
		_ = os.Chmod(target, os.FileMode(h.Mode))
		at := h.AccessTime
		if at.IsZero() {
			at = h.ModTime
		}
		_ = os.Chtimes(target, at, h.ModTime)
		if runtime.GOOS != "windows" {
			_ = os.Chown(target, h.Uid, h.Gid) // Ignore errors (e.g., EPERM)
		}
	}

	return written, nil
}

func extractSymlink(h *tar.Header, target string, preserveMetadata bool) error {
	if err := unlink(target); err != nil {
		return fmt.Errorf("unlink <%s>, %w", target, err)
	}

	if err := os.Symlink(h.Linkname, target); err != nil {
		return fmt.Errorf("create symbolic link <%s>, %w", target, err)
	}

	// Apply ownership if preserveMetadata is enabled (Unix only)
	if preserveMetadata && runtime.GOOS != "windows" {
		_ = os.Lchown(target, h.Uid, h.Gid) // Ignore errors (e.g., EPERM)
	}

	return nil
}

func extractLink(h *tar.Header, target string, preserveMetadata bool) error {
	if err := unlink(target); err != nil {
		return fmt.Errorf("unlink <%s>, %w", target, err)
	}

	if err := os.Link(h.Linkname, target); err != nil {
		return fmt.Errorf("create hard link <%s>, %w", h.Linkname, err)
	}

	// Apply metadata if preserveMetadata is enabled
	if preserveMetadata {
		_ = os.Chmod(target, os.FileMode(h.Mode))
		at := h.AccessTime
		if at.IsZero() {
			at = h.ModTime
		}
		_ = os.Chtimes(target, at, h.ModTime)
		if runtime.GOOS != "windows" {
			_ = os.Chown(target, h.Uid, h.Gid) // Ignore errors (e.g., EPERM)
		}
	}

	return nil
}

func unlink(path string) error {
	_, err := os.Lstat(path)
	if err == nil {
		return os.Remove(path)
	}

	return nil
}