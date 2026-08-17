package hyprland

import (
	"strings"
	"testing"

	"axctl/pkg/ipc"
)

func TestGenerateKeybindsLuaExecFlags(t *testing.T) {
	gen := &LuaGenerator{}
	cfg := ipc.ConfigKeybinds{
		Ambxst: &ipc.AmbxstKeybinds{
			Binds: map[string]ipc.Keybind{
				"launcher": {Modifiers: []string{"SUPER"}, Key: "Super_R", Dispatcher: "exec", Argument: "ambxst+ run launcher", Flags: "e", Enabled: true},
				"volume":   {Modifiers: []string{""}, Key: "XF86AudioRaiseVolume", Dispatcher: "exec", Argument: "ambxst+ run volume-up", Flags: "lr", Enabled: true},
				"plain":    {Modifiers: []string{"SUPER"}, Key: "Q", Dispatcher: "window.close", Enabled: true},
				"mouse":    {Modifiers: []string{"SUPER"}, Key: "mouse:272", Dispatcher: "exec", Argument: "ambxst+ run drag", Flags: "", Enabled: true},
			},
		},
	}
	out := gen.GenerateKeybindsLua(cfg)

	if !strings.Contains(out, `hl.bind("SUPER + Super_R", hl.dsp.exec_cmd("ambxst+ run launcher"), { release = true })`) {
		t.Errorf("release flag missing on exec bind:\n%s", out)
	}
	if !strings.Contains(out, `{ locked = true, repeating = true })`) {
		t.Errorf("locked+repeat flags missing on exec bind:\n%s", out)
	}
	if strings.Contains(out, `hl.bind("SUPER + Q", hl.dsp.window.close(), {`) {
		t.Errorf("non-exec bind gained spurious opts:\n%s", out)
	}
	if !strings.Contains(out, `{ mouse = true })`) {
		t.Errorf("mouse flag missing on exec mouse bind:\n%s", out)
	}
}

func TestGenerateKeybindsHyprlangFlags(t *testing.T) {
	gen := &Generator{}
	cfg := ipc.ConfigKeybinds{
		Custom: []ipc.Keybind{
			{Modifiers: []string{"SUPER"}, Key: "D", Dispatcher: "exec", Argument: "ambxst+ run dashboard", Flags: "e", Enabled: true},
			{Modifiers: []string{"SUPER"}, Key: "L", Dispatcher: "exec", Argument: "ambxst+ run volume-up", Flags: "lr", Enabled: true},
		},
	}
	out := gen.GenerateKeybinds(cfg)

	if !strings.Contains(out, "binde = SUPER, D, exec, ambxst+ run dashboard") {
		t.Errorf("release keyword missing:\n%s", out)
	}
	if !strings.Contains(out, "bindlr = SUPER, L, exec, ambxst+ run volume-up") {
		t.Errorf("locked+repeat keyword missing:\n%s", out)
	}
}