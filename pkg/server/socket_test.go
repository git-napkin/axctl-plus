package server

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestDefaultSocketPathPrefersAXCTLSocket(t *testing.T) {
	want := filepath.Join(t.TempDir(), "custom.sock")
	t.Setenv("AXCTL_SOCKET", want)
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/0")
	if got := DefaultSocketPath(); got != want {
		t.Fatalf("DefaultSocketPath() = %q, want override %q", got, want)
	}
}

func TestDefaultSocketPathUsesRuntimeDir(t *testing.T) {
	t.Setenv("AXCTL_SOCKET", "")
	runtime := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtime)
	want := filepath.Join(runtime, "axctl.sock")
	if got := DefaultSocketPath(); got != want {
		t.Fatalf("DefaultSocketPath() = %q, want %q", got, want)
	}
}

func TestDefaultSocketPathRuntimeUserFallback(t *testing.T) {
	t.Setenv("AXCTL_SOCKET", "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	want := filepath.Join("/run/user", strconv.Itoa(os.Getuid()), "axctl.sock")
	got := DefaultSocketPath()
	if got != want {
		t.Fatalf("DefaultSocketPath() = %q, want %q", got, want)
	}
	if strings.HasPrefix(got, "/tmp/") {
		t.Fatalf("DefaultSocketPath() must not use sticky /tmp: %q", got)
	}
}
