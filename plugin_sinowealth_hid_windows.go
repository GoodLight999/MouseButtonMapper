//go:build windows

package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	swHIDDLL      = syscall.NewLazyDLL("hid.dll")
	swSetupAPIDLL = syscall.NewLazyDLL("setupapi.dll")
	swKernel32DLL = syscall.NewLazyDLL("kernel32.dll")

	swHidDGetHidGuid        = swHIDDLL.NewProc("HidD_GetHidGuid")
	swHidDGetAttributes     = swHIDDLL.NewProc("HidD_GetAttributes")
	swHidDGetProductString  = swHIDDLL.NewProc("HidD_GetProductString")
	swHidDGetPreparsedData  = swHIDDLL.NewProc("HidD_GetPreparsedData")
	swHidDFreePreparsedData = swHIDDLL.NewProc("HidD_FreePreparsedData")
	swHidPGetCaps           = swHIDDLL.NewProc("HidP_GetCaps")
	swHidDSetFeature        = swHIDDLL.NewProc("HidD_SetFeature")
	swHidDGetFeature        = swHIDDLL.NewProc("HidD_GetFeature")

	swSetupDiGetClassDevs             = swSetupAPIDLL.NewProc("SetupDiGetClassDevsW")
	swSetupDiEnumDeviceInterfaces     = swSetupAPIDLL.NewProc("SetupDiEnumDeviceInterfaces")
	swSetupDiGetDeviceInterfaceDetail = swSetupAPIDLL.NewProc("SetupDiGetDeviceInterfaceDetailW")
	swSetupDiDestroyDeviceInfoList    = swSetupAPIDLL.NewProc("SetupDiDestroyDeviceInfoList")

	swCreateFile  = swKernel32DLL.NewProc("CreateFileW")
	swCloseHandle = swKernel32DLL.NewProc("CloseHandle")
)

type swGUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type swSPDeviceInterfaceData struct {
	CbSize             uint32
	InterfaceClassGuid swGUID
	Flags              uint32
	Reserved           uintptr
}

type swHIDDAttributes struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

type swHIDPCaps struct {
	Usage                     uint16
	UsagePage                 uint16
	InputReportByteLength     uint16
	OutputReportByteLength    uint16
	FeatureReportByteLength   uint16
	Reserved                  [17]uint16
	NumberLinkCollectionNodes uint16
	NumberInputButtonCaps     uint16
	NumberInputValueCaps      uint16
	NumberInputDataIndices    uint16
	NumberOutputButtonCaps    uint16
	NumberOutputValueCaps     uint16
	NumberOutputDataIndices   uint16
	NumberFeatureButtonCaps   uint16
	NumberFeatureValueCaps    uint16
	NumberFeatureDataIndices  uint16
}

type swDevicePaths struct {
	VID         uint16
	PID         uint16
	ProductName string
	CmdPath     string
	CfgPath     string
}

type sinoWealthBackupFile struct {
	SavedAt    string `json:"savedAt"`
	VendorID   uint16 `json:"vendorId"`
	ProductID  uint16 `json:"productId"`
	Firmware   string `json:"firmware"`
	ConfigHex  string `json:"configHex"`
	ButtonsHex string `json:"buttonsHex"`
}

// SinoWealthPlugin manages SinoWealth-based mice (such as Ray Pawn+ RM-3360LR, 258A:1007, FW GM07).
type SinoWealthPlugin struct {
	ioMu          sync.Mutex
	stateMu       sync.RWMutex
	lastState     *SinoWealthConfigState
	lastReadAt    time.Time
	lastError     string
	backupPath    string
	hasBackup     bool
	lastSyncAt    time.Time
	lastSyncMsg   string
	syncCh        chan sinoWealthSyncJob
	stopCh        chan struct{}
	stopOnce      sync.Once
	workerStarted bool
}

type sinoWealthSyncJob struct {
	app     *App
	profile Profile
}

