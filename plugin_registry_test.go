package main

import (
	"net/http"
	"testing"
)

type stubPlugin struct {
	id             string
	defaultEnabled bool
	lastProfileID  string
	notifyCount    int
	shutdownCalled bool
}

func (s *stubPlugin) ID() string           { return s.id }
func (s *stubPlugin) DefaultEnabled() bool { return s.defaultEnabled }
func (s *stubPlugin) Descriptor(_ *App) PluginDescriptor {
	return PluginDescriptor{ID: s.id, Name: "Stub", Enabled: s.defaultEnabled}
}
func (s *stubPlugin) RegisterRoutes(_ *http.ServeMux, _ *App) {}
func (s *stubPlugin) OnEnabledChanged(_ *App, _ bool)         {}
func (s *stubPlugin) OnEffectiveProfileChanged(_ *App, p Profile) {
	s.notifyCount++
	s.lastProfileID = p.Id
}
func (s *stubPlugin) Shutdown() { s.shutdownCalled = true }

func TestPluginRegistryLifecycleAndDedup(t *testing.T) {
	p1 := &stubPlugin{id: "alpha", defaultEnabled: true}
	pDup := &stubPlugin{id: "alpha", defaultEnabled: false}
	p2 := &stubPlugin{id: "beta", defaultEnabled: false}

	reg := newPluginRegistry(p1, pDup, p2, nil)
	list := reg.List(nil)
	if len(list) != 2 {
		t.Fatalf("len(list)=%d want 2", len(list))
	}
	if reg.Get("alpha") != p1 {
		t.Fatal("duplicate plugin ID must keep first registration")
	}
	reg.NotifyEffectiveProfile(nil, Profile{Id: "game-1", Name: "Game"})
	if p1.notifyCount != 1 || p1.lastProfileID != "game-1" || p2.notifyCount != 1 {
		t.Fatalf("unexpected notify state: p1=%+v p2=%+v", p1, p2)
	}
	reg.ShutdownAll()
	if !p1.shutdownCalled || !p2.shutdownCalled {
		t.Fatal("ShutdownAll did not invoke every plugin")
	}
}

func TestPluginsConfigDefaultsAndOverride(t *testing.T) {
	var cfg PluginsConfig
	if !cfg.IsEnabled("sinowealth", true) {
		t.Fatal("expected default true when map is nil")
	}
	if cfg.IsEnabled("experimental", false) {
		t.Fatal("expected default false when map is nil")
	}
	cfg.Enabled = map[string]bool{"sinowealth": false}
	if cfg.IsEnabled("sinowealth", true) {
		t.Fatal("explicit false override should win over default true")
	}
}
