package ipc

import (
	"encoding/json"
)

type Gaps struct {
	Inner *int `json:"inner,omitempty"`
	Outer *int `json:"outer,omitempty"`
}

type Border struct {
	Width         *int    `json:"width,omitempty"`
	ActiveColor   *string `json:"active_color,omitempty"`
	InactiveColor *string `json:"inactive_color,omitempty"`
	Rounding      *int    `json:"rounding,omitempty"`
}

type Opacity struct {
	Active   *float64 `json:"active,omitempty"`
	Inactive *float64 `json:"inactive,omitempty"`
}

type Blur struct {
	Enabled *bool `json:"enabled,omitempty"`
	Size    *int  `json:"size,omitempty"`
	Passes  *int  `json:"passes,omitempty"`
}

type Shadow struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Size    *int    `json:"size,omitempty"`
	Color   *string `json:"color,omitempty"`
}

type Animations struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type ConfigAppearance struct {
	Gaps       *Gaps       `json:"gaps,omitempty"`
	Border     *Border     `json:"border,omitempty"`
	Opacity    *Opacity    `json:"opacity,omitempty"`
	Blur       *Blur       `json:"blur,omitempty"`
	Shadow     *Shadow     `json:"shadow,omitempty"`
	Animations *Animations `json:"animations,omitempty"`
	Layout     *string     `json:"layout,omitempty"`
}

type Keybind struct {
	Modifiers  []string `json:"modifiers"`
	Key        string   `json:"key"`
	Dispatcher string   `json:"dispatcher"`
	Argument   string   `json:"argument"`
	Flags      string   `json:"flags,omitempty"`
	Enabled    bool     `json:"enabled"`
}

type KeybindTarget struct {
	Modifiers []string `json:"modifiers"`
	Key       string   `json:"key"`
}

type BatchKeybindsPayload struct {
	Binds   []Keybind       `json:"binds"`
	Unbinds []KeybindTarget `json:"unbinds"`
}

type SystemKeybinds map[string]Keybind

type AmbxstKeybinds struct {
	System map[string]Keybind `json:"system,omitempty"`
	Binds  map[string]Keybind `json:"-"`
}

func (a *AmbxstKeybinds) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	a.Binds = make(map[string]Keybind)

	for k, v := range raw {
		if k == "system" {
			if err := json.Unmarshal(v, &a.System); err != nil {
				return err
			}
		} else {
			var kb Keybind
			if err := json.Unmarshal(v, &kb); err != nil {
				return err
			}
			a.Binds[k] = kb
		}
	}
	return nil
}

type ConfigKeybinds struct {
	Ambxst *AmbxstKeybinds `json:"ambxst,omitempty"`
	Custom []Keybind       `json:"custom,omitempty"`
}

type WindowRule struct {
	Match  string `json:"match"`
	Rule   string `json:"rule"`
	Action string `json:"action"`

	Float         *bool   `json:"float,omitempty"`
	NoBlur        *bool   `json:"no_blur,omitempty"`
	NoShadow      *bool   `json:"no_shadow,omitempty"`
	Rounding      *int    `json:"rounding,omitempty"`
	BorderSize    *int    `json:"border_size,omitempty"`
	Pin           *bool   `json:"pin,omitempty"`
	Fullscreen    *bool   `json:"fullscreen,omitempty"`
	IdleInhibit   *bool   `json:"idle_inhibit,omitempty"`
	NoScreenShare *bool   `json:"no_screen_share,omitempty"`
	Move          *string `json:"move,omitempty"`
	Size          *string `json:"size,omitempty"`
	Name          string  `json:"name,omitempty"`
}

type LayerRule struct {
	NoAnim           *bool    `json:"no_anim,omitempty"`
	Blur             *bool    `json:"blur,omitempty"`
	BlurPopups       *bool    `json:"blur_popups,omitempty"`
	IgnoreAlpha      *bool    `json:"ignore_alpha,omitempty"`
	NoShadow         *bool    `json:"no_shadow,omitempty"`
	IgnoreZeroAlpha  *bool    `json:"ignore_zero_alpha,omitempty"`
	IgnoreAlphaValue *float64 `json:"ignore_alpha_value,omitempty"`
	Namespace        string   `json:"namespace"`
}

type ConfigUniversal struct {
	Appearance  ConfigAppearance `json:"appearance"`
	Keybinds    ConfigKeybinds   `json:"keybinds"`
	WindowRules []WindowRule     `json:"window_rules"`
	LayerRules  []LayerRule      `json:"layer_rules"`
	Exec        []string         `json:"exec,omitempty"`
	ExecOnce    []string         `json:"exec_once,omitempty"`
}

type ConfigGenerator interface {
	GenerateAppearance(config ConfigAppearance) string
	GenerateKeybinds(config ConfigKeybinds) string
	GenerateWindowRules(rules []WindowRule) string
	GenerateLayerRules(rules []LayerRule) string
	GenerateStartup(exec []string, execOnce []string) string
}

type LuaConfigGenerator interface {
	GenerateAppearanceLua(config ConfigAppearance) string
	GenerateKeybindsLua(config ConfigKeybinds) string
	GenerateWindowRulesLua(rules []WindowRule) string
	GenerateLayerRulesLua(rules []LayerRule) string
	GenerateStartupLua(exec []string, execOnce []string) string
}
