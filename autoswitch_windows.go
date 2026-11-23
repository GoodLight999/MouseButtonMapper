//go:build windows

package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

var (
	asUser32   = syscall.NewLazyDLL("user32.dll")
	asAdvapi32 = syscall.NewLazyDLL("advapi32.dll")
	asKernel32 = syscall.NewLazyDLL("kernel32.dll")

	asEnumWindows         = asUser32.NewProc("EnumWindows")
	asIsWindowVisible     = asUser32.NewProc("IsWindowVisible")
	asGetWindowLongPtrW   = asUser32.NewProc("GetWindowLongPtrW")
	asGetShellWindow      = asUser32.NewProc("GetShellWindow")
	asOpenProcessToken    = asAdvapi32.NewProc("OpenProcessToken")
	asGetTokenInformation = asAdvapi32.NewProc("GetTokenInformation")
	asGetCurrentProcess   = asKernel32.NewProc("GetCurrentProcess")
)

type RunningAppEntry struct {
	PID         uint32 `json:"pid"`
	ProcessName string `json:"processName"`
	Title       string `json:"title"`
	Path        string `json:"path"`
	Elevated    bool   `json:"elevated"`
}

var (
	selfElevatedOnce sync.Once
	selfElevatedVal  bool
)

func isCurrentProcessElevated() bool {
	selfElevatedOnce.Do(func() {
		const (
			tokenQuery     = 0x0008
			tokenElevation = 20
		)
		hProc, _, _ := asGetCurrentProcess.Call()
		var tok uintptr
		r, _, _ := asOpenProcessToken.Call(hProc, tokenQuery, uintptr(unsafe.Pointer(&tok)))
		if r == 0 || tok == 0 {
			return
		}
		defer pCloseHandle.Call(tok)
		var elev uint32
		var retLen uint32
		r, _, _ = asGetTokenInformation.Call(tok, tokenElevation, uintptr(unsafe.Pointer(&elev)), unsafe.Sizeof(elev), uintptr(unsafe.Pointer(&retLen)))
		if r != 0 && elev != 0 {
			selfElevatedVal = true
		}
	})
	return selfElevatedVal
}

// isProcessElevated returns true if the target process appears to be running with
// elevated privileges (Admin/UIPI boundary) relative to a standard user process.
func isProcessElevated(pid uint32) bool {
	if pid == 0 {
		return false
	}
	const (
		tokenQuery     = 0x0008
		tokenElevation = 20
	)
	h, _, _ := pOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if h == 0 {
		// Opening even with QUERY_LIMITED_INFORMATION failed on a live window PID:
		// strong indicator of an elevated / protected game process.
		return !isCurrentProcessElevated()
	}
	defer pCloseHandle.Call(h)
	var tok uintptr
	r, _, _ := asOpenProcessToken.Call(h, tokenQuery, uintptr(unsafe.Pointer(&tok)))
	if r == 0 || tok == 0 {
		// Standard user process cannot query token of an elevated process (ERROR_ACCESS_DENIED).
		return !isCurrentProcessElevated()
	}
	defer pCloseHandle.Call(tok)
	var elev uint32
	var retLen uint32
	r, _, _ = asGetTokenInformation.Call(tok, tokenElevation, uintptr(unsafe.Pointer(&elev)), unsafe.Sizeof(elev), uintptr(unsafe.Pointer(&retLen)))
	return r != 0 && elev != 0
}