func newSinoWealthPlugin() *SinoWealthPlugin {
	p := &SinoWealthPlugin{
		syncCh: make(chan sinoWealthSyncJob, 1),
		stopCh: make(chan struct{}),
	}
	go p.syncWorker()
	return p
}

func (p *SinoWealthPlugin) ID() string { return "sinowealth" }

func (p *SinoWealthPlugin) DefaultEnabled() bool { return true }

func (p *SinoWealthPlugin) Shutdown() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
	})
}

func (p *SinoWealthPlugin) OnEnabledChanged(a *App, enabled bool) {
	if enabled {
		if a != nil {
			a.mu.RLock()
			var prof Profile
			if a.activeProfileIndex >= 0 && a.activeProfileIndex < len(a.config.Profiles) {
				prof = a.config.Profiles[a.activeProfileIndex]
			}
			a.mu.RUnlock()
			p.OnEffectiveProfileChanged(a, prof)
		}
	}
}

func (p *SinoWealthPlugin) OnEffectiveProfileChanged(a *App, profile Profile) {
	// Never acquire a.mu here because caller may hold a.mu.Lock(); check enabled inside syncWorker.
	if a == nil {
		return
	}
	if !profile.SinoWealth.Enabled {
		return
	}
	if profile.SinoWealth.ReportRateHz == 0 && profile.SinoWealth.ActiveDPISlot == 0 && profile.SinoWealth.LEDMode == 0 {
		return
	}
	// Non-blocking replace of pending job so hook/autoswitch threads never stall.
	select {
	case <-p.stopCh:
		return
	default:
	}
	select {
	case <-p.syncCh:
	default:
	}
	select {
	case p.syncCh <- sinoWealthSyncJob{app: a, profile: profile}:
	default:
	}
}

func (p *SinoWealthPlugin) syncWorker() {
	for {
		select {
		case <-p.stopCh:
			return
		case job := <-p.syncCh:
			p.applyProfileHardwareLink(job.app, job.profile)
		}
	}
}

func (p *SinoWealthPlugin) colorOrder(a *App) string {
	if a == nil {
		return "RGB"
	}
	a.mu.RLock()
	order := a.config.Plugins.SinoWealthColorOrder
	a.mu.RUnlock()
	return normalizeColorOrder(order)
}

