//go:build windows

package main

import "testing"

func TestRuleOnlyRebuildDoesNotRequestJoyConRescan(t *testing.T) {
	worker := &JoyConWorker{rescan: make(chan struct{}, 1)}
	app := &App{
		config: Config{
			ActiveProfileId: "profile",
			Profiles: []Profile{{
				Id:   "profile",
				Name: "Profile",
			}},
		},
		activeProfileIndex: 0,
		joyConWorker:       worker,
	}

	app.rebuildRulesWithoutJoyConRescanLocked()
	select {
	case <-worker.rescan:
		t.Fatal("rule-only rebuild requested a Joy-Con reconnect")
	default:
	}

	app.rebuildRulesLocked()
	select {
	case <-worker.rescan:
	default:
		t.Fatal("profile/config rebuild did not request a Joy-Con reconnect")
	}
}
