package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
)

func handleRPC(category string, args []string) {
	action := args[0]
	method := fmt.Sprintf("%s.%s", capitalize(category), capitalize(action))

	params := make(map[string]interface{})

	switch method {
	case "Window.Focus":
		if len(args) > 1 {
			params["id"] = args[1]
		}
	case "Window.FocusDir":
		if len(args) > 1 {
			params["direction"] = args[1]
		}
	case "Window.Close":
		if len(args) > 1 {
			params["id"] = args[1]
		}
	case "Window.Move":
		if len(args) > 1 {
			params["direction"] = args[1]
		}
		if len(args) > 2 {
			params["id"] = args[2]
		}
	case "Window.Resize":
		if len(args) > 2 {
			params["width"] = parseInt(args[1])
			params["height"] = parseInt(args[2])
		}
		if len(args) > 3 {
			params["id"] = args[3]
		}
	case "Window.ToggleFloating":
		if len(args) > 1 {
			params["id"] = args[1]
		}
	case "Window.Fullscreen":
		if len(args) > 1 {
			params["state"] = args[1] == "1"
		}
		if len(args) > 2 {
			params["id"] = args[2]
		}
	case "Window.Maximize":
		if len(args) > 1 {
			params["state"] = args[1] == "1"
		}
		if len(args) > 2 {
			params["id"] = args[2]
		}
	case "Window.Pin":
		if len(args) > 1 {
			params["state"] = args[1] == "1"
		}
		if len(args) > 2 {
			params["id"] = args[2]
		}
	case "Window.ToggleGroup":
		if len(args) > 1 {
			params["id"] = args[1]
		}
	case "Window.GroupNav":
		if len(args) > 1 {
			params["direction"] = args[1]
		}
	case "Window.LayoutProp":
		if len(args) > 2 {
			params["key"] = args[1]
			params["value"] = args[2]
		}
		if len(args) > 3 {
			params["id"] = args[3]
		}
	case "Window.MovePixel":
		if len(args) > 2 {
			params["x"] = parseInt(args[1])
			params["y"] = parseInt(args[2])
		}
		if len(args) > 3 {
			params["id"] = args[3]
		}
	case "Window.MoveToWorkspaceSilent":
		if len(args) > 1 {
			params["workspace_id"] = args[1]
		}
		if len(args) > 2 {
			params["window_id"] = args[2]
		}
	case "Workspace.Switch":
		if len(args) > 1 {
			params["id"] = args[1]
		}
	case "Workspace.MoveTo":
		if len(args) > 1 {
			params["workspace_id"] = args[1]
		}
		if len(args) > 2 {
			params["window_id"] = args[2]
		}
	case "Workspace.ToggleSpecial":
		if len(args) > 1 {
			params["name"] = args[1]
		}
	case "Monitor.Focus":
		if len(args) > 1 {
			params["id"] = args[1]
		}
	case "Monitor.MoveTo":
		if len(args) > 1 {
			params["monitor_id"] = args[1]
		}
		if len(args) > 2 {
			params["window_id"] = args[2]
		}
	case "Monitor.SetDpms":
		if len(args) > 1 {
			params["monitor_id"] = args[1]
		}
		if len(args) > 2 {
			params["on"] = args[2] == "1"
		}
	case "Layout.Set":
		if len(args) > 1 {
			params["name"] = args[1]
		}
	case "Config.Get":
		if len(args) > 1 {
			params["key"] = args[1]
		}
	case "Config.Set":
		if len(args) > 2 {
			params["key"] = args[1]
			params["value"] = args[2]
		}
	case "Config.Apply":
		if len(args) > 1 {
			params["payload"] = args[1]
		}
	case "Config.RawBatch":
		if len(args) > 1 {
			params["command"] = args[1]
		}
	case "Config.Batch":
		if len(args) > 1 {
			var configs map[string]interface{}
			if err := json.Unmarshal([]byte(args[1]), &configs); err == nil {
				params["configs"] = configs
			} else {
				fmt.Printf("Error parsing JSON: %v\n", err)
				return
			}
		}
	case "Config.BindKey":
		if len(args) > 3 {
			params["mods"] = args[1]
			params["key"] = args[2]
			params["command"] = args[3]
		}
	case "Config.UnbindKey":
		if len(args) > 2 {
			params["mods"] = args[1]
			params["key"] = args[2]
		}
	case "Config.KeybindsBatch":
		if len(args) > 1 {
			params["payload"] = args[1]
		}
	case "System.Execute":
		if len(args) > 1 {
			params["command"] = args[1]
		}
	case "System.MoveCursor":
		if len(args) > 2 {
			params["x"] = parseInt(args[1])
			params["y"] = parseInt(args[2])
		}
	case "System.SendShortcut":
		if len(args) > 1 {
			params["mods"] = args[1]
		}
		if len(args) > 2 {
			params["key"] = args[2]
		}
		if len(args) > 3 {
			params["window"] = args[3]
		}
	case "System.SwitchKeyboardLayout":
		if len(args) > 1 {
			params["action"] = args[1]
		} else {
			params["action"] = "next"
		}
	case "System.SetKeyboardLayouts":
		if len(args) > 1 {
			params["layouts"] = args[1]
		}
		if len(args) > 2 {
			params["variants"] = args[2]
		}
	case "System.IdleInhibit":
		if len(args) > 1 {
			params["on"] = args[1] == "1"
		}
	case "System.IdleWait", "System.ResumeWait", "System.IsIdle", "System.InputIdleWait", "System.InputResumeWait", "System.IsInputIdle":
		if len(args) > 1 {
			params["timeout_ms"] = parseInt(args[1])
		}
	case "System.IdleMonitorCreate":
		if len(args) > 1 {
			params["timeout_ms"] = parseInt(args[1])
		}
		if len(args) > 2 {
			params["respect_inhibitors"] = args[2] == "1"
		}
		if len(args) > 3 {
			params["enabled"] = args[3] == "1"
		}
	case "System.IdleMonitorUpdate":
		if len(args) > 1 {
			params["id"] = parseInt(args[1])
		}
		if len(args) > 2 {
			params["timeout_ms"] = parseInt(args[2])
		}
		if len(args) > 3 {
			params["respect_inhibitors"] = args[3] == "1"
		}
		if len(args) > 4 {
			params["enabled"] = args[4] == "1"
		}
	case "System.IdleMonitorGet", "System.IdleMonitorDestroy":
		if len(args) > 1 {
			params["id"] = parseInt(args[1])
		}
	case "System.IdleInhibitorCreate":
		if len(args) > 1 {
			params["enabled"] = args[1] == "1"
		}
	case "System.IdleInhibitorSet":
		if len(args) > 2 {
			params["id"] = parseInt(args[1])
			params["enabled"] = args[2] == "1"
		}
	case "System.IdleInhibitorGet", "System.IdleInhibitorDestroy":
		if len(args) > 1 {
			params["id"] = parseInt(args[1])
		}
	case "System.InhibitSystem":
		if len(args) > 1 {
			params["on"] = args[1] == "1"
		}
	case "System.AppInhibitCheck":
		if len(args) > 1 {
			params["patterns"] = args[1:]
		}
	}

	socketPath := daemonSocketPath()
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Printf("Error connecting to daemon: %v\n", err)
		return
	}
	defer conn.Close()

	req := map[string]interface{}{
		"id":     1,
		"method": method,
		"params": params,
	}
	json.NewEncoder(conn).Encode(req)

	var resp struct {
		Result interface{} `json:"result"`
		Error  string      `json:"error"`
	}
	json.NewDecoder(conn).Decode(&resp)

	if resp.Error != "" {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	if s, ok := resp.Result.(string); ok && s == "ok" {
		fmt.Println("Success")
		return
	}

	if method == "System.GetCursorPosition" && printCursorXY(resp.Result) {
		return
	}

	out, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println(string(out))
}

func printCursorXY(result interface{}) bool {
	m, ok := result.(map[string]interface{})
	if !ok {
		return false
	}
	x, okX := jsonNumber(m["x"])
	y, okY := jsonNumber(m["y"])
	if !okX || !okY {
		return false
	}
	fmt.Printf("%d,%d\n", x, y)
	return true
}

func jsonNumber(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}

func parseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}