func (p *SinoWealthPlugin) applyProfileHardwareLink(a *App, profile Profile) {
	if a != nil && !a.isPluginEnabled(p.ID(), p.DefaultEnabled()) {
		return
	}
	link := profile.SinoWealth
	if !link.Enabled || (link.ReportRateHz == 0 && link.ActiveDPISlot == 0 && link.LEDMode == 0) {
		return
	}
	// Rate-limit Flash writes to avoid wear and USB stutter during rapid Alt+Tab.
	p.stateMu.RLock()
	since := time.Since(p.lastSyncAt)
	p.stateMu.RUnlock()
	if since < 900*time.Millisecond {
		select {
		case <-p.stopCh:
			return
		case <-time.After(900*time.Millisecond - since):
		}
	}

	st, _, err := p.ReadDevice(a)
	if err != nil {
		p.setSyncStatus(fmt.Sprintf("連動スキップ (%s): %v", profile.Name, err))
		return
	}

	order := p.colorOrder(a)
	patch := SinoWealthConfigPatch{
		ActiveDPIIndex: -1,
		RGBEffectMode:  -1,
		ColorOrder:     order,
	}
	needWrite := false
	if link.ReportRateHz > 0 && link.ReportRateHz != st.ReportRateHz {
		patch.ReportRateHz = link.ReportRateHz
		needWrite = true
	}
	if link.ActiveDPISlot >= 1 && link.ActiveDPISlot <= sinoWealthDPISlots {
		targetIdx := link.ActiveDPISlot - 1
		if targetIdx < len(st.DPISlots) && st.DPISlots[targetIdx].Enabled && targetIdx != st.ActiveDPIIndex {
			patch.ActiveDPIIndex = targetIdx
			needWrite = true
		}
	}
	switch link.LEDMode {
	case 1: // Single solid color
		wantColor := strings.ToUpper(strings.TrimSpace(link.LEDColor))
		if wantColor == "" {
			wantColor = "#0000FF"
		}
		curActiveColor := ""
		if st.ActiveDPIIndex >= 0 && st.ActiveDPIIndex < len(st.DPISlots) {
			curActiveColor = strings.ToUpper(st.DPISlots[st.ActiveDPIIndex].Color)
		}
		if st.RGBEffectMode != 2 || !strings.EqualFold(st.SingleColor, wantColor) || !strings.EqualFold(curActiveColor, wantColor) {
			patch.RGBEffectMode = 2
			patch.SingleColor = wantColor
			patch.SingleBrightness = 4
			patch.SyncActiveDPIColor = true
			needWrite = true
		}
	case 2: // LED OFF
		if st.RGBEffectMode != 0 {
			patch.RGBEffectMode = 0
			needWrite = true
		}
	case 3: // Rainbow
		if st.RGBEffectMode != 1 {
			patch.RGBEffectMode = 1
			needWrite = true
		}
	}

	if !needWrite {
		p.setSyncStatus(fmt.Sprintf("プロファイル「%s」のマウス設定はすでに一致しています (%dHz / DPI #%d / LED %s)", profile.Name, st.ReportRateHz, st.ActiveDPIIndex+1, st.SingleColor))
		return
	}

	updated, err := p.WriteConfigPatch(a, patch)
	if err != nil {
		p.setSyncStatus(fmt.Sprintf("連動適用失敗 (%s): %v", profile.Name, err))
		if a != nil {
			a.logf("sinowealth profile sync failed (%s): %v", profile.Name, err)
		}
		return
	}
	p.setSyncStatus(fmt.Sprintf("プロファイル「%s」へ連動適用: %dHz / DPI #%d (%d DPI) / LED mode=%d %s", profile.Name, updated.ReportRateHz, updated.ActiveDPIIndex+1, updated.DPISlots[updated.ActiveDPIIndex].DPI, updated.RGBEffectMode, updated.SingleColor))
	if a != nil {
		a.logf("sinowealth profile synced: profile=%q rate=%dHz dpiSlot=%d ledMode=%d color=%s", profile.Name, updated.ReportRateHz, updated.ActiveDPIIndex+1, updated.RGBEffectMode, updated.SingleColor)
	}
}

func (p *SinoWealthPlugin) setSyncStatus(msg string) {
	p.stateMu.Lock()
	p.lastSyncAt = time.Now()
	p.lastSyncMsg = msg
	p.stateMu.Unlock()
}

func (p *SinoWealthPlugin) Descriptor(a *App) PluginDescriptor {
	enabled := true
	if a != nil {
		enabled = a.isPluginEnabled(p.ID(), p.DefaultEnabled())
	}
	paths, _ := swFindTargetDevice()
	detected := paths != nil

	p.stateMu.RLock()
	st := p.lastState
	p.stateMu.RUnlock()

	summary := "未検出（Ray Pawn+ / 258A:1007 を接続すると利用可能）"
	if detected {
		if st != nil {
			summary = fmt.Sprintf("接続中: %s (FW %s / %s / %dHz)", st.ProductName, st.Firmware, st.SensorName, st.ReportRateHz)
		} else {
			summary = fmt.Sprintf("検出済み: %s (VID %04X / PID %04X)", paths.ProductName, paths.VID, paths.PID)
		}
	}
	return PluginDescriptor{
		ID:          p.ID(),
		Name:        "Ray Pawn+ / SinoWealth マウス設定",
		Category:    "mouse-hardware",
		Description: "公式ソフトが失われた SinoWealth 系マウス (258A:1007 / FW GM07) のオンボードボタン・F13〜F24化・DPI・ポーリングレート・プロファイル連動を安全に管理します。",
		Enabled:     enabled,
		Detected:    detected,
		Summary:     summary,
		ScriptURL:   "/plugins/sinowealth-ui.js",
	}
}

