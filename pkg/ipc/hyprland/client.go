package hyprland

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"axctl/pkg/ipc"
)

type Hyprland struct {
	signature      string
	mu             sync.Mutex
	versionMu      sync.Mutex
	versionKnown   bool
	useLuaDispatch bool
}

func New() (*Hyprland, error) {
	sig := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")
	if sig == "" {
		return nil, fmt.Errorf("HYPRLAND_INSTANCE_SIGNATURE not set")
	}
	return &Hyprland{signature: sig}, nil
}

func (h *Hyprland) getSocketPath(socketName string) string {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	return fmt.Sprintf("%s/hypr/%s/%s", runtimeDir, h.signature, socketName)
}

func (h *Hyprland) dispatch(cmd string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conn, err := net.Dial("unix", h.getSocketPath(".socket.sock"))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(cmd)); err != nil {
		return "", err
	}

	response, err := io.ReadAll(conn)
	if err != nil {
		return "", err
	}
	resp := string(response)
	trimmed := strings.TrimSpace(resp)
	if strings.HasPrefix(trimmed, "error:") || trimmed == "unknown request" {
		return resp, fmt.Errorf("hyprland rejected request: %s", trimmed)
	}
	return resp, nil
}

func (h *Hyprland) supportsLuaDispatchers() bool {
	h.versionMu.Lock()
	if h.versionKnown {
		useLuaDispatch := h.useLuaDispatch
		h.versionMu.Unlock()
		return useLuaDispatch
	}
	h.versionMu.Unlock()

	resp, err := h.dispatch("j/version")
	if err != nil {
		return false
	}

	version, err := parseHyprlandVersion(resp)
	if err != nil {
		return false
	}

	useLuaDispatch := isHyprlandVersionAtLeast(version, 0, 55)
	h.versionMu.Lock()
	h.useLuaDispatch = useLuaDispatch
	h.versionKnown = true
	h.versionMu.Unlock()
	return useLuaDispatch
}

func (h *Hyprland) dispatchVersioned(legacy, lua string) (string, error) {
	if h.supportsLuaDispatchers() {
		return h.dispatch("dispatch " + lua)
	}
	return h.dispatch("dispatch " + legacy)
}

func parseHyprlandVersion(resp string) (string, error) {
	trimmed := strings.TrimSpace(resp)
	if strings.HasPrefix(trimmed, "{") {
		var data struct {
			Version string `json:"version"`
			Tag     string `json:"tag"`
		}
		if err := json.Unmarshal([]byte(trimmed), &data); err != nil {
			return "", err
		}
		if data.Version != "" {
			return data.Version, nil
		}
		if data.Tag != "" {
			return data.Tag, nil
		}
		return "", fmt.Errorf("hyprland version response did not include version")
	}

	fields := strings.Fields(trimmed)
	for _, field := range fields {
		field = strings.TrimPrefix(field, "v")
		if _, err := parseVersionNumber(field); err == nil {
			return field, nil
		}
	}
	return "", fmt.Errorf("could not parse hyprland version from %q", trimmed)
}

func isHyprlandVersionAtLeast(version string, major, minor int) bool {
	parsed, err := parseVersionNumber(version)
	if err != nil {
		return false
	}
	if parsed.major != major {
		return parsed.major > major
	}
	return parsed.minor >= minor
}

type versionNumber struct {
	major int
	minor int
}

func parseVersionNumber(version string) (versionNumber, error) {
	clean := strings.TrimPrefix(strings.TrimSpace(version), "v")
	parts := strings.Split(clean, ".")
	if len(parts) < 2 {
		return versionNumber{}, fmt.Errorf("invalid version %q", version)
	}

	major, err := strconv.Atoi(versionDigits(parts[0]))
	if err != nil {
		return versionNumber{}, err
	}
	minor, err := strconv.Atoi(versionDigits(parts[1]))
	if err != nil {
		return versionNumber{}, err
	}
	return versionNumber{major: major, minor: minor}, nil
}

func versionDigits(value string) string {
	for i, r := range value {
		if r < '0' || r > '9' {
			return value[:i]
		}
	}
	return value
}

func hyprTarget(id string) string {
	if id == "" {
		return ""
	}
	return "address:" + id
}

func luaTargetField(id string) string {
	if id == "" {
		return ""
	}
	return fmt.Sprintf(", window = %q", hyprTarget(id))
}

func luaDirection(direction string) string {
	switch direction {
	case "l":
		return "left"
	case "r":
		return "right"
	case "u":
		return "up"
	case "d":
		return "down"
	default:
		return direction
	}
}

