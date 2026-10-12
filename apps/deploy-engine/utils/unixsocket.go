package utils

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const unixSocketDialTimeout = 200 * time.Millisecond

// PrepareUnixSocketDir creates the directory that will hold a unix socket, restricted to
// the owning user, and tightens an existing directory to the same.
//
// The directory is the control over who can reach the socket, on both platforms. On unix its
// mode restricts traversal and the socket inherits nothing, so the socket is restricted
// separately. On Windows the socket file inherits the directory's ACL, and bind and connect
// are checked against it, so a restrictive directory is what restricts the socket too.
//
// Go's os.Chmod cannot express an ACL, on Windows it only toggles the read-only attribute
// so the directory, not a mode on the socket, is the portable way to set this.
func PrepareUnixSocketDir(unixSocketPath string) error {
	dir := filepath.Dir(unixSocketPath)

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("error creating directory for unix socket: %w", err)
	}

	// MkdirAll applies the umask, and leaves an existing directory's permissions alone, so
	// neither case is restrictive without setting the mode explicitly.
	if err := os.Chmod(dir, 0o700); err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("error restricting unix socket directory permissions: %w", err)
	}

	return nil
}

// RestrictUnixSocket restricts a unix socket to the owning user on unix, where net.Listen
// would otherwise create it with umask-dependent permissions.
func RestrictUnixSocket(unixSocketPath string) error {
	if err := os.Chmod(unixSocketPath, 0o600); err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("error restricting unix socket permissions: %w", err)
	}

	return nil
}

// ClearStaleUnixSocket removes a socket file left behind by a previous run, so that
// net.Listen does not fail with "address already in use".
//
// A path that is not a socket is left alone and reported, rather than removed.
// A socket another process is still listening on is also left alone,
// so a second engine cannot hijack the path of a running one.
func ClearStaleUnixSocket(unixSocketPath string) error {
	info, err := os.Stat(unixSocketPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("error inspecting unix socket path: %w", err)
	}

	// os.Stat reports ModeSocket for a unix socket on Windows as well, where the socket file
	// is a reparse point the os package knows about, so this check is not unix-only.
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf(
			"path %q exists and is not a socket, refusing to remove it",
			unixSocketPath,
		)
	}

	if conn, err := net.DialTimeout("unix", unixSocketPath, unixSocketDialTimeout); err == nil {
		conn.Close()
		return fmt.Errorf(
			"another deploy engine appears to be listening on %q",
			unixSocketPath,
		)
	}

	if err := os.Remove(unixSocketPath); err != nil {
		return fmt.Errorf("error removing stale unix socket: %w", err)
	}

	return nil
}
