//go:build linux

package capsule

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// bundleDirectory creates a new output directory below a symlink-free parent chain.
// All child paths are opened relative to the final directory descriptor.
type bundleDirectory struct{ fd int }

func openBundleDirectory(path string) (*bundleDirectory, error) {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		absolute, err := filepath.Abs(clean)
		if err != nil {
			return nil, err
		}
		clean = absolute
	}
	parts := strings.Split(strings.TrimPrefix(clean, "/"), "/")
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	for index, part := range parts {
		if part == "" || part == "." || part == ".." {
			unix.Close(fd)
			return nil, fmt.Errorf("unsafe output path")
		}
		if index == len(parts)-1 {
			err = unix.Mkdirat(fd, part, 0o700)
			if err != nil {
				unix.Close(fd)
				return nil, fmt.Errorf("output directory must not already exist: %s", path)
			}
			next, openErr := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
			unix.Close(fd)
			if openErr != nil {
				return nil, openErr
			}
			return &bundleDirectory{fd: next}, nil
		}
		next, openErr := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		unix.Close(fd)
		if openErr != nil {
			return nil, fmt.Errorf("output parent contains an unsafe or missing component %q: %w", part, openErr)
		}
		fd = next
	}
	unix.Close(fd)
	return nil, fmt.Errorf("invalid output path")
}

func (directory *bundleDirectory) close() error { return unix.Close(directory.fd) }
func (directory *bundleDirectory) mkdir(name string) error {
	return unix.Mkdirat(directory.fd, name, 0o700)
}
func (directory *bundleDirectory) writeFile(name string, content []byte) error {
	fd, err := unix.Openat(directory.fd, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	for len(content) > 0 {
		count, writeErr := unix.Write(fd, content)
		if writeErr != nil {
			return writeErr
		}
		content = content[count:]
	}
	return nil
}
func (directory *bundleDirectory) writeLog(content []byte) error {
	logs, err := unix.Openat(directory.fd, "logs", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(logs)
	fd, err := unix.Openat(logs, "failed-job.txt", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	for len(content) > 0 {
		count, writeErr := unix.Write(fd, content)
		if writeErr != nil {
			return writeErr
		}
		content = content[count:]
	}
	return nil
}