func (h *Hyprland) ListWindows() ([]ipc.Window, error) {
	resp, err := h.dispatch("j/clients")
	if err != nil {
		return nil, err
	}

	if resp == "" || resp == "[]" {
		return []ipc.Window{}, nil
	}

	var clients []struct {
		Address    string `json:"address"`
		Title      string `json:"title"`
		Class      string `json:"class"`
		Floating   bool   `json:"floating"`
		Fullscreen int    `json:"fullscreen"`
		Pinned     bool   `json:"pinned"`
		Monitor    int    `json:"monitor"`
		Urgent     bool   `json:"urgent"`
		At         []int  `json:"at"`
		Size       []int  `json:"size"`
		Workspace  struct {
			ID int `json:"id"`
		} `json:"workspace"`
	}

	if err := json.Unmarshal([]byte(resp), &clients); err != nil {
		fmt.Printf("[Hyprland] Unmarshal error: %v | Raw: %s\n", err, resp)
		return nil, err
	}

	windows := make([]ipc.Window, len(clients))
	for i, c := range clients {
		windows[i] = ipc.Window{
			ID:           c.Address,
			Title:        c.Title,
			AppID:        c.Class,
			WorkspaceID:  fmt.Sprintf("%d", c.Workspace.ID),
			IsFocused:    false, // Will be updated if active
			IsUrgent:     c.Urgent,
			IsFloating:   c.Floating,
			IsFullscreen: c.Fullscreen == 2,
			IsHidden:     false,
			Metadata: map[string]interface{}{
				"monitor_id": fmt.Sprintf("%d", c.Monitor),
				"pinned":     c.Pinned,
				"x":          c.At[0],
				"y":          c.At[1],
				"width":      c.Size[0],
				"height":     c.Size[1],
			},
		}
	}
	return windows, nil
}

func (h *Hyprland) ActiveWindow() (string, error) {
	resp, err := h.dispatch("j/activewindow")
	if err != nil {
		return "", err
	}
	var active struct {
		Address string `json:"address"`
	}
	if err := json.Unmarshal([]byte(resp), &active); err != nil {
		return "", err
	}
	return active.Address, nil
}

func (h *Hyprland) FocusWindow(id string) error {
	_, err := h.dispatchVersioned(
		fmt.Sprintf("focuswindow %s", hyprTarget(id)),
		fmt.Sprintf("hl.dsp.focus({ window = %q })", hyprTarget(id)),
	)
	return err
}

func (h *Hyprland) FocusDir(direction string) error {
	_, err := h.dispatchVersioned(
		fmt.Sprintf("movefocus %s", direction),
		fmt.Sprintf("hl.dsp.focus({ direction = %q })", luaDirection(direction)),
	)
	return err
}

func (h *Hyprland) CloseWindow(id string) error {
	target := hyprTarget(id)
	lua := "hl.dsp.window.close()"
	if target != "" {
		lua = fmt.Sprintf("hl.dsp.window.close({ window = %q })", target)
	}
	_, err := h.dispatchVersioned(fmt.Sprintf("closewindow %s", target), lua)
	return err
}

func (h *Hyprland) MoveWindow(id string, direction string) error {
	arg := direction
	if id != "" {
		arg = direction + "," + hyprTarget(id)
	}
	_, err := h.dispatchVersioned(
		fmt.Sprintf("movewindow %s", arg),
		fmt.Sprintf("hl.dsp.window.move({ direction = %q%s })", luaDirection(direction), luaTargetField(id)),
	)
	return err
}

func (h *Hyprland) ResizeWindow(id string, width, height int) error {
	target := hyprTarget(id)
	_, err := h.dispatchVersioned(
		fmt.Sprintf("resizewindowpixel exact %d %d,%s", width, height, target),
		fmt.Sprintf("hl.dsp.window.resize({ x = %d, y = %d%s })", width, height, luaTargetField(id)),
	)
	return err
}

func (h *Hyprland) ToggleFloating(id string) error {
	target := hyprTarget(id)
	_, err := h.dispatchVersioned(
		fmt.Sprintf("togglefloating %s", target),
		fmt.Sprintf("hl.dsp.window.float({ action = \"toggle\"%s })", luaTargetField(id)),
	)
	return err
}

