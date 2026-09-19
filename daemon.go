package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"axctl/pkg/config"
	"axctl/pkg/ipc"
	"axctl/pkg/ipc/hyprland"
	"axctl/pkg/ipc/mango"
	"axctl/pkg/ipc/niri"
	"axctl/pkg/server"
)

func usage() {
	fmt.Println("Usage: axctl [-c <path>] <command> <action> [args]")
	fmt.Println("\nOptions:")
	fmt.Println("  -c <path>                 Use custom config file path (daemon only)")
	fmt.Println("\nCommands:")
	fmt.Println("  daemon                    Start the IPC daemon")
	fmt.Println("  subscribe                 Stream events from the daemon")
	fmt.Println("\n  window <action> [args]")
	fmt.Println("    list                    List all windows")
	fmt.Println("    active                  Get active window ID")
	fmt.Println("    focus <id>              Focus a window")
	fmt.Println("    focus-dir <l|r|u|d>     Focus in direction")
	fmt.Println("    close [id]              Close a window")
	fmt.Println("    move <dir> [id]         Move window")
	fmt.Println("    resize <w> <h> [id]     Resize window")
	fmt.Println("    toggle-floating [id]    Toggle floating")
	fmt.Println("    fullscreen <0|1> [id]   Set fullscreen")
	fmt.Println("    maximize <0|1> [id]     Set maximized")
	fmt.Println("    pin <0|1> [id]          Pin window")
	fmt.Println("    toggle-group [id]       Toggle window group (Hyprland)")
	fmt.Println("    group-nav <f|b>         Navigate group tabs")
	fmt.Println("    layout-prop <k> <v> [id] Set layout property (Niri/Mango)")
	fmt.Println("    move-pixel <x> <y> [id] Move window exactly by pixel")
	fmt.Println("    move-to-workspace-silent <ws> [id] Move window silently")
	fmt.Println("\n  workspace <action> [args]")
	fmt.Println("    list                    List all workspaces")
	fmt.Println("    active                  Get active workspace")
	fmt.Println("    switch <id>             Switch workspace")
	fmt.Println("    move-to <ws_id> [win_id] Move window to workspace")
	fmt.Println("    toggle-special [name]   Toggle special workspace")
	fmt.Println("\n  monitor <action> [args]")
	fmt.Println("    list                    List all monitors")
	fmt.Println("    focus <id>              Focus monitor")
	fmt.Println("    move-to <mon_id> [win_id] Move window to monitor")
	fmt.Println("    set-dpms <mon_id> <0|1> Set DPMS on/off")
	fmt.Println("\n  layout <action> [args]")
	fmt.Println("    set <name>              Set layout")
	fmt.Println("\n  config <action> [args]")
	fmt.Println("    get <key>               Get config value")
	fmt.Println("    set <key> <value>       Set config key")
	fmt.Println("                            Keys: gaps.inner, gaps.outer, border.width,")
	fmt.Println("                                  border.active_color, border.inactive_color,")
	fmt.Println("                                  opacity.active, opacity.inactive,")
	fmt.Println("                                  blur.enabled, blur.size, blur.passes")
	fmt.Println("    batch <json_string>     Batch apply configs")
	fmt.Println("    apply <json_string>     Apply declarative universal config payload")
	fmt.Println("    raw-batch <command>     Send raw compositor batch command")
	fmt.Println("    get-animations          Get animation configs")
	fmt.Println("    bind-key <mods> <key> <cmd> Bind a key")
	fmt.Println("    unbind-key <mods> <key> Unbind a key")
	fmt.Println("    keybinds-batch <json>   Batch bind/unbind keys (structured JSON)")
	fmt.Println("    reload                  Reload config")
	fmt.Println("\n  system <action> [args]")
	fmt.Println("    execute <cmd>           Execute command")
	fmt.Println("    get-cursor-position     Get absolute cursor position")
	fmt.Println("    move-cursor <x> <y>     Move the compositor cursor")
	fmt.Println("    send-shortcut <mods> <key> [window] Send a compositor shortcut")
	fmt.Println("    switch-keyboard-layout [next|prev] Switch keyboard layout")
	fmt.Println("    set-keyboard-layouts <layouts> [variants] Set keyboard layouts (e.g. \"us,es\" \"altgr-intl,\")")
	fmt.Println("    idle-inhibit <0|1>      Inhibit or allow idle/sleep")
	fmt.Println("    idle-wait <ms>          Block until system is idle for <ms> milliseconds (honors inhibitors)")
	fmt.Println("    resume-wait <ms>        Block until system resumes from <ms> milliseconds of idle")
	fmt.Println("    is-idle <ms>            Check if system is currently idle for <ms> milliseconds (returns true/false)")
	fmt.Println("    input-idle-wait <ms>    Block until physical input is idle (ignores inhibitors)")
	fmt.Println("    input-resume-wait <ms>  Block until physical input resumes")
	fmt.Println("    is-input-idle <ms>      Check if physical input is idle (returns true/false)")
	fmt.Println("    is-inhibited            Check if axctl has idle inhibited (returns true/false)")
	fmt.Println("    idle-monitor-create [timeout_ms] [respect_inhibitors 0|1] [enabled 0|1] Create an idle monitor")
	fmt.Println("    idle-monitor-update <id> [timeout_ms] [respect_inhibitors 0|1] [enabled 0|1] Update an idle monitor")
	fmt.Println("    idle-monitor-get <id>   Get idle monitor state")
	fmt.Println("    idle-monitor-destroy <id> Destroy an idle monitor")
	fmt.Println("    idle-inhibitor-create [enabled 0|1] Create an idle inhibitor")
	fmt.Println("    idle-inhibitor-set <id> <0|1> Enable/disable an idle inhibitor")
	fmt.Println("    idle-inhibitor-get <id> Get idle inhibitor state")
	fmt.Println("    idle-inhibitor-destroy <id> Destroy an idle inhibitor")
	fmt.Println("    inhibit-system <0|1>    Enable or disable system-wide idle inhibition (systemd)")
	fmt.Println("    is-system-inhibited   Check if system-wide idle inhibition is active")
	fmt.Println("    app-inhibit-check [patterns...] Check if apps are inhibiting idle (default: vlc,mpv,firefox,chromium,brave,steam)")
	fmt.Println("    media-inhibit-check   Check for active audio/media (PulseAudio/PipeWire)")
	fmt.Println("    get-capabilities        Get compositor capabilities")
	fmt.Println("    exit                    Exit compositor")
	fmt.Println("\n  darkmode <action>")
	fmt.Println("    on                      Set system color-scheme to prefer-dark")
	fmt.Println("    off                     Set system color-scheme to prefer-light")
	fmt.Println("    toggle                  Toggle dark/light color-scheme")
	fmt.Println("    status                  Show current color-scheme state")
}