func (p *SinoWealthPlugin) RegisterRoutes(mux *http.ServeMux, a *App) {
	mux.HandleFunc("/api/plugins/sinowealth", func(w http.ResponseWriter, r *http.Request) {
		p.handleAPI(w, r, a)
	})
	mux.HandleFunc("/plugins/sinowealth-ui.js", p.handleUIJS)
}

func swFindTargetDevice() (*swDevicePaths, error) {
	var guid swGUID
	swHidDGetHidGuid.Call(uintptr(unsafe.Pointer(&guid)))

	const (
		digcfPresent         = 0x02
		digcfDeviceInterface = 0x10
		fileShareRead        = 0x01
		fileShareWrite       = 0x02
		openExisting         = 3
		hidpStatusSuccess    = 0x00110000
	)

	hDev, _, _ := swSetupDiGetClassDevs.Call(uintptr(unsafe.Pointer(&guid)), 0, 0, digcfPresent|digcfDeviceInterface)
	if hDev == 0 || hDev == ^uintptr(0) {
		return nil, errors.New("HIDデバイスの列挙に失敗しました")
	}
	defer swSetupDiDestroyDeviceInfoList.Call(hDev)

	var found swDevicePaths
	for idx := uint32(0); ; idx++ {
		var ifData swSPDeviceInterfaceData
		ifData.CbSize = uint32(unsafe.Sizeof(ifData))
		ok, _, _ := swSetupDiEnumDeviceInterfaces.Call(hDev, 0, uintptr(unsafe.Pointer(&guid)), uintptr(idx), uintptr(unsafe.Pointer(&ifData)))
		if ok == 0 {
			break
		}
		var req uint32
		swSetupDiGetDeviceInterfaceDetail.Call(hDev, uintptr(unsafe.Pointer(&ifData)), 0, 0, uintptr(unsafe.Pointer(&req)), 0)
		if req < 8 {
			continue
		}
		buf := make([]byte, req)
		// SP_DEVICE_INTERFACE_DETAIL_DATA_W.cbSize is 8 on amd64, 6 on 386
		cbSize := uint32(8)
		if unsafe.Sizeof(uintptr(0)) == 4 {
			cbSize = 6
		}
		*(*uint32)(unsafe.Pointer(&buf[0])) = cbSize
		ok, _, _ = swSetupDiGetDeviceInterfaceDetail.Call(hDev, uintptr(unsafe.Pointer(&ifData)), uintptr(unsafe.Pointer(&buf[0])), uintptr(req), 0, 0)
		if ok == 0 {
			continue
		}
		pathUTF16 := unsafe.Slice((*uint16)(unsafe.Pointer(&buf[4])), (len(buf)-4)/2)
		devPath := syscall.UTF16ToString(pathUTF16)

		pPtr, err := syscall.UTF16PtrFromString(devPath)
		if err != nil {
			continue
		}
		h, _, _ := swCreateFile.Call(uintptr(unsafe.Pointer(pPtr)), 0, fileShareRead|fileShareWrite, 0, openExisting, 0, 0)
		if h == 0 || h == ^uintptr(0) {
			continue
		}
		var attrs swHIDDAttributes
		attrs.Size = uint32(unsafe.Sizeof(attrs))
		attrOK, _, _ := swHidDGetAttributes.Call(h, uintptr(unsafe.Pointer(&attrs)))
		if attrOK != 0 && attrs.VendorID == sinoWealthVID && attrs.ProductID == sinoWealthPID1007 {
			found.VID = attrs.VendorID
			found.PID = attrs.ProductID
			if found.ProductName == "" {
				prodBuf := make([]uint16, 128)
				if r, _, _ := swHidDGetProductString.Call(h, uintptr(unsafe.Pointer(&prodBuf[0])), uintptr(len(prodBuf)*2)); r != 0 {
					found.ProductName = strings.TrimSpace(syscall.UTF16ToString(prodBuf))
				}
			}
			var ppd uintptr
			if r, _, _ := swHidDGetPreparsedData.Call(h, uintptr(unsafe.Pointer(&ppd))); r != 0 && ppd != 0 {
				var caps swHIDPCaps
				if st, _, _ := swHidPGetCaps.Call(ppd, uintptr(unsafe.Pointer(&caps))); uint32(st) == hidpStatusSuccess && caps.UsagePage == 0xFF00 {
					switch caps.FeatureReportByteLength {
					case sinoWealthCmdLen:
						found.CmdPath = devPath
					case sinoWealthReportLen:
						found.CfgPath = devPath
					}
				}
				swHidDFreePreparsedData.Call(ppd)
			}
		}
		swCloseHandle.Call(h)
	}
	if found.CmdPath == "" || found.CfgPath == "" {
		return nil, nil
	}
	if found.ProductName == "" {
		found.ProductName = "SINOWEALTH Wired Gaming Mouse"
	}
	return &found, nil
}

