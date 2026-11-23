package main

import (
	"fmt"
	"net/http"
)

// PluginsConfig stores opt-in plugin states inside config.json (backward compatible optional field).
type PluginsConfig struct {
	Enabled map[string]bool `json:"Enabled,omitempty"`
}

func (c PluginsConfig) IsEnabled(id string, defaultEnabled bool) bool {
	if c.Enabled == nil {
		return defaultEnabled
	}
	if v, ok := c.Enabled[id]; ok {
		return v
	}
	return defaultEnabled
}

// PluginDescriptor describes a modular built-in plugin exposed to the UI and HTTP API.
type PluginDescriptor struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"` // e.g. "mouse-hardware", "controller"
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Detected    bool   `json:"detected"`
	Summary     string `json:"summary"`
	ScriptURL   string `json:"scriptUrl,omitempty"`
}

// AppPlugin is the decoupled extension contract for adding hardware or workflow plugins
// without touching the low-level hook core.
type AppPlugin interface {
	ID() string
	DefaultEnabled() bool
	Descriptor(a *App) PluginDescriptor
	RegisterRoutes(mux *http.ServeMux, a *App)
	OnEnabledChanged(a *App, enabled bool)
	OnEffectiveProfileChanged(a *App, profile Profile)
	Shutdown()
}

type pluginRegistry struct {
	plugins []AppPlugin
	byID    map[string]AppPlugin
}

func newPluginRegistry(plugins ...AppPlugin) *pluginRegistry {
	reg := &pluginRegistry{
		plugins: make([]AppPlugin, 0, len(plugins)),
		byID:    make(map[string]AppPlugin, len(plugins)),
	}
	for _, p := range plugins {
		if p == nil || p.ID() == "" {
			continue
		}
		if _, dup := reg.byID[p.ID()]; dup {
			continue
		}
		reg.plugins = append(reg.plugins, p)
		reg.byID[p.ID()] = p
	}
	return reg
}

func (r *pluginRegistry) List(a *App) []PluginDescriptor {
	if r == nil {
		return []PluginDescriptor{}
	}
	out := make([]PluginDescriptor, 0, len(r.plugins))
	for _, p := range r.plugins {
		out = append(out, p.Descriptor(a))
	}
	return out
}

func (r *pluginRegistry) Get(id string) AppPlugin {
	if r == nil {
		return nil
	}
	return r.byID[id]
}

func (r *pluginRegistry) RegisterRoutes(mux *http.ServeMux, a *App) {
	if r == nil {
		return
	}
	for _, p := range r.plugins {
		p.RegisterRoutes(mux, a)
	}
}

func (r *pluginRegistry) NotifyEffectiveProfile(a *App, profile Profile) {
	if r == nil {
		return
	}
	for _, p := range r.plugins {
		p.OnEffectiveProfileChanged(a, profile)
	}
}

func (r *pluginRegistry) ShutdownAll() {
	if r == nil {
		return
	}
	for _, p := range r.plugins {
		p.Shutdown()
	}
}

func (a *App) isPluginEnabled(id string, defaultEnabled bool) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.Plugins.IsEnabled(id, defaultEnabled)
}

func (a *App) setPluginEnabled(id string, enabled bool) error {
	if a.plugins == nil {
		return fmt.Errorf("プラグインレジストリが初期化されていません。")
	}
	p := a.plugins.Get(id)
	if p == nil {
		return fmt.Errorf("未知のプラグインです: %s", id)
	}
	a.mu.Lock()
	if a.config.Plugins.Enabled == nil {
		a.config.Plugins.Enabled = map[string]bool{}
	}
	prev, hadPrev := a.config.Plugins.Enabled[id]
	a.config.Plugins.Enabled[id] = enabled
	if err := a.saveConfigLocked(); err != nil {
		if hadPrev {
			a.config.Plugins.Enabled[id] = prev
		} else {
			delete(a.config.Plugins.Enabled, id)
		}
		a.mu.Unlock()
		return err
	}
	a.postUIRefreshLocked()
	a.mu.Unlock()

	p.OnEnabledChanged(a, enabled)
	a.logf("plugin state changed: id=%s enabled=%v", id, enabled)
	return nil
}

type pluginToggleReq struct {
	Op      string `json:"op"`
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

func (a *App) webAPIPlugins(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, map[string]any{
			"ok":      true,
			"plugins": a.plugins.List(a),
		})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req pluginToggleReq
	if !decodeJSON(w, r, &req) {
		return
	}
	switch req.Op {
	case "", "set-enabled":
		if err := a.setPluginEnabled(req.ID, req.Enabled); err != nil {
			writeError(w, err)
			return
		}
		msg := fmt.Sprintf("プラグイン「%s」を無効化しました。", req.ID)
		if req.Enabled {
			msg = fmt.Sprintf("プラグイン「%s」を有効化しました。", req.ID)
		}
		writeJSON(w, map[string]any{
			"ok":      true,
			"message": msg,
			"plugins": a.plugins.List(a),
			"state":   a.buildWebState(),
		})
	default:
		writeError(w, fmt.Errorf("未知のプラグイン操作です: %s", req.Op))
	}
}