func daemonSocketPath() string {
	return server.DefaultSocketPath()
}

func detectCompositor() (ipc.Compositor, error) {
	if c, err := hyprland.New(); err == nil {
		return c, nil
	}
	if c, err := niri.New(); err == nil {
		return c, nil
	}
	if c, err := mango.New(); err == nil {
		return c, nil
	}
	return nil, fmt.Errorf("no supported compositor detected")
}

func runDaemon(customConfigPath string) {
	comp, err := detectCompositor()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	fmt.Printf("Detected compositor: %T\n", comp)

	socketPath := daemonSocketPath()
	if conn, err := net.Dial("unix", socketPath); err == nil {
		conn.Close()
		fmt.Println("Error: axctl daemon is already running.")
		os.Exit(1)
	}
	os.Remove(socketPath)

	srv := server.New(comp, socketPath)
	fmt.Printf("Starting axctl daemon on %s\n", socketPath)

	var cfgWatcher *config.ConfigWatcher
	configPath := customConfigPath
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	reloadFromToml := func() {
		cfg, loadErr := config.LoadConfig(configPath)
		if loadErr != nil {
			fmt.Printf("[axctl-config] Error reloading config: %v\n", loadErr)
			return
		}
		if applyErr := config.ApplyConfig(cfg, comp); applyErr != nil {
			fmt.Printf("[axctl-config] Error applying config: %v\n", applyErr)
		}
	}

	srv.ConfigReloader = reloadFromToml

	if _, statErr := os.Stat(configPath); statErr == nil {
		cfg, cfgErr := config.LoadConfig(configPath)
		if cfgErr != nil {
			fmt.Printf("[axctl-config] Error loading config: %v\n", cfgErr)
		} else {
			fmt.Printf("[axctl-config] Loaded config from %s\n", configPath)
			if applyErr := config.ApplyConfig(cfg, comp); applyErr != nil {
				fmt.Printf("[axctl-config] Error applying config: %v\n", applyErr)
			}
		}

		watcher, watchErr := config.NewConfigWatcher()
		if watchErr != nil {
			fmt.Printf("[axctl-config] Warning: could not start watcher: %v\n", watchErr)
		} else {
			watcher.Start(configPath, func(newCfg *config.TOMLConfig) {
				fmt.Println("[axctl-config] Config changed, reloading...")
				if applyErr := config.ApplyConfig(newCfg, comp); applyErr != nil {
					fmt.Printf("[axctl-config] Error applying config: %v\n", applyErr)
				}
			})
			cfgWatcher = watcher
		}
	} else {
		fmt.Printf("[axctl-config] No config file at %s, skipping\n", configPath)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-sig
	if cfgWatcher != nil {
		cfgWatcher.Stop()
	}
	os.Remove(socketPath)
}

func runSubscribe() {
	socketPath := daemonSocketPath()
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Printf("Error connecting to daemon: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	req := map[string]interface{}{
		"id":     1,
		"method": "System.Subscribe",
		"params": map[string]interface{}{},
	}
	json.NewEncoder(conn).Encode(req)

	dec := json.NewDecoder(conn)
	for {
		var msg json.RawMessage
		if err := dec.Decode(&msg); err != nil {
			fmt.Printf("Connection closed or error: %v\n", err)
			os.Exit(1)
		}
		var notif struct {
			JSONRPC string `json:"jsonrpc"`
		}
		if err := json.Unmarshal(msg, &notif); err == nil && notif.JSONRPC == "2.0" {
			fmt.Println(string(msg))
		}
	}
}