func swOpenRW(path string) (uintptr, error) {
	const (
		genericRead    = 0x80000000
		genericWrite   = 0x40000000
		fileShareRead  = 0x01
		fileShareWrite = 0x02
		openExisting   = 3
	)
	pPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	h, _, callErr := swCreateFile.Call(uintptr(unsafe.Pointer(pPtr)), genericRead|genericWrite, fileShareRead|fileShareWrite, 0, openExisting, 0, 0)
	if h == 0 || h == ^uintptr(0) {
		return 0, fmt.Errorf("HIDハンドルを開けませんでした: %v", callErr)
	}
	return h, nil
}

func swQueryCmd(hCmd uintptr, cmdID byte) ([]byte, error) {
	req := [sinoWealthCmdLen]byte{sinoWealthReportCmd, cmdID}
	if r, _, err := swHidDSetFeature.Call(hCmd, uintptr(unsafe.Pointer(&req[0])), sinoWealthCmdLen); r == 0 {
		return nil, fmt.Errorf("SetFeature(cmd=0x%02X) 失敗: %v", cmdID, err)
	}
	resp := [sinoWealthCmdLen]byte{sinoWealthReportCmd}
	if r, _, err := swHidDGetFeature.Call(hCmd, uintptr(unsafe.Pointer(&resp[0])), sinoWealthCmdLen); r == 0 {
		return nil, fmt.Errorf("GetFeature(cmd=0x%02X) 失敗: %v", cmdID, err)
	}
	if resp[1] != cmdID {
		return nil, fmt.Errorf("コマンド 0x%02X に未対応です（応答: 0x%02X）", cmdID, resp[1])
	}
	return resp[:], nil
}

func swReadReportOnce(hCmd, hCfg uintptr, cmdID byte) ([]byte, error) {
	req := [sinoWealthCmdLen]byte{sinoWealthReportCmd, cmdID}
	if r, _, err := swHidDSetFeature.Call(hCmd, uintptr(unsafe.Pointer(&req[0])), sinoWealthCmdLen); r == 0 {
		return nil, fmt.Errorf("SetFeature(cmd=0x%02X) 失敗: %v", cmdID, err)
	}
	buf := make([]byte, sinoWealthReportLen)
	buf[0] = sinoWealthReportConfig
	if r, _, err := swHidDGetFeature.Call(hCfg, uintptr(unsafe.Pointer(&buf[0])), sinoWealthReportLen); r == 0 {
		return nil, fmt.Errorf("GetFeature(0x04 / cmd=0x%02X) 失敗: %v", cmdID, err)
	}
	if buf[1] != cmdID {
		return nil, fmt.Errorf("期待しないレポート応答です (cmd=0x%02X, got=0x%02X)", cmdID, buf[1])
	}
	return buf, nil
}

