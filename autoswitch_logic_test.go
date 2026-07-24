package main

import "testing"

func TestBindingMatchesProcessExactCaseInsensitive(t *testing.T) {
	b := AppBinding{Enabled: true, ProcessName: "GAME.EXE", ProfileId: "game"}
	app := ForegroundAppInfo{ProcessName: `C:\\Games\\game.exe`}
	if !bindingMatches(b, app) {
		t.Fatal("expected exact process-name match")
	}
}

func TestBindingMatchesProcessWithoutExe(t *testing.T) {
	b := AppBinding{Enabled: true, ProcessName: "msedge", ProfileId: "browser"}
	app := ForegroundAppInfo{ProcessName: "msedge.exe"}
	if !bindingMatches(b, app) {
		t.Fatal("expected Get-Process style name to match tasklist style name")
	}
}

func TestBindingMatchesProcessWildcard(t *testing.T) {
	b := AppBinding{Enabled: true, ProcessName: "red*", ProfileId: "game"}
	app := ForegroundAppInfo{ProcessName: "RedGame-Win64-Shipping.exe"}
	if !bindingMatches(b, app) {
		t.Fatal("expected wildcard process-name match")
	}
}

func TestBindingConditionsAreANDed(t *testing.T) {
	b := AppBinding{Enabled: true, ProcessName: "game.exe", TitleContains: "Battle", ProfileId: "game"}
	if bindingMatches(b, ForegroundAppInfo{ProcessName: "game.exe", Title: "Launcher"}) {
		t.Fatal("title condition must also match")
	}
	if !bindingMatches(b, ForegroundAppInfo{ProcessName: "game.exe", Title: "Battle Arena"}) {
		t.Fatal("all specified conditions should match")
	}
}

func TestFirstEnabledBindingWins(t *testing.T) {
	bindings := []AppBinding{
		{Id: "off", Enabled: false, ProcessName: "game.exe", ProfileId: "p1"},
		{Id: "first", Enabled: true, ProcessName: "game.exe", ProfileId: "p2"},
		{Id: "second", Enabled: true, ProcessName: "game.exe", ProfileId: "p3"},
	}
	idx, b, ok := firstMatchingBinding(bindings, ForegroundAppInfo{ProcessName: "game.exe"}, func(id string) bool { return true })
	if !ok || idx != 1 || b.Id != "first" {
		t.Fatalf("unexpected winner: ok=%v idx=%d id=%q", ok, idx, b.Id)
	}
}

func TestBindingRequiresAtLeastOneCondition(t *testing.T) {
	b := AppBinding{Enabled: true, ProfileId: "p"}
	if bindingMatches(b, ForegroundAppInfo{ProcessName: "anything.exe"}) {
		t.Fatal("empty binding must not match everything")
	}
}

func TestBindingPathAndTitleContainsCaseInsensitive(t *testing.T) {
	b := AppBinding{Enabled: true, TitleContains: "ARENA", PathContains: `games\\foo`, ProfileId: "p"}
	app := ForegroundAppInfo{Title: "Foo Arena - Match", Path: `C:\\Games\\Foo\\foo.exe`}
	if !bindingMatches(b, app) {
		t.Fatal("expected case-insensitive title/path containment match")
	}
}

func TestFirstMatchingBindingSkipsMissingProfile(t *testing.T) {
	bindings := []AppBinding{
		{Id: "bad", Enabled: true, ProcessName: "game.exe", ProfileId: "missing"},
		{Id: "good", Enabled: true, ProcessName: "game.exe", ProfileId: "valid"},
	}
	idx, b, ok := firstMatchingBinding(bindings, ForegroundAppInfo{ProcessName: "game.exe"}, func(id string) bool { return id == "valid" })
	if !ok || idx != 1 || b.Id != "good" {
		t.Fatalf("unexpected winner: ok=%v idx=%d id=%q", ok, idx, b.Id)
	}
}

func TestMatchResultExplainsFirstFailure(t *testing.T) {
	b := AppBinding{Enabled: true, ProcessName: "game.exe", TitleContains: "battle", ProfileId: "p"}
	r := evaluateBindingMatch(b, ForegroundAppInfo{ProcessName: "game.exe", Title: "launcher"})
	if r.Matches || len(r.Reasons) != 2 || r.Reasons[1] != "ウィンドウタイトル: 不一致" {
		t.Fatalf("unexpected result: %+v", r)
	}
}

func TestForegroundMatchTargetUsesLastExternalForSettings(t *testing.T) {
	settings := ForegroundAppInfo{ProcessName: "msedge.exe", Title: "MouseButtonMapper"}
	game := ForegroundAppInfo{ProcessName: "game.exe", Title: "Game"}
	got, reused := foregroundMatchTarget(settings, game, true)
	if !reused || got.ProcessName != "game.exe" {
		t.Fatalf("settings UI must preserve last external app: reused=%v got=%+v", reused, got)
	}
}

func TestForegroundMatchTargetNeedsHistoryForSettings(t *testing.T) {
	settings := ForegroundAppInfo{ProcessName: "msedge.exe", Title: "MouseButtonMapper"}
	got, reused := foregroundMatchTarget(settings, ForegroundAppInfo{}, true)
	if reused || got.Valid() {
		t.Fatalf("unexpected target without history: reused=%v got=%+v", reused, got)
	}
}