func (h *Hyprland) SetFullscreen(id string, state bool) error {
	windows, err := h.ListWindows()
	if err != nil {
		return err
	}

	targetID := id
	if targetID == "" {
		targetID, _ = h.ActiveWindow()
	}

	isFs := false
	for _, w := range windows {
		if w.ID == targetID {
			isFs = w.IsFullscreen
			break
		}
	}

	if isFs != state {
		_, err := h.dispatchVersioned(
			"fullscreen 0",
			"hl.dsp.window.fullscreen({ mode = \"fullscreen\", action = \"toggle\" })",
		)
		return err
	}
	return nil
}

func (h *Hyprland) SetMaximized(id string, state bool) error {
	val := "0"
	if state {
		val = "1"
	}
	action := "unset"
	if state {
		action = "set"
	}
	_, err := h.dispatchVersioned(
		fmt.Sprintf("fullscreen %s", val),
		fmt.Sprintf("hl.dsp.window.fullscreen({ mode = \"maximized\", action = %q })", action),
	)
	return err
}

func (h *Hyprland) PinWindow(id string, state bool) error {
	target := hyprTarget(id)
	_, err := h.dispatchVersioned(
		fmt.Sprintf("pin %s", target),
		fmt.Sprintf("hl.dsp.window.pin({ action = \"toggle\"%s })", luaTargetField(id)),
	)
	return err
}

func (h *Hyprland) ToggleGroup(id string) error {
	_, err := h.dispatchVersioned("togglegroup", "hl.dsp.group.toggle()")
	return err
}

func (h *Hyprland) GroupNav(direction string) error {
	dir := "f"
	if direction == "l" || direction == "u" || direction == "b" {
		dir = "b"
	}
	luaDir := "next"
	if dir == "b" {
		luaDir = "prev"
	}
	_, err := h.dispatchVersioned(
		fmt.Sprintf("changegroupactive %s", dir),
		fmt.Sprintf("hl.dsp.group.%s()", luaDir),
	)
	return err
}

func (h *Hyprland) SetLayoutProperty(id string, key, value string) error {
	return ipc.ErrNotSupported
}

func (h *Hyprland) ListWorkspaces() ([]ipc.Workspace, error) {
	resp, err := h.dispatch("j/workspaces")
	if err != nil {
		return nil, err
	}

	var workspaces []struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Monitor string `json:"monitor"`
	}

	if err := json.Unmarshal([]byte(resp), &workspaces); err != nil {
		return nil, err
	}

	activeResp, _ := h.dispatch("j/activeworkspace")
	var activeWS struct {
		ID int `json:"id"`
	}
	if activeResp != "" {
		json.Unmarshal([]byte(activeResp), &activeWS)
	}

	res := make([]ipc.Workspace, len(workspaces))
	for i, w := range workspaces {
		res[i] = ipc.Workspace{
			ID:        fmt.Sprintf("%d", w.ID),
			Name:      w.Name,
			MonitorID: w.Monitor,
			IsActive:  w.ID == activeWS.ID,
			IsEmpty:   false, // Not directly available from basic j/workspaces without parsing windows
			Metadata: map[string]interface{}{
				"focused": w.ID == activeWS.ID,
			},
		}
	}
	return res, nil
}

func (h *Hyprland) ActiveWorkspace() (*ipc.Workspace, error) {
	resp, err := h.dispatch("j/activeworkspace")
	if err != nil {
		return nil, err
	}
	var ws struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Monitor string `json:"monitor"`
	}
	if err := json.Unmarshal([]byte(resp), &ws); err != nil {
		return nil, err
	}
	return &ipc.Workspace{
		ID:        fmt.Sprintf("%d", ws.ID),
		Name:      ws.Name,
		MonitorID: ws.Monitor,
		IsActive:  true,
		IsEmpty:   false,
		Metadata: map[string]interface{}{
			"focused": true,
		},
	}, nil
}

func (h *Hyprland) SwitchWorkspace(id string) error {
	_, err := h.dispatchVersioned(
		fmt.Sprintf("workspace %s", id),
		fmt.Sprintf("hl.dsp.focus({ workspace = %q })", id),
	)
	return err
}

func (h *Hyprland) MoveToWorkspace(windowID, workspaceID string) error {
	target := hyprTarget(windowID)
	_, err := h.dispatchVersioned(
		fmt.Sprintf("movetoworkspace %s,%s", workspaceID, target),
		fmt.Sprintf("hl.dsp.window.move({ workspace = %q%s })", workspaceID, luaTargetField(windowID)),
	)
	return err
}