// swReadReport reads a 520-byte report and confirms two consecutive reads match
// so transient USB HID glitches never corrupt UI state or Read-Modify-Write patches.
func swReadReport(hCmd, hCfg uintptr, cmdID byte) ([]byte, error) {
	var prev []byte
	for attempt := 0; attempt < 5; attempt++ {
		cur, err := swReadReportOnce(hCmd, hCfg, cmdID)
		if err != nil {
			return nil, err
		}
		if prev != nil && bytes.Equal(prev, cur) {
			return cur, nil
		}
		prev = cur
		time.Sleep(15 * time.Millisecond)
	}
	return nil, fmt.Errorf("レポート 0x%02X の読み出しが安定しませんでした（再試行してください）", cmdID)
}

func swWriteReport(hCfg uintptr, report []byte) error {
	if len(report) != sinoWealthReportLen {
		return fmt.Errorf("書き込みレポート長が不正です: %d", len(report))
	}
	if r, _, err := swHidDSetFeature.Call(hCfg, uintptr(unsafe.Pointer(&report[0])), sinoWealthReportLen); r == 0 {
		return fmt.Errorf("SetFeature(0x04) 書き込み失敗: %v", err)
	}
	// Critical safety wait: SinoWealth Flash writes corrupt adjacent blocks if accessed immediately.
	time.Sleep(450 * time.Millisecond)
	return nil
}

func (p *SinoWealthPlugin) backupFilePath(a *App, vid, pid uint16, fw string) string {
	baseDir := ""
	if a != nil && a.configPath != "" {
		baseDir = filepath.Join(filepath.Dir(a.configPath), "backups")
	} else {
		baseDir = filepath.Join(os.TempDir(), "MouseButtonMapper", "backups")
	}
	safeFW := strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, fw)
	if safeFW == "" {
		safeFW = "unknown"
	}
	return filepath.Join(baseDir, fmt.Sprintf("sinowealth_%04x_%04x_%s_initial.json", vid, pid, safeFW))
}

func (p *SinoWealthPlugin) ensureInitialBackup(a *App, vid, pid uint16, fw string, cfgBuf, btnBuf []byte) (string, bool) {
	bpath := p.backupFilePath(a, vid, pid, fw)
	if _, err := os.Stat(bpath); err == nil {
		return bpath, true
	}
	if err := os.MkdirAll(filepath.Dir(bpath), 0o755); err != nil {
		return bpath, false
	}
	rec := sinoWealthBackupFile{
		SavedAt:    time.Now().Format(time.RFC3339),
		VendorID:   vid,
		ProductID:  pid,
		Firmware:   fw,
		ConfigHex:  hex.EncodeToString(cfgBuf),
		ButtonsHex: hex.EncodeToString(btnBuf),
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return bpath, false
	}
	if err := os.WriteFile(bpath, data, 0o644); err != nil {
		return bpath, false
	}
	return bpath, true
}

