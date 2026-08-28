package ipc

type Window struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	AppID        string                 `json:"app_id"`
	WorkspaceID  string                 `json:"workspace_id"`
	IsFocused    bool                   `json:"is_focused"`
	IsUrgent     bool                   `json:"is_urgent"`
	IsFloating   bool                   `json:"is_floating"`
	IsFullscreen bool                   `json:"is_fullscreen"`
	IsHidden     bool                   `json:"is_hidden"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type Workspace struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	MonitorID string                 `json:"monitor_id"`
	IsActive  bool                   `json:"is_active"`
	IsEmpty   bool                   `json:"is_empty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type Monitor struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Width       int                    `json:"width"`
	Height      int                    `json:"height"`
	RefreshRate float64                `json:"refresh_rate,omitempty"`
	Scale       float64                `json:"scale,omitempty"`
	IsFocused   bool                   `json:"is_focused"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type Capabilities struct {
	Blur                bool `json:"blur"`
	Shadows             bool `json:"shadows"`
	Animations          bool `json:"animations"`
	RoundedCorners      bool `json:"rounded_corners"`
	WorkspacesSupported bool `json:"workspaces_supported"`
	WindowsSupported    bool `json:"windows_supported"`
}

type EventType string

const (
	EventWindowCreated         EventType = "window_created"
	EventWindowClosed          EventType = "window_closed"
	EventWindowFocused         EventType = "window_focused"
	EventWindowUrgent          EventType = "window_urgent"
	EventWindowTitleChanged    EventType = "window_title_changed"
	EventWindowMoved           EventType = "window_moved"
	EventWorkspaceChanged      EventType = "workspace_changed"
	EventMonitorChanged        EventType = "monitor_changed"
	EventConfigReloaded        EventType = "config_reloaded"
	EventFullscreenChanged     EventType = "fullscreen_changed"
	EventFocusedMonitorChanged EventType = "focused_monitor_changed"
)

type Event struct {
	Type      EventType
	Timestamp int64
	Window    *Window
	Workspace *Workspace
	Payload   map[string]interface{}
}