func (h *Hyprland) ListMonitors() ([]ipc.Monitor, error) {
	resp, err := h.dispatch("j/monitors")
	if err != nil {
		return nil, err
	}

	var monitors []struct {
		ID              int     `json:"id"`
		Name            string  `json:"name"`
		Width           int     `json:"width"`
		Height          int     `json:"height"`
		RefreshRate     float64 `json:"refreshRate"`
		Focused         bool    `json:"focused"`
		Scale           float64 `json:"scale"`
		X               int     `json:"x"`
		Y               int     `json:"y"`
		Transform       int     `json:"transform"`
		ActiveWorkspace struct {
			Name string `json:"name"`
		} `json:"activeWorkspace"`
	}

	if err := json.Unmarshal([]byte(resp), &monitors); err != nil {
		return nil, err
	}

	res := make([]ipc.Monitor, len(monitors))
	for i, m := range monitors {
		res[i] = ipc.Monitor{
			ID:          fmt.Sprintf("%d", m.ID),
			Name:        m.Name,
			Description: "",
			Width:       m.Width,
			Height:      m.Height,
			RefreshRate: m.RefreshRate,
			Scale:       m.Scale,
			IsFocused:   m.Focused,
			Metadata: map[string]interface{}{
				"active_workspace": m.ActiveWorkspace.Name,
				"x":                m.X,
				"y":                m.Y,
				"transform":        m.Transform,
			},
		}
	}
	return res, nil
}

func (h *Hyprland) FocusMonitor(id string) error {
	_, err := h.dispatchVersioned(
		fmt.Sprintf("focusmonitor %s", id),
		fmt.Sprintf("hl.dsp.focus({ monitor = %q })", id),
	)
	return err
}

func (h *Hyprland) MoveToMonitor(windowID, monitorID string) error {
	target := hyprTarget(windowID)
	_, err := h.dispatchVersioned(
		fmt.Sprintf("movewindowmon %s,%s", monitorID, target),
		fmt.Sprintf("hl.dsp.window.move({ monitor = %q%s })", monitorID, luaTargetField(windowID)),
	)
	return err
}

func (h *Hyprland) SetDpms(monitorID string, on bool) error {
	state := "off"
	if on {
		state = "on"
	}
	if monitorID != "" {
		_, err := h.dispatchVersioned(
			fmt.Sprintf("dpms %s %s", state, monitorID),
			fmt.Sprintf("hl.dsp.dpms({ action = %q, monitor = %q })", state, monitorID),
		)
		return err
	}
	_, err := h.dispatchVersioned(
		fmt.Sprintf("dpms %s", state),
		fmt.Sprintf("hl.dsp.dpms({ action = %q })", state),
	)
	return err
}

func (h *Hyprland) SetLayout(name string) error {
	_, err := h.dispatch(fmt.Sprintf("keyword general:layout %s", name))
	return err
}

func (h *Hyprland) MoveWindowPixel(id string, x, y int) error {
	target := hyprTarget(id)
	_, err := h.dispatchVersioned(
		fmt.Sprintf("movewindowpixel exact %d %d,%s", x, y, target),
		fmt.Sprintf("hl.dsp.window.move({ x = %d, y = %d%s })", x, y, luaTargetField(id)),
	)
	return err
}

func (h *Hyprland) MoveToWorkspaceSilent(windowID, workspaceID string) error {
	target := hyprTarget(windowID)
	_, err := h.dispatchVersioned(
		fmt.Sprintf("movetoworkspacesilent %s,%s", workspaceID, target),
		fmt.Sprintf("hl.dsp.window.move({ workspace = %q, follow = false%s })", workspaceID, luaTargetField(windowID)),
	)
	return err
}

func (h *Hyprland) ToggleSpecialWorkspace(name string) error {
	if name == "" {
		_, err := h.dispatchVersioned("togglespecialworkspace", "hl.dsp.workspace.toggle_special()")
		return err
	}
	_, err := h.dispatchVersioned(
		fmt.Sprintf("togglespecialworkspace %s", name),
		fmt.Sprintf("hl.dsp.workspace.toggle_special(%q)", name),
	)
	return err
}