func (p *SinoWealthPlugin) readLocked(a *App) (SinoWealthConfigState, []byte, []byte, error) {
	paths, err := swFindTargetDevice()
	if err != nil {
		return SinoWealthConfigState{}, nil, nil, err
	}
	if paths == nil {
		return SinoWealthConfigState{}, nil, nil, errors.New("対応する SinoWealth マウス (VID 258A / PID 1007) が見つかりません")
	}
	hCmd, err := swOpenRW(paths.CmdPath)
	if err != nil {
		return SinoWealthConfigState{}, nil, nil, err
	}
	defer swCloseHandle.Call(hCmd)
	hCfg, err := swOpenRW(paths.CfgPath)
	if err != nil {
		return SinoWealthConfigState{}, nil, nil, err
	}
	defer swCloseHandle.Call(hCfg)

	fwResp, err := swQueryCmd(hCmd, sinoWealthCmdFWVersion)
	if err != nil {
		return SinoWealthConfigState{}, nil, nil, fmt.Errorf("ファームウェア読み出し失敗: %w", err)
	}
	fw := strings.TrimRight(string(fwResp[2:6]), "\x00 ")

	activeProfile := 1
	if profResp, err := swQueryCmd(hCmd, sinoWealthCmdProfile); err == nil && profResp[2] >= 1 {
		activeProfile = int(profResp[2])
	}

	cfgBuf, err := swReadReport(hCmd, hCfg, sinoWealthCmdConfigP1)
	if err != nil {
		return SinoWealthConfigState{}, nil, nil, err
	}
	time.Sleep(40 * time.Millisecond)
	btnBuf, err := swReadReport(hCmd, hCfg, sinoWealthCmdButtonsP1)
	if err != nil {
		return SinoWealthConfigState{}, nil, nil, err
	}

	displayName := paths.ProductName
	if fw == "GM07" {
		displayName = "Ray Pawn+ (RM-3360LR)"
	}

	st, err := decodeSinoWealthState(paths.VID, paths.PID, fw, displayName, activeProfile, cfgBuf, btnBuf, p.colorOrder(a))
	if err != nil {
		return SinoWealthConfigState{}, nil, nil, err
	}

	bpath, hasBkp := p.ensureInitialBackup(a, paths.VID, paths.PID, fw, cfgBuf, btnBuf)
	stCopy := st
	p.stateMu.Lock()
	p.lastState = &stCopy
	p.lastReadAt = time.Now()
	p.lastError = ""
	p.backupPath = bpath
	p.hasBackup = hasBkp
	p.stateMu.Unlock()
	updateMouseAliasesFromSinoWealth(&stCopy)

	return st, cfgBuf, btnBuf, nil
}

func (p *SinoWealthPlugin) Warmup(a *App) {
	if a != nil && !a.isPluginEnabled(p.ID(), p.DefaultEnabled()) {
		return
	}
	_, _, _ = p.ReadDevice(a)
}

func (p *SinoWealthPlugin) ReadDevice(a *App) (SinoWealthConfigState, string, error) {
	p.ioMu.Lock()
	defer p.ioMu.Unlock()
	st, _, _, err := p.readLocked(a)
	if err != nil {
		p.stateMu.Lock()
		p.lastError = err.Error()
		p.stateMu.Unlock()
		return SinoWealthConfigState{}, "", err
	}
	p.stateMu.RLock()
	bp := p.backupPath
	p.stateMu.RUnlock()
	return st, bp, nil
}

func (p *SinoWealthPlugin) WriteButtons(a *App, updates []SinoWealthButtonAction) (SinoWealthConfigState, error) {
	p.ioMu.Lock()
	defer p.ioMu.Unlock()

	_, _, origBtn, err := p.readLocked(a)
	if err != nil {
		return SinoWealthConfigState{}, err
	}
	writeBuf, err := buildSinoWealthButtonsWriteReport(origBtn, updates)
	if err != nil {
		return SinoWealthConfigState{}, err
	}

	paths, err := swFindTargetDevice()
	if err != nil || paths == nil {
		return SinoWealthConfigState{}, errors.New("書き込み直前にマウスが切断されました")
	}
	hCfg, err := swOpenRW(paths.CfgPath)
	if err != nil {
		return SinoWealthConfigState{}, err
	}
	writeErr := swWriteReport(hCfg, writeBuf)
	swCloseHandle.Call(hCfg)
	if writeErr != nil {
		return SinoWealthConfigState{}, writeErr
	}

	// Verify read-back
	st, _, verifyBtn, err := p.readLocked(a)
	if err != nil {
		return SinoWealthConfigState{}, fmt.Errorf("書き込み後の確認読み出しに失敗: %w", err)
	}
	if !sinoWealthReportsEqualIgnoringWriteFlag(writeBuf, verifyBtn) {
		return st, errors.New("書き込み後の検証で一部バイトが一致しませんでした（再読み込みで確認してください）")
	}
	return st, nil
}

