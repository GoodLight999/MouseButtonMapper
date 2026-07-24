package main

import (
	"path"
	"strings"
	"time"
)

// AutoSwitchConfig is optional in config.json. Older configurations without
// this block remain valid and simply start with automatic switching disabled.
type AutoSwitchConfig struct {
	Enabled    bool         `json:"Enabled"`
	DebounceMs int          `json:"DebounceMs,omitempty"`
	Bindings   []AppBinding `json:"Bindings,omitempty"`
}

// AppBinding is evaluated in slice order. The first enabled binding whose
// non-empty conditions all match wins.
type AppBinding struct {
	Id            string `json:"Id"`
	Enabled       bool   `json:"Enabled"`
	Name          string `json:"Name"`
	ProfileId     string `json:"ProfileId"`
	ProcessName   string `json:"ProcessName,omitempty"`
	TitleContains string `json:"TitleContains,omitempty"`
	PathContains  string `json:"PathContains,omitempty"`
}

// ForegroundAppInfo is the foreground-window snapshot used by the matcher.
type ForegroundAppInfo struct {
	HWND        uintptr
	PID         uint32
	ProcessName string
	Path        string
	Title       string
	Source      string
	SeenAt      time.Time
}

func (f ForegroundAppInfo) Valid() bool {
	return f.HWND != 0 || f.PID != 0 || strings.TrimSpace(f.ProcessName) != "" || strings.TrimSpace(f.Title) != ""
}

func baseNameAnySeparator(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\\", "/"))
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "/")
	return parts[len(parts)-1]
}

func hasGlobMeta(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

func processNameCandidates(s string) []string {
	s = strings.ToLower(baseNameAnySeparator(s))
	if s == "" {
		return nil
	}
	out := []string{s}
	if strings.HasSuffix(s, ".exe") {
		out = append(out, strings.TrimSuffix(s, ".exe"))
	}
	return out
}

// matchProcessName accepts either tasklist style names (foo.exe) or
// PowerShell Get-Process style names (foo). Wildcards are supported.
func matchProcessName(actual, pattern string) bool {
	actuals := processNameCandidates(actual)
	patterns := processNameCandidates(pattern)
	if len(actuals) == 0 || len(patterns) == 0 {
		return false
	}
	for _, p := range patterns {
		for _, a := range actuals {
			if hasGlobMeta(p) {
				ok, err := path.Match(p, a)
				if err == nil && ok {
					return true
				}
			} else if a == p {
				return true
			}
		}
	}
	return false
}

func matchContains(actual, needle string) bool {
	actual = strings.ToLower(strings.TrimSpace(actual))
	needle = strings.ToLower(strings.TrimSpace(needle))
	return needle != "" && strings.Contains(actual, needle)
}

type BindingMatchResult struct {
	Matches        bool
	ConditionCount int
	Reasons        []string
}

func evaluateBindingMatch(b AppBinding, app ForegroundAppInfo) BindingMatchResult {
	result := BindingMatchResult{}
	if !b.Enabled {
		result.Reasons = append(result.Reasons, "ルールが無効")
		return result
	}
	if strings.TrimSpace(b.ProcessName) != "" {
		result.ConditionCount++
		if matchProcessName(app.ProcessName, b.ProcessName) {
			result.Reasons = append(result.Reasons, "プロセス名: 一致")
		} else {
			result.Reasons = append(result.Reasons, "プロセス名: 不一致")
			return result
		}
	}
	if strings.TrimSpace(b.TitleContains) != "" {
		result.ConditionCount++
		if matchContains(app.Title, b.TitleContains) {
			result.Reasons = append(result.Reasons, "ウィンドウタイトル: 一致")
		} else {
			result.Reasons = append(result.Reasons, "ウィンドウタイトル: 不一致")
			return result
		}
	}
	if strings.TrimSpace(b.PathContains) != "" {
		result.ConditionCount++
		if matchContains(app.Path, b.PathContains) {
			result.Reasons = append(result.Reasons, "実行ファイルパス: 一致")
		} else {
			result.Reasons = append(result.Reasons, "実行ファイルパス: 不一致")
			return result
		}
	}
	if result.ConditionCount == 0 {
		result.Reasons = append(result.Reasons, "判定条件が未設定")
		return result
	}
	result.Matches = true
	return result
}

func bindingMatches(b AppBinding, app ForegroundAppInfo) bool {
	return evaluateBindingMatch(b, app).Matches
}

func firstMatchingBinding(bindings []AppBinding, app ForegroundAppInfo, validProfile func(string) bool) (int, AppBinding, bool) {
	for i, b := range bindings {
		if strings.TrimSpace(b.ProfileId) == "" || (validProfile != nil && !validProfile(b.ProfileId)) {
			continue
		}
		if bindingMatches(b, app) {
			return i, b, true
		}
	}
	return -1, AppBinding{}, false
}

// foregroundMatchTarget keeps the settings UI out of app matching. When the
// settings window is foreground, the last non-settings app remains the target
// so enabling or editing automatic switching can complete its debounce.
func foregroundMatchTarget(observed, lastExternal ForegroundAppInfo, settingsWindow bool) (ForegroundAppInfo, bool) {
	if settingsWindow {
		if lastExternal.Valid() {
			return lastExternal, true
		}
		return ForegroundAppInfo{}, false
	}
	if observed.Valid() {
		return observed, false
	}
	return ForegroundAppInfo{}, false
}
