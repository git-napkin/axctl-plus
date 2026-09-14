package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPathFallsBackToAmbxstPlus(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdg-config"))

	got := DefaultConfigPath()
	want := filepath.Join(tmp, ".local", "share", "ambxst+", "axctl.toml")
	if got != want {
		t.Fatalf("DefaultConfigPath() = %q, want %q", got, want)
	}
}

func TestDefaultConfigPathUsesLegacyAmbxstIfPlusMissing(t *testing.T) {
	tmp := t.TempDir()
	legacy := filepath.Join(tmp, ".local", "share", "ambxst", "axctl.toml")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("[appearance]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdg-config"))

	got := DefaultConfigPath()
	if got != legacy {
		t.Fatalf("DefaultConfigPath() = %q, want legacy %q", got, legacy)
	}
}

func TestDefaultConfigPathPrefersPlusOverLegacy(t *testing.T) {
	tmp := t.TempDir()
	plus := filepath.Join(tmp, ".local", "share", "ambxst+", "axctl.toml")
	legacy := filepath.Join(tmp, ".local", "share", "ambxst", "axctl.toml")
	for _, p := range []string{plus, legacy} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("[appearance]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdg-config"))

	got := DefaultConfigPath()
	if got != plus {
		t.Fatalf("DefaultConfigPath() = %q, want plus %q", got, plus)
	}
}

func TestDefaultConfigPathPrefersXDGWhenPresent(t *testing.T) {
	tmp := t.TempDir()
	xdg := filepath.Join(tmp, "xdg-config")
	primary := filepath.Join(xdg, "axctl", "config.toml")
	if err := os.MkdirAll(filepath.Dir(primary), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(primary, []byte("[appearance]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got := DefaultConfigPath()
	if got != primary {
		t.Fatalf("DefaultConfigPath() = %q, want %q", got, primary)
	}
}

func TestToIPCConfigPassesWorkspaceStyle(t *testing.T) {
	style := "slidefadevert 20%"
	enabled := true
	cfg := &TOMLConfig{
		Appearance: &AppearanceConfig{
			Animations: &AnimationsConfig{
				Enabled:        &enabled,
				WorkspaceStyle: &style,
			},
		},
	}
	ipcCfg := cfg.ToIPCConfig()
	if ipcCfg.Appearance.Animations == nil {
		t.Fatal("expected animations in IPC config")
	}
	got := ipcCfg.Appearance.Animations.WorkspaceStyle
	if got == nil || *got != style {
		t.Fatalf("WorkspaceStyle = %v, want %q", got, style)
	}
}

func TestToIPCConfigLayerNoScreenShareAlias(t *testing.T) {
	on := true
	cfg := &TOMLConfig{
		LayerRules: []LayerRuleConfig{
			{Namespace: "ambxst+:computer-use", Noscreenshare: &on},
		},
	}
	ipcCfg := cfg.ToIPCConfig()
	if len(ipcCfg.LayerRules) != 1 {
		t.Fatalf("LayerRules len = %d, want 1", len(ipcCfg.LayerRules))
	}
	flag := ipcCfg.LayerRules[0].NoScreenShare
	if flag == nil || !*flag {
		t.Fatalf("NoScreenShare = %v, want true from noscreenshare alias", flag)
	}
}