func listRunningApps(selfPID uint32) []RunningAppEntry {
	const (
		gwlExStyle     = -20
		wsExToolWindow = 0x00000080
		wsExNoActivate = 0x08000000
	)
	shellHWND, _, _ := asGetShellWindow.Call()
	seen := map[string]bool{}
	var out []RunningAppEntry

	cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		if hwnd == 0 || hwnd == shellHWND {
			return 1
		}
		vis, _, _ := asIsWindowVisible.Call(hwnd)
		if vis == 0 {
			return 1
		}
		title := strings.TrimSpace(foregroundWindowText(hwnd))
		if title == "" {
			return 1
		}
		exIdx := gwlExStyle
		exStyle, _, _ := asGetWindowLongPtrW.Call(hwnd, uintptr(exIdx))
		if exStyle&wsExToolWindow != 0 {
			return 1
		}
		var pid uint32
		pGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid == 0 || pid == selfPID {
			return 1
		}
		if isExactSettingsWindowTitle(title) {
			return 1
		}
		imgPath := processImagePath(pid)
		procName := baseNameAnySeparator(imgPath)
		if procName == "" {
			procName = processNameFromSnapshot(pid)
		}
		if procName == "" {
			return 1
		}
		key := strings.ToLower(procName) + "|" + strings.ToLower(title)
		if seen[key] {
			return 1
		}
		seen[key] = true
		out = append(out, RunningAppEntry{
			PID:         pid,
			ProcessName: procName,
			Title:       title,
			Path:        imgPath,
			Elevated:    isProcessElevated(pid),
		})
		return 1
	})

	asEnumWindows.Call(cb, 0)
	if len(out) == 0 {
		// Fallback for non-interactive desktops / headless test sessions: enumerate processes via snapshot.
		const th32csSnapProcess = 0x00000002
		hSnap, _, _ := pCreateToolhelp32Snapshot.Call(th32csSnapProcess, 0)
		if hSnap != 0 && hSnap != ^uintptr(0) {
			defer pCloseHandle.Call(hSnap)
			var pe PROCESSENTRY32W
			pe.DwSize = uint32(unsafe.Sizeof(pe))
			r, _, _ := pProcess32FirstW.Call(hSnap, uintptr(unsafe.Pointer(&pe)))
			for r != 0 {
				if pe.Th32ProcessID != 0 && pe.Th32ProcessID != selfPID {
					procName := syscall.UTF16ToString(pe.SzExeFile[:])
					key := strings.ToLower(procName)
					if procName != "" && !seen[key] {
						seen[key] = true
						imgPath := processImagePath(pe.Th32ProcessID)
						out = append(out, RunningAppEntry{
							PID:         pe.Th32ProcessID,
							ProcessName: procName,
							Path:        imgPath,
							Elevated:    isProcessElevated(pe.Th32ProcessID),
						})
					}
				}
				r, _, _ = pProcess32NextW.Call(hSnap, uintptr(unsafe.Pointer(&pe)))
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		pi := strings.ToLower(out[i].ProcessName)
		pj := strings.ToLower(out[j].ProcessName)
		if pi != pj {
			return pi < pj
		}
		return out[i].Title < out[j].Title
	})
	return out
}

func (a *App) webAPIRunningApps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	apps := listRunningApps(a.currentPID)
	writeJSON(w, map[string]any{
		"ok":           true,
		"selfElevated": isCurrentProcessElevated(),
		"apps":         apps,
	})
}

func (a *App) webAddBindingFromApp(req autoSwitchReq) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	profileID, err := a.resolveBindingProfileLocked(req.ProfileID)
	if err != nil {
		return err
	}
	procName := baseNameAnySeparator(req.ProcessName)
	title := strings.TrimSpace(req.TitleContains)
	pathContains := strings.TrimSpace(req.PathContains)
	if procName == "" && title == "" && pathContains == "" {
		return fmt.Errorf("アプリ情報が空のため自動切替ルールを作成できません。")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = defaultBindingName(ForegroundAppInfo{ProcessName: procName, Title: title})
	}
	b := AppBinding{
		Id:            newBindingID(),
		Enabled:       true,
		Name:          name,
		ProfileId:     profileID,
		ProcessName:   procName,
		TitleContains: title,
		PathContains:  pathContains,
	}
	a.config.AutoSwitch.Bindings = append(a.config.AutoSwitch.Bindings, b)
	a.autoCandidateKey = ""
	a.autoCandidateSince = a.autoCandidateSince.Truncate(0)
	return a.saveConfigLocked()
}