func (h *Hyprland) GetConfig(key string) (interface{}, error) {
	resp, err := h.dispatch(fmt.Sprintf("j/getoption %s", key))
	if err != nil {
		return nil, err
	}
	var data interface{}
	if err := json.Unmarshal([]byte(resp), &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (h *Hyprland) BatchConfig(configs map[string]interface{}) error {
	var cmds []string

	mapping := map[string]string{
		"gaps.inner":            "general:gaps_in",
		"gaps.outer":            "general:gaps_out",
		"border.width":          "general:border_size",
		"border.active_color":   "general:col.active_border",
		"border.inactive_color": "general:col.inactive_border",
		"opacity.active":        "decoration:active_opacity",
		"opacity.inactive":      "decoration:inactive_opacity",
		"blur.enabled":          "decoration:blur:enabled",
		"blur.size":             "decoration:blur:size",
		"blur.passes":           "decoration:blur:passes",
	}

	for k, v := range configs {
		hyprKey := k
		if mapped, ok := mapping[k]; ok {
			hyprKey = mapped
		}
		cmds = append(cmds, fmt.Sprintf("keyword %s %v", hyprKey, v))
	}
	_, err := h.dispatch(fmt.Sprintf("[[BATCH]]%s", strings.Join(cmds, ";")))
	return err
}

// keybindKeyString renders modifiers + key in the Lua config syntax
// ("MODS + KEY", e.g. "SUPER + Q"); bare key with no modifiers.
func keybindKeyString(mods []string, key string) string {
	joined := strings.Join(mods, " + ")
	if joined == "" {
		return key
	}
	return joined + " + " + key
}

// bindToLua renders one hl.bind(...) expression for a keybind.
func bindToLua(b ipc.Keybind) string {
	keyStr := keybindKeyString(b.Modifiers, b.Key)

	dispatcher := b.Dispatcher
	if dispatcher == "" || dispatcher == "exec" {
		dispatcher = fmt.Sprintf("hl.dsp.exec_cmd(%q)", b.Argument)
	} else {
		dispatcher = dispatcherToLua(dispatcher, b.Argument)
	}

	var opts []string
	if flags := bindFlagsToLua(b.Flags); flags != "" {
		opts = append(opts, flags)
	}
	if strings.HasPrefix(strings.ToLower(b.Key), "mouse:") {
		opts = append(opts, "mouse = true")
	}
	if len(opts) == 0 {
		return fmt.Sprintf("hl.bind(%q, %s)", keyStr, dispatcher)
	}
	return fmt.Sprintf("hl.bind(%q, %s, { %s })", keyStr, dispatcher, strings.Join(opts, ", "))
}

// dispatchLuaFunction runs a Lua body through the text socket. On Hyprland
// >= 0.55 the socket compiles `dispatch <args>` as `return hl.dispatch(...)`,
// so passing a function executes it (the pre-0.55 `keyword`/`[[BATCH]]`
// commands no longer exist there).
func (h *Hyprland) dispatchLuaFunction(body string) (string, error) {
	return h.dispatch("dispatch function() " + body + " end")
}

func (h *Hyprland) BatchKeybinds(jsonPayload string) error {
	var payload ipc.BatchKeybindsPayload
	if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
		return fmt.Errorf("invalid keybinds payload: %w", err)
	}

	// Debug log – always, so we can diagnose total-failure reports.
	// Best-effort: ignore log errors.
	func() {
		f, err := os.OpenFile("/tmp/axctl_batch.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		defer f.Close()
		fmt.Fprintf(f, "[%s] BatchKeybinds payload: %d binds, %d unbinds, lua=%v\n",
			time.Now().Format(time.RFC3339), len(payload.Binds), len(payload.Unbinds), h.supportsLuaDispatchers())
		// Truncate payload for log if huge
		if len(jsonPayload) < 8192 {
			fmt.Fprintf(f, "  payload: %s\n", jsonPayload)
		} else {
			fmt.Fprintf(f, "  payload: %d bytes\n", len(jsonPayload))
		}
	}()

	// Try Lua path first – current Hyprland (>=0.55) requires it. The
	// supportsLuaDispatchers() check is advisory; if the dispatch socket is
	// temporarily unavailable it returns false and would incorrectly force the
	// legacy [[BATCH]] path which no longer exists on 0.56+. So we try Lua
	// first and fall back only on dispatch error.
	var luaBody string
	{
		var body strings.Builder
		for _, u := range payload.Unbinds {
			body.WriteString(fmt.Sprintf("pcall(function() hl.unbind(%q) end);",
				keybindKeyString(u.Modifiers, u.Key)))
		}
		for _, b := range payload.Binds {
			body.WriteString("pcall(function() " + bindToLua(b) + " end);")
		}
		luaBody = body.String()
	}

	if luaBody != "" {
		// Prefer Lua when version says so, but also try it even when version
		// detection failed – the fallback will handle the error.
		shouldTryLua := h.supportsLuaDispatchers()
		// Always try Lua first on modern Hyprland; if version detection said
		// false (e.g. transient socket error), still attempt Lua and fall back.
		if shouldTryLua || true {
			_, err := h.dispatchLuaFunction(luaBody)
			func() {
				f, _ := os.OpenFile("/tmp/axctl_batch.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if f == nil {
					return
				}
				defer f.Close()
				if err != nil {
					fmt.Fprintf(f, "[%s] Lua dispatch error: %v\n", time.Now().Format(time.RFC3339), err)
					fmt.Fprintf(f, "  lua body: %s\n", luaBody)
				} else {
					fmt.Fprintf(f, "[%s] Lua dispatch ok: %d binds, %d unbinds\n", time.Now().Format(time.RFC3339), len(payload.Binds), len(payload.Unbinds))
				}
			}()
			if err == nil {
				return nil
			}
			// If Lua failed with "unknown request" / "keyword" / "error:" it may be
			// an old Hyprland that doesn't understand Lua – fall through to
			// legacy. For other errors (socket not found etc.) propagate.
			if !strings.Contains(err.Error(), "unknown") && !strings.Contains(err.Error(), "keyword") && !strings.Contains(strings.ToLower(err.Error()), "error:") {
				return err
			}
			// Fall through to legacy on fallback-eligible errors
			fmt.Printf("[axctl] Lua BatchKeybinds failed (%v), falling back to legacy [[BATCH]]\n", err)
		}
	}

	var cmds []string

	// Process unbinds first
	for _, u := range payload.Unbinds {
		mods := strings.Join(u.Modifiers, " ")
		cmds = append(cmds, fmt.Sprintf("keyword unbind %s,%s", mods, u.Key))
	}

	// Process binds
	for _, b := range payload.Binds {
		mods := strings.Join(b.Modifiers, " ")
		bindKeyword := "bind"
		if b.Flags != "" {
			bindKeyword = "bind" + b.Flags
		}
		if b.Flags == "m" && b.Argument == "" {
			cmds = append(cmds, fmt.Sprintf("keyword %s %s,%s,%s", bindKeyword, mods, b.Key, b.Dispatcher))
		} else {
			cmds = append(cmds, fmt.Sprintf("keyword %s %s,%s,%s,%s", bindKeyword, mods, b.Key, b.Dispatcher, b.Argument))
		}
	}

	if len(cmds) == 0 {
		return nil
	}

	_, err := h.dispatch("[[BATCH]]" + strings.Join(cmds, ";"))
	func() {
		f, _ := os.OpenFile("/tmp/axctl_batch.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if f == nil {
			return
		}
		defer f.Close()
		if err != nil {
			fmt.Fprintf(f, "[%s] Legacy batch dispatch error: %v\n", time.Now().Format(time.RFC3339), err)
			fmt.Fprintf(f, "  cmds: %s\n", strings.Join(cmds, ";"))
		} else {
			fmt.Fprintf(f, "[%s] Legacy batch dispatch ok: %d cmds\n", time.Now().Format(time.RFC3339), len(cmds))
		}
	}()
	return err
}

func (h *Hyprland) RawBatch(command string) error {
	_, err := h.dispatch("[[BATCH]]" + command)
	return err
}

func (h *Hyprland) GetAnimations() (interface{}, error) {
	resp, err := h.dispatch("j/animations")
	if err != nil {
		return nil, err
	}
	var data interface{}
	if err := json.Unmarshal([]byte(resp), &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (h *Hyprland) GetCursorPosition() (int, int, error) {
	resp, err := h.dispatch("j/cursorpos")
	if err != nil {
		return 0, 0, err
	}
	var pos struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	if err := json.Unmarshal([]byte(resp), &pos); err != nil {
		return 0, 0, err
	}
	return pos.X, pos.Y, nil
}

func (h *Hyprland) BindKey(mods, key, command string) error {
	if h.supportsLuaDispatchers() {
		dispatcher, arg := command, ""
		if i := strings.IndexByte(command, ','); i != -1 {
			dispatcher, arg = command[:i], command[i+1:]
		}
		kb := ipc.Keybind{
			Modifiers:  []string{mods},
			Key:        key,
			Dispatcher: strings.TrimSpace(dispatcher),
			Argument:   strings.TrimSpace(arg),
			Enabled:    true,
		}
		_, err := h.dispatchLuaFunction(bindToLua(kb))
		return err
	}
	_, err := h.dispatch(fmt.Sprintf("keyword bind %s,%s,%s", mods, key, command))
	return err
}

func (h *Hyprland) UnbindKey(mods, key string) error {
	if h.supportsLuaDispatchers() {
		_, err := h.dispatchLuaFunction(fmt.Sprintf("hl.unbind(%q)", keybindKeyString([]string{mods}, key)))
		return err
	}
	_, err := h.dispatch(fmt.Sprintf("keyword unbind %s,%s", mods, key))
	return err
}

// hyprKeyToLuaTable converts a "general:col.active_border" style key into a
// nested hl.config table ({ general = { col = { ["active_border"] = v } } }).
func hyprKeyToLuaTable(hyprKey string, value interface{}) (string, error) {
	var luaValue string
	switch v := value.(type) {
	case string:
		switch strings.ToLower(v) {
		case "true", "false":
			luaValue = strings.ToLower(v)
		default:
			if n, err := strconv.ParseFloat(v, 64); err == nil {
				luaValue = fmt.Sprintf("%g", n)
			} else {
				luaValue = fmt.Sprintf("%q", v)
			}
		}
	case bool:
		luaValue = fmt.Sprintf("%t", v)
	case float64:
		luaValue = fmt.Sprintf("%g", v)
	case int:
		luaValue = strconv.Itoa(v)
	default:
		return "", fmt.Errorf("unsupported config value type %T", value)
	}

	parts := strings.FieldsFunc(hyprKey, func(r rune) bool { return r == ':' || r == '.' })
	if len(parts) == 0 {
		return "", fmt.Errorf("empty config key %q", hyprKey)
	}

	expr := luaValue
	for i := len(parts) - 1; i > 0; i-- {
		expr = fmt.Sprintf("{ %s = %s }", luaIdent(parts[i]), expr)
	}
	return expr, nil
}

func luaIdent(s string) string {
	if s == "" || (s[0] >= '0' && s[0] <= '9') || strings.ContainsAny(s, "-.") {
		return fmt.Sprintf("[%q]", s)
	}
	for _, r := range s {
		if !(r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return fmt.Sprintf("[%q]", s)
		}
	}
	return s
}

func (h *Hyprland) SetConfig(key string, value interface{}) error {
	mapping := map[string]string{
		"gaps.inner":            "general:gaps_in",
		"gaps.outer":            "general:gaps_out",
		"border.width":          "general:border_size",
		"border.active_color":   "general:col.active_border",
		"border.inactive_color": "general:col.inactive_border",
		"opacity.active":        "decoration:active_opacity",
		"opacity.inactive":      "decoration:inactive_opacity",
		"blur.enabled":          "decoration:blur:enabled",
		"blur.size":             "decoration:blur:size",
		"blur.passes":           "decoration:blur:passes",
	}

	hyprKey, ok := mapping[key]
	if !ok {
		hyprKey = key
	}

	if h.supportsLuaDispatchers() {
		table, err := hyprKeyToLuaTable(hyprKey, value)
		if err != nil {
			return err
		}
		_, err = h.dispatchLuaFunction(fmt.Sprintf("hl.config(%s)", table))
		return err
	}

	_, err := h.dispatch(fmt.Sprintf("keyword %s %v", hyprKey, value))
	return err
}

func (h *Hyprland) ReloadConfig() error {
	_, err := h.dispatch("reload")
	return err
}

func (h *Hyprland) Execute(command string) error {
	_, err := h.dispatchVersioned(
		fmt.Sprintf("exec %s", command),
		fmt.Sprintf("hl.dsp.exec_cmd(%q)", command),
	)
	return err
}

func (h *Hyprland) Exit() error {
	_, err := h.dispatchVersioned("exit", "hl.dsp.exit()")
	return err
}

func (h *Hyprland) Subscribe() (<-chan ipc.Event, error) {
	conn, err := net.Dial("unix", h.getSocketPath(".socket2.sock"))
	if err != nil {
		return nil, err
	}

	ch := make(chan ipc.Event)
	go func() {
		defer conn.Close()
		defer close(ch)
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.SplitN(line, ">>", 2)
			if len(parts) < 2 {
				continue
			}

			event := ipc.Event{
				Timestamp: time.Now().Unix(),
				Payload:   make(map[string]interface{}),
			}

			switch parts[0] {
			case "openwindow":
				event.Type = ipc.EventWindowCreated
				data := strings.SplitN(parts[1], ",", 4)
				if len(data) >= 4 {
					event.Window = &ipc.Window{
						ID:          "0x" + data[0],
						WorkspaceID: data[1],
						AppID:       data[2],
						Title:       data[3],
					}
				}
			case "closewindow":
				event.Type = ipc.EventWindowClosed
				event.Payload["address"] = "0x" + parts[1]
			case "activewindow":
				event.Type = ipc.EventWindowFocused
				data := strings.SplitN(parts[1], ",", 2)
				if len(data) >= 2 {
					event.Payload["class"] = data[0]
					event.Payload["title"] = data[1]
				}
			case "activewindowv2":
				event.Type = ipc.EventWindowFocused
				addr := strings.TrimSpace(parts[1])
				if addr != "" {
					event.Payload["address"] = "0x" + addr
				}
			case "workspace":
				event.Type = ipc.EventWorkspaceChanged
				event.Payload["name"] = parts[1]
			case "movewindow":
				data := strings.SplitN(parts[1], ",", 2)
				if len(data) >= 2 {
					event.Type = ipc.EventWindowMoved
					event.Payload["address"] = "0x" + data[0]
					event.Payload["workspace"] = data[1]
				}
			case "changefloatingmode", "floating":
				data := strings.SplitN(parts[1], ",", 2)
				if len(data) >= 2 {
					event.Payload["address"] = "0x" + data[0]
					event.Payload["floating"] = data[1] == "1"
				}
			case "fullscreen":
				event.Type = ipc.EventFullscreenChanged
				event.Payload["fullscreen"] = parts[1] == "1"
			case "monitoradded":
				event.Type = ipc.EventMonitorChanged
				event.Payload["monitor"] = parts[1]
				event.Payload["action"] = "added"
			case "monitorremoved":
				event.Type = ipc.EventMonitorChanged
				event.Payload["monitor"] = parts[1]
				event.Payload["action"] = "removed"
			case "configreloaded":
				event.Type = ipc.EventConfigReloaded
			case "focusedmon":
				event.Type = ipc.EventFocusedMonitorChanged
				data := strings.SplitN(parts[1], ",", 2)
				if len(data) >= 2 {
					event.Payload["monitor"] = data[0]
					event.Payload["workspace"] = data[1]
				}
			case "windowtitle":
				event.Type = ipc.EventWindowTitleChanged
				event.Payload["address"] = "0x" + parts[1]
			case "urgent":
				event.Type = ipc.EventWindowUrgent
				addr, urgent := parseUrgentPayload(parts[1])
				event.Payload["address"] = addr
				event.Payload["urgent"] = urgent
			}

			if event.Type != "" || len(event.Payload) > 0 {
				ch <- event
			}
		}
	}()

	return ch, nil
}

// parseUrgentPayload normalizes the payload of Hyprland's "urgent" socket2
// event into a window address (with "0x" prefix) and the urgent state.
// Newer Hyprland versions prefix the address with the urgency state
// ("1,0xADDR" / "0,0xADDR"); older versions send just the address, which the
// event only carries on transitions to urgent.
func parseUrgentPayload(payload string) (address string, urgent bool) {
	p := strings.TrimSpace(payload)
	if idx := strings.IndexByte(p, ','); idx != -1 {
		state, addr := strings.TrimSpace(p[:idx]), strings.TrimSpace(p[idx+1:])
		urgent = state == "1"
		address = "0x" + strings.TrimPrefix(addr, "0x")
		return address, urgent
	}
	return "0x" + strings.TrimPrefix(p, "0x"), true
}

func (h *Hyprland) SwitchKeyboardLayout(action string) error {
	cmd := fmt.Sprintf("switchxkblayout current %s", action)
	_, err := h.dispatch(cmd)
	return err
}

func (h *Hyprland) SetKeyboardLayouts(layouts string, variants string) error {
	if _, err := h.dispatch(fmt.Sprintf("keyword input:kb_layout %s", layouts)); err != nil {
		return err
	}
	if variants != "" {
		if _, err := h.dispatch(fmt.Sprintf("keyword input:kb_variant %s", variants)); err != nil {
			return err
		}
	} else {
		h.dispatch("keyword input:kb_variant ") // clear
	}
	return nil
}

func (h *Hyprland) GetCapabilities() (ipc.Capabilities, error) {
	return ipc.Capabilities{
		Blur:                true,
		Shadows:             true,
		Animations:          true,
		RoundedCorners:      true,
		WorkspacesSupported: true,
		WindowsSupported:    true,
	}, nil
}
