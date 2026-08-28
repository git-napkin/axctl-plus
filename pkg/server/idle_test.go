package server

import (
	"testing"

	"axctl/pkg/ipc"
)

func TestMatchAppPatterns(t *testing.T) {
	windows := []ipc.Window{
		{AppID: "firefox"},
		{AppID: "org.mozilla.firefox"},
		{AppID: "kitty"},
		{AppID: ""},
	}

	got := matchAppPatterns(windows, []string{"firefox", "mpv"})
	if !got["firefox"] || !got["org.mozilla.firefox"] {
		t.Fatalf("expected firefox windows, got %#v", got)
	}
	if got["kitty"] {
		t.Fatalf("did not expect kitty: %#v", got)
	}

	if len(matchAppPatterns(windows, []string{"vlc"})) != 0 {
		t.Fatal("expected no vlc matches")
	}
}

func TestAppInhibitorCheckUsesWindows(t *testing.T) {
	windows := []ipc.Window{{AppID: "mpv"}}
	got := AppInhibitorCheck(windows, nil)
	if !got["mpv"] {
		t.Fatalf("default patterns should match mpv, got %#v", got)
	}
}
