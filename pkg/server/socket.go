package server

import (
	"os"
	"path/filepath"
	"strconv"
)

// DefaultSocketPath returns the conventional per-user socket path.
// XDG_RUNTIME_DIR (/run/user/<uid>, mode 0700) keeps the socket out of the
// shared /tmp namespace, where a second local account could squat the path
// and either block the daemon or impersonate it. Tests and operators can
// override the location with AXCTL_SOCKET. If XDG_RUNTIME_DIR is unset,
// fall back to /run/user/<uid> rather than sticky-bit /tmp.
func DefaultSocketPath() string {
	if p := os.Getenv("AXCTL_SOCKET"); p != "" {
		return p
	}
	if runtime := os.Getenv("XDG_RUNTIME_DIR"); runtime != "" {
		return filepath.Join(runtime, "axctl.sock")
	}
	return filepath.Join("/run/user", strconv.Itoa(os.Getuid()), "axctl.sock")
}