func (p *SinoWealthPlugin) WriteConfigPatch(a *App, patch SinoWealthConfigPatch) (SinoWealthConfigState, error) {
	p.ioMu.Lock()
	defer p.ioMu.Unlock()

	_, origCfg, _, err := p.readLocked(a)
	if err != nil {
		return SinoWealthConfigState{}, err
	}
	if patch.ColorOrder == "" {
		patch.ColorOrder = p.colorOrder(a)
	}
	writeBuf, err := buildSinoWealthConfigWriteReport(origCfg, patch)
	if err != nil {
		return SinoWealthConfigState{}, err
	}

	paths, err := swFindTargetDevice()
	if err != nil || paths == nil {
		return SinoWealthConfigState{}, errors.New("書き込み直前にマウスが切断されました")
	}
	hCfg, err := swOpenRW(paths.CfgPath)
	if err != nil {
		return SinoWealthConfigState{}, err
	}
	writeErr := swWriteReport(hCfg, writeBuf)
	swCloseHandle.Call(hCfg)
	if writeErr != nil {
		return SinoWealthConfigState{}, writeErr
	}

	st, verifyCfg, _, err := p.readLocked(a)
	if err != nil {
		return SinoWealthConfigState{}, fmt.Errorf("書き込み後の確認読み出しに失敗: %w", err)
	}
	if !sinoWealthReportsEqualIgnoringWriteFlag(writeBuf, verifyCfg) {
		return st, errors.New("書き込み後の設定検証で差分が検出されました")
	}
	return st, nil
}

func (p *SinoWealthPlugin) RestoreInitialBackup(a *App) (SinoWealthConfigState, error) {
	p.ioMu.Lock()
	defer p.ioMu.Unlock()

	st, _, _, err := p.readLocked(a)
	if err != nil {
		return SinoWealthConfigState{}, err
	}
	bpath := p.backupFilePath(a, st.VendorID, st.ProductID, st.Firmware)
	raw, err := os.ReadFile(bpath)
	if err != nil {
		return SinoWealthConfigState{}, fmt.Errorf("初期バックアップファイルが見つかりません (%s): %w", bpath, err)
	}
	var bkp sinoWealthBackupFile
	if err := json.Unmarshal(raw, &bkp); err != nil {
		return SinoWealthConfigState{}, fmt.Errorf("初期バックアップの解析に失敗: %w", err)
	}
	cfgBytes, err := hex.DecodeString(bkp.ConfigHex)
	if err != nil || len(cfgBytes) != sinoWealthReportLen {
		return SinoWealthConfigState{}, errors.New("バックアップ内の設定データが不正です")
	}
	btnBytes, err := hex.DecodeString(bkp.ButtonsHex)
	if err != nil || len(btnBytes) != sinoWealthReportLen {
		return SinoWealthConfigState{}, errors.New("バックアップ内のボタンデータが不正です")
	}
	cfgBytes[0x03] = sinoWealthConfigWrite
	btnBytes[0x03] = sinoWealthButtonsWrite

	paths, err := swFindTargetDevice()
	if err != nil || paths == nil {
		return SinoWealthConfigState{}, errors.New("復元直前にマウスが切断されました")
	}
	hCfg, err := swOpenRW(paths.CfgPath)
	if err != nil {
		return SinoWealthConfigState{}, err
	}
	// Write CONFIG first, wait safely, then write BUTTONS (never back-to-back without sleep).
	if err := swWriteReport(hCfg, cfgBytes); err != nil {
		swCloseHandle.Call(hCfg)
		return SinoWealthConfigState{}, fmt.Errorf("設定ブロックの復元に失敗: %w", err)
	}
	time.Sleep(250 * time.Millisecond)
	if err := swWriteReport(hCfg, btnBytes); err != nil {
		swCloseHandle.Call(hCfg)
		return SinoWealthConfigState{}, fmt.Errorf("ボタンブロックの復元に失敗: %w", err)
	}
	swCloseHandle.Call(hCfg)

	restored, _, _, err := p.readLocked(a)
	if err != nil {
		return SinoWealthConfigState{}, fmt.Errorf("復元後の再読み込みに失敗: %w", err)
	}
	return restored, nil
}
