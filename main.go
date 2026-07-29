//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const (
	WH_KEYBOARD_LL = 13
	WH_MOUSE_LL    = 14

	WM_CREATE        = 0x0001
	WM_DESTROY       = 0x0002
	WM_MOVE          = 0x0003
	WM_SIZE          = 0x0005
	WM_CLOSE         = 0x0010
	WM_QUIT          = 0x0012
	WM_COMMAND       = 0x0111
	WM_SETFONT       = 0x0030
	WM_APP           = 0x8000
	WM_USER          = 0x0400
	WM_RBUTTONUP     = 0x0205
	WM_LBUTTONDBLCLK = 0x0203
	WM_LBUTTONUP     = 0x0202
	WM_MOUSEWHEEL    = 0x020A
	WM_XBUTTONDOWN   = 0x020B
	WM_XBUTTONUP     = 0x020C
	WM_LBUTTONDOWN   = 0x0201
	WM_RBUTTONDOWN   = 0x0204
	WM_MBUTTONDOWN   = 0x0207
	WM_MBUTTONUP     = 0x0208
	WM_KEYDOWN       = 0x0100
	WM_KEYUP         = 0x0101
	WM_SYSKEYDOWN    = 0x0104
	WM_SYSKEYUP      = 0x0105

	HC_ACTION   = 0
	PM_NOREMOVE = 0x0000
	PM_REMOVE   = 0x0001

	EVENT_SYSTEM_FOREGROUND = 0x0003
	WINEVENT_OUTOFCONTEXT   = 0x0000
	WINEVENT_SKIPOWNPROCESS = 0x0002

	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_TABSTOP          = 0x00010000
	WS_BORDER           = 0x00800000
	WS_VSCROLL          = 0x00200000
	WS_EX_CLIENTEDGE    = 0x00000200

	BS_PUSHBUTTON        = 0x00000000
	BS_DEFPUSHBUTTON     = 0x00000001
	SS_LEFT              = 0x00000000
	ES_MULTILINE         = 0x0004
	ES_AUTOVSCROLL       = 0x0040
	ES_READONLY          = 0x0800
	LBS_NOTIFY           = 0x0001
	LBS_NOINTEGRALHEIGHT = 0x0100
	LB_ADDSTRING         = 0x0180
	LB_RESETCONTENT      = 0x0184
	LB_SETCURSEL         = 0x0186
	LB_GETCURSEL         = 0x0188
	LB_ERR               = -1
	LBN_SELCHANGE        = 1

	WM_PAINT         = 0x000F
	WM_GETTEXT       = 0x000D
	WM_GETTEXTLENGTH = 0x000E
	WM_NOTIFY        = 0x004E

	ES_AUTOHSCROLL   = 0x0080
	BS_AUTOCHECKBOX  = 0x00000003
	CBS_DROPDOWNLIST = 0x0003
	CBS_HASSTRINGS   = 0x0200
	CBN_SELCHANGE    = 1
	CB_ADDSTRING     = 0x0143
	CB_RESETCONTENT  = 0x014B
	CB_GETCURSEL     = 0x0147
	CB_SETCURSEL     = 0x014E
	CB_ERR           = -1
	BM_GETCHECK      = 0x00F0
	BM_SETCHECK      = 0x00F1
	BST_CHECKED      = 1
	BST_UNCHECKED    = 0

	LVS_REPORT                   = 0x0001
	LVS_SINGLESEL                = 0x0004
	LVS_SHOWSELALWAYS            = 0x0008
	LVS_NOSORTHEADER             = 0x8000
	LVS_EX_GRIDLINES             = 0x00000001
	LVS_EX_FULLROWSELECT         = 0x00000020
	LVS_EX_DOUBLEBUFFER          = 0x00010000
	LVM_FIRST                    = 0x1000
	LVM_DELETEALLITEMS           = LVM_FIRST + 9
	LVM_GETNEXTITEM              = LVM_FIRST + 12
	LVM_ENSUREVISIBLE            = LVM_FIRST + 19
	LVM_SETITEMSTATE             = LVM_FIRST + 43
	LVM_SETEXTENDEDLISTVIEWSTYLE = LVM_FIRST + 54
	LVM_INSERTCOLUMNW            = LVM_FIRST + 97
	LVM_INSERTITEMW              = LVM_FIRST + 77
	LVM_SETITEMTEXTW             = LVM_FIRST + 116
	LVM_SUBITEMHITTEST           = LVM_FIRST + 57
	LVNI_SELECTED                = 0x0002
	LVIF_TEXT                    = 0x0001
	LVCF_FMT                     = 0x0001
	LVCF_WIDTH                   = 0x0002
	LVCF_TEXT                    = 0x0004
	LVCF_SUBITEM                 = 0x0008
	LVIS_FOCUSED                 = 0x0001
	LVIS_SELECTED                = 0x0002
	LVN_FIRST                    = -100
	LVN_ITEMCHANGED              = LVN_FIRST - 1
	NM_CLICK                     = -2
	NM_DBLCLK                    = -3
	NM_CUSTOMDRAW                = -12
	CDDS_PREPAINT                = 0x00000001
	CDDS_ITEMPREPAINT            = 0x00010001
	CDDS_SUBITEM                 = 0x00020000
	CDRF_DODEFAULT               = 0x00000000
	CDRF_SKIPDEFAULT             = 0x00000004
	CDRF_NOTIFYITEMDRAW          = 0x00000020
	CDRF_NOTIFYSUBITEMDRAW       = 0x00000020
	CDIS_SELECTED                = 0x0001
	DFC_BUTTON                   = 4
	DFCS_BUTTONCHECK             = 0x0000
	DFCS_CHECKED                 = 0x0400
	DFCS_INACTIVE                = 0x0100
	COLOR_WINDOW                 = 5
	COLOR_HIGHLIGHT              = 13
	COLOR_BTNFACE                = 15

	ICC_LISTVIEW_CLASSES = 0x00000001
	TRANSPARENT          = 1
	DT_LEFT              = 0x00000000
	DT_VCENTER           = 0x00000004
	DT_SINGLELINE        = 0x00000020
	DT_END_ELLIPSIS      = 0x00008000
	DT_NOPREFIX          = 0x00000800

	SW_HIDE       = 0
	SW_SHOWNORMAL = 1
	SW_SHOW       = 5
	SW_RESTORE    = 9

	COINIT_APARTMENTTHREADED = 0x2
	S_OK                     = 0
	S_FALSE                  = 1

	CW_USEDEFAULT = 0x80000000

	NIM_ADD     = 0x00000000
	NIM_MODIFY  = 0x00000001
	NIM_DELETE  = 0x00000002
	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004

	TPM_RIGHTBUTTON = 0x0002
	TPM_BOTTOMALIGN = 0x0020

	MF_STRING    = 0x00000000
	MF_SEPARATOR = 0x00000800

	INPUT_MOUSE           = 0
	INPUT_KEYBOARD        = 1
	KEYEVENTF_KEYUP       = 0x0002
	KEYEVENTF_EXTENDEDKEY = 0x0001

	IMAGE_ICON      = 1
	LR_LOADFROMFILE = 0x00000010
	LR_DEFAULTSIZE  = 0x00000040

	DEFAULT_GUI_FONT = 17

	VK_LBUTTON  = 0x01
	VK_RBUTTON  = 0x02
	VK_MBUTTON  = 0x04
	VK_XBUTTON1 = 0x05
	VK_XBUTTON2 = 0x06
	VK_BACK     = 0x08
	VK_TAB      = 0x09
	VK_RETURN   = 0x0D
	VK_SHIFT    = 0x10
	VK_CONTROL  = 0x11
	VK_MENU     = 0x12
	VK_PAUSE    = 0x13
	VK_CAPITAL  = 0x14
	VK_ESCAPE   = 0x1B
	VK_SPACE    = 0x20
	VK_PRIOR    = 0x21
	VK_NEXT     = 0x22
	VK_END      = 0x23
	VK_HOME     = 0x24
	VK_LEFT     = 0x25
	VK_UP       = 0x26
	VK_RIGHT    = 0x27
	VK_DOWN     = 0x28
	VK_SNAPSHOT = 0x2C
	VK_LWIN     = 0x5B
	VK_RWIN     = 0x5C
	VK_F12      = 0x7B
	VK_LSHIFT   = 0xA0
	VK_RSHIFT   = 0xA1
	VK_LCONTROL = 0xA2
	VK_RCONTROL = 0xA3
	VK_LMENU    = 0xA4
	VK_RMENU    = 0xA5

	ID_BTN_STARTSTOP        = 1001
	ID_BTN_EMERGENCY        = 1002
	ID_BTN_RELEASE          = 1003
	ID_BTN_RELOAD           = 1004
	ID_BTN_OPENCFG          = 1005
	ID_BTN_OPENFOLDER       = 1006
	ID_BTN_ENABLE_RULE      = 1007
	ID_BTN_DISABLE_RULE     = 1008
	ID_BTN_EXPORT           = 1009
	ID_BTN_SAFE             = 1010
	ID_BTN_QUIT             = 1011
	ID_PROFILE_COMBO        = 1012
	ID_EDIT_INPUT           = 1013
	ID_EDIT_OUTPUT          = 1014
	ID_CHK_ENABLED          = 1015
	ID_CHK_SUPPRESS_TRIGGER = 1016
	ID_CHK_SUPPRESS_PREFIX  = 1017
	ID_COMBO_MODE           = 1018
	ID_BTN_SAVE_RULE        = 1019
	ID_BTN_ADD_RULE         = 1020
	ID_BTN_DELETE_RULE      = 1021
	ID_BTN_MOVE_UP          = 1022
	ID_BTN_MOVE_DOWN        = 1023
	ID_BTN_TEST_OUTPUT      = 1024
	ID_BTN_DUP_RULE         = 1025
	ID_BTN_REVERT_RULE      = 1026
	ID_BTN_OPENLOG          = 1027
	ID_BTN_RECORD_INPUT     = 1028
	ID_BTN_RECORD_OUTPUT    = 1029
	ID_BTN_RECORD_STOP      = 1030
	ID_BTN_DEFAULT_RULES    = 1031
	ID_BTN_MOVE_TOP         = 1032
	ID_BTN_MOVE_BOTTOM      = 1033
	ID_BTN_IMPORT           = 1034
	ID_BTN_COPY_DIAG        = 1035
	ID_LIST_RULES           = 1101

	ID_TRAY_SHOW       = 2001
	ID_TRAY_STARTSTOP  = 2002
	ID_TRAY_EMERGENCY  = 2003
	ID_TRAY_RELEASE    = 2004
	ID_TRAY_RELOAD     = 2005
	ID_TRAY_OPENCFG    = 2006
	ID_TRAY_OPENFOLDER = 2007
	ID_TRAY_ABOUT      = 2008
	ID_TRAY_EXIT       = 2009
)

const trayMsg = WM_APP + 77
const showMsg = WM_APP + 78
const refreshMsg = WM_APP + 79
const activityMsg = WM_APP + 80
const hookReinstallMsg = WM_APP + 181
const extraInfoMarker uintptr = 0x4D424D47584D4150 // "MBMGXMAP"
const mainWindowClass = "MouseButtonMapperMainWindow_v820"

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	pSetWindowsHookExW        = user32.NewProc("SetWindowsHookExW")
	pUnhookWindowsHookEx      = user32.NewProc("UnhookWindowsHookEx")
	pCallNextHookEx           = user32.NewProc("CallNextHookEx")
	pGetMessageW              = user32.NewProc("GetMessageW")
	pTranslateMessage         = user32.NewProc("TranslateMessage")
	pDispatchMessageW         = user32.NewProc("DispatchMessageW")
	pPostThreadMessageW       = user32.NewProc("PostThreadMessageW")
	pPeekMessageW             = user32.NewProc("PeekMessageW")
	pGetLastInputInfo         = user32.NewProc("GetLastInputInfo")
	pSendInput                = user32.NewProc("SendInput")
	pGetAsyncKeyState         = user32.NewProc("GetAsyncKeyState")
	pRegisterClassExW         = user32.NewProc("RegisterClassExW")
	pCreateWindowExW          = user32.NewProc("CreateWindowExW")
	pDefWindowProcW           = user32.NewProc("DefWindowProcW")
	pDestroyWindow            = user32.NewProc("DestroyWindow")
	pPostQuitMessage          = user32.NewProc("PostQuitMessage")
	pLoadIconW                = user32.NewProc("LoadIconW")
	pLoadImageW               = user32.NewProc("LoadImageW")
	pCreatePopupMenu          = user32.NewProc("CreatePopupMenu")
	pAppendMenuW              = user32.NewProc("AppendMenuW")
	pTrackPopupMenu           = user32.NewProc("TrackPopupMenu")
	pDestroyMenu              = user32.NewProc("DestroyMenu")
	pSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	pGetCursorPos             = user32.NewProc("GetCursorPos")
	pMessageBoxW              = user32.NewProc("MessageBoxW")
	pSendMessageW             = user32.NewProc("SendMessageW")
	pSetWindowTextW           = user32.NewProc("SetWindowTextW")
	pMoveWindow               = user32.NewProc("MoveWindow")
	pShowWindow               = user32.NewProc("ShowWindow")
	pGetClientRect            = user32.NewProc("GetClientRect")
	pFindWindowW              = user32.NewProc("FindWindowW")
	pPostMessageW             = user32.NewProc("PostMessageW")
	pSetFocus                 = user32.NewProc("SetFocus")
	pBeginPaint               = user32.NewProc("BeginPaint")
	pEndPaint                 = user32.NewProc("EndPaint")
	pFillRect                 = user32.NewProc("FillRect")
	pFrameRect                = user32.NewProc("FrameRect")
	pDrawTextW                = user32.NewProc("DrawTextW")
	pInvalidateRect           = user32.NewProc("InvalidateRect")
	pDrawFrameControl         = user32.NewProc("DrawFrameControl")
	pGetSysColor              = user32.NewProc("GetSysColor")
	pGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	pSetWinEventHook          = user32.NewProc("SetWinEventHook")
	pUnhookWinEvent           = user32.NewProc("UnhookWinEvent")
	pGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	pGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	pGetWindowTextW           = user32.NewProc("GetWindowTextW")

	pGetModuleHandleW           = kernel32.NewProc("GetModuleHandleW")
	pGetCurrentThreadId         = kernel32.NewProc("GetCurrentThreadId")
	pGetLastError               = kernel32.NewProc("GetLastError")
	pCreateMutexW               = kernel32.NewProc("CreateMutexW")
	pLoadLibraryW               = kernel32.NewProc("LoadLibraryW")
	pGetProcAddress             = kernel32.NewProc("GetProcAddress")
	pGetCurrentProcessId        = kernel32.NewProc("GetCurrentProcessId")
	pOpenProcess                = kernel32.NewProc("OpenProcess")
	pCloseHandle                = kernel32.NewProc("CloseHandle")
	pQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	pCreateToolhelp32Snapshot   = kernel32.NewProc("CreateToolhelp32Snapshot")
	pProcess32FirstW            = kernel32.NewProc("Process32FirstW")
	pProcess32NextW             = kernel32.NewProc("Process32NextW")
	pCoInitializeEx             = ole32.NewProc("CoInitializeEx")
	pCoUninitialize             = ole32.NewProc("CoUninitialize")
	pGetStockObject             = gdi32.NewProc("GetStockObject")
	pCreateSolidBrush           = gdi32.NewProc("CreateSolidBrush")
	pDeleteObject               = gdi32.NewProc("DeleteObject")
	pSetBkMode                  = gdi32.NewProc("SetBkMode")
	pSetTextColor               = gdi32.NewProc("SetTextColor")
	pSelectObject               = gdi32.NewProc("SelectObject")
	pCreateFontW                = gdi32.NewProc("CreateFontW")
	pShellNotifyIconW           = shell32.NewProc("Shell_NotifyIconW")
	pInitCommonControlsEx       = comctl32.NewProc("InitCommonControlsEx")
)

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }
type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}
type LASTINPUTINFO struct {
	CbSize uint32
	DwTime uint32
}

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type NOTIFYICONDATA struct {
	CbSize            uint32
	HWnd              uintptr
	UID               uint32
	UFlags            uint32
	UCallbackMessage  uint32
	HIcon             uintptr
	SzTip             [128]uint16
	DwState           uint32
	DwStateMask       uint32
	SzInfo            [256]uint16
	UTimeoutOrVersion uint32
	SzInfoTitle       [64]uint16
	DwInfoFlags       uint32
	GuidItem          [16]byte
	HBalloonIcon      uintptr
}

type PAINTSTRUCT struct {
	Hdc         uintptr
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}
type INITCOMMONCONTROLSEX struct {
	DwSize uint32
	DwICC  uint32
}
type NMHDR struct {
	HwndFrom uintptr
	IdFrom   uintptr
	Code     int32
}
type LVCOLUMNW struct {
	Mask       uint32
	Fmt        int32
	Cx         int32
	PszText    *uint16
	CchTextMax int32
	ISubItem   int32
	IImage     int32
	IOrder     int32
	CxMin      int32
	CxDefault  int32
	CxIdeal    int32
}
type NMITEMACTIVATE struct {
	Hdr       NMHDR
	IItem     int32
	ISubItem  int32
	UNewState uint32
	UOldState uint32
	UChanged  uint32
	PtAction  POINT
	LParam    uintptr
	UKeyFlags uint32
}

type NMCUSTOMDRAW struct {
	Hdr         NMHDR
	DwDrawStage uint32
	Hdc         uintptr
	Rc          RECT
	DwItemSpec  uintptr
	UItemState  uint32
	LItemlParam uintptr
}

type NMLVCUSTOMDRAW struct {
	Nmcd        NMCUSTOMDRAW
	ClrText     uint32
	ClrTextBk   uint32
	ISubItem    int32
	DwItemType  uint32
	ClrFace     uint32
	IIconEffect int32
	IIconPhase  int32
	IPartId     int32
	IStateId    int32
	RcText      RECT
	UAlign      uint32
}

type LVITEMW struct {
	Mask       uint32
	IItem      int32
	ISubItem   int32
	State      uint32
	StateMask  uint32
	PszText    *uint16
	CchTextMax int32
	IImage     int32
	LParam     uintptr
	IIndent    int32
	IGroupId   int32
	CColumns   uint32
	PuColumns  uintptr
	PiColFmt   uintptr
	IGroup     int32
}

type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}
type MSLLHOOKSTRUCT struct {
	Pt          POINT
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}
type PROCESSENTRY32W struct {
	DwSize              uint32
	CntUsage            uint32
	Th32ProcessID       uint32
	Th32DefaultHeapID   uintptr
	Th32ModuleID        uint32
	CntThreads          uint32
	Th32ParentProcessID uint32
	PcPriClassBase      int32
	DwFlags             uint32
	SzExeFile           [260]uint16
}
type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}
type INPUT struct {
	Type uint32
	_    uint32
	Data [4]uint64
}

type Config struct {
	Version         int                     `json:"Version"`
	SavedBy         string                  `json:"SavedBy,omitempty"`
	SavedAt         string                  `json:"SavedAt,omitempty"`
	ActiveProfileId string                  `json:"ActiveProfileId"`
	Profiles        []Profile               `json:"Profiles"`
	AutoSwitch      AutoSwitchConfig        `json:"AutoSwitch,omitempty"`
	Controller      ControllerFeatureConfig `json:"Controller,omitempty"`
}
type Profile struct {
	Id     string              `json:"Id"`
	Name   string              `json:"Name"`
	Rules  []Rule              `json:"Rules"`
	JoyCon JoyConProfileConfig `json:"JoyCon,omitempty"`
}
type Rule struct {
	Enabled          bool   `json:"Enabled"`
	Input            []Item `json:"Input"`
	Mode             string `json:"Mode"`
	Output           []Item `json:"Output"`
	SuppressTrigger  bool   `json:"SuppressTrigger"`
	SuppressPrefix   bool   `json:"SuppressPrefix"`
	LongPressEnabled bool   `json:"LongPressEnabled,omitempty"`
	LongPressMs      int    `json:"LongPressMs,omitempty"`
	LongPressAction  string `json:"LongPressAction,omitempty"`
	LongPressOutput  []Item `json:"LongPressOutput,omitempty"`
}
type Item struct {
	Kind string `json:"Kind"`
	Code string `json:"Code"`
}

type outputJob struct {
	Rule     Rule
	QueuedAt time.Time
}

type App struct {
	mu                 sync.RWMutex
	config             Config
	rules              []Rule
	activeProfileIndex int
	editorProfileIndex int
	enabled            bool
	emergency          bool
	startedSafe        bool
	startTray          bool
	configPath         string
	logPath            string
	iconPath           string

	mouseDown            map[string]bool
	mouseDownAt          map[string]time.Time
	keyDown              map[uint32]bool
	controllerDown       map[string]bool
	controllerPending    map[string]bool
	controllerConsumed   map[string]bool
	controllerHoldRules  map[string]Rule
	lastControllerInput  Item
	lastControllerSource string
	pendingTap           map[string]bool
	consumedPrefix       map[string]bool
	suppressedDown       map[string]bool
	longPress            map[string]*longPressState
	longPressSeq         uint64

	joyConWorker             *JoyConWorker
	joyConCancel             func()
	joyConDone               chan struct{}
	joyConStatus             JoyConConnectionStatus
	joyConOutputRefs         map[uint32]joyConOutputReference
	joyConCalibration        *JoyConCalibrationSession
	joyConCalibrationActive  bool
	joyConCalibrationMessage string
	xInputCancel             func()
	xInputDone               chan struct{}
	xInputStatus             XInputConnectionStatus

	sendMu                  sync.Mutex
	hookMu                  sync.RWMutex
	workerWG                sync.WaitGroup
	actionCh                chan outputJob
	shutdownCh              chan struct{}
	configSaveCh            chan []byte
	hookReady               chan struct{}
	hookDone                chan struct{}
	hookThreadID            atomic.Uint32
	hookEventSeq            atomic.Uint64
	mouseEventSeq           atomic.Uint64
	keyEventSeq             atomic.Uint64
	hookGeneration          atomic.Uint64
	hookReinstallCount      atomic.Uint64
	outputDropped           atomic.Uint64
	rehookPending           atomic.Bool
	shuttingDown            atomic.Bool
	hwnd                    uintptr
	ctrlList                uintptr
	ctrlStatus              uintptr
	ctrlMessage             uintptr
	ctrlLastInput           uintptr
	ctrlProfile             uintptr
	ctrlDetail              uintptr
	lblInput                uintptr
	lblOutput               uintptr
	lblMode                 uintptr
	lblFlags                uintptr
	editInput               uintptr
	editOutput              uintptr
	comboMode               uintptr
	chkEnabled              uintptr
	chkSuppressTrigger      uintptr
	chkSuppressPrefix       uintptr
	btnStartStop            uintptr
	btnEmergency            uintptr
	btnRelease              uintptr
	btnReload               uintptr
	btnOpenCfg              uintptr
	btnOpenFolder           uintptr
	btnEnableRule           uintptr
	btnDisableRule          uintptr
	btnSaveRule             uintptr
	btnAddRule              uintptr
	btnDeleteRule           uintptr
	btnMoveUp               uintptr
	btnMoveDown             uintptr
	btnMoveTop              uintptr
	btnMoveBottom           uintptr
	btnDupRule              uintptr
	btnRevertRule           uintptr
	btnTestOutput           uintptr
	btnExport               uintptr
	btnImport               uintptr
	btnCopyDiag             uintptr
	btnOpenLog              uintptr
	btnRecordInput          uintptr
	btnRecordOutput         uintptr
	btnRecordStop           uintptr
	btnDefaultRules         uintptr
	btnSafe                 uintptr
	btnQuit                 uintptr
	fontTitle               uintptr
	fontSmall               uintptr
	fontButton              uintptr
	appIcon                 uintptr
	font                    uintptr
	mouseHook               uintptr
	keyHook                 uintptr
	wndCb                   uintptr
	keyCb                   uintptr
	mouseCb                 uintptr
	logCh                   chan string
	recordingMode           string
	recordingRuleIndex      int
	recordingProfileID      string
	recordedItems           []Item
	recordHeld              map[string]bool
	lastInputText           string
	lastInputAt             time.Time
	httpAddr                string
	webviewLoading          bool
	webviewReady            bool
	webviewController       uintptr
	webviewCore             uintptr
	webviewEnvHandler       *webView2EnvCompletedHandler
	webviewCtlHandler       *webView2ControllerCompletedHandler
	webviewURL              string
	forceWebView2           bool
	currentPID              uint32
	foregroundApp           ForegroundAppInfo
	lastExternalApp         ForegroundAppInfo
	externalCandidate       ForegroundAppInfo
	externalCandidateSince  time.Time
	settingsGraceUntil      time.Time
	autoProfileID           string
	autoBindingID           string
	autoBindingName         string
	autoCandidateKey        string
	autoCandidateSince      time.Time
	autoLastSwitchAt        time.Time
	foregroundEventCh       chan uintptr
	foregroundHook          uintptr
	foregroundMonitorStatus string
	foregroundLastEventAt   time.Time
	foregroundFallbackPolls uint64
	autoDecision            string
	autoDecisionDetail      string
	autoDecisionAt          time.Time
	autoLastDecisionKey     string
	autoBlockedSince        time.Time
}

var app = &App{enabled: true, foregroundMonitorStatus: "起動準備中", mouseDown: map[string]bool{}, mouseDownAt: map[string]time.Time{}, keyDown: map[uint32]bool{}, controllerDown: map[string]bool{}, controllerPending: map[string]bool{}, controllerConsumed: map[string]bool{}, controllerHoldRules: map[string]Rule{}, pendingTap: map[string]bool{}, consumedPrefix: map[string]bool{}, suppressedDown: map[string]bool{}, longPress: map[string]*longPressState{}, joyConOutputRefs: map[uint32]joyConOutputReference{}, joyConStatus: JoyConConnectionStatus{BatteryPercent: -1}, logCh: make(chan string, 4096), recordHeld: map[string]bool{}, recordingRuleIndex: -1, actionCh: make(chan outputJob, 8192), configSaveCh: make(chan []byte, 8), shutdownCh: make(chan struct{}), hookReady: make(chan struct{}), hookDone: make(chan struct{}), foregroundEventCh: make(chan uintptr, 32)}

func main() {
	if hasCommandLineFlag("--self-test", "/self-test") {
		if err := runSelfTest(); err != nil {
			os.Exit(1)
		}
		return
	}

	runtime.LockOSThread()
	if hr, _, _ := pCoInitializeEx.Call(0, COINIT_APARTMENTTHREADED); hr != S_OK && hr != S_FALSE {
		// WebView2だけがCOM STAを必要とする。失敗しても入力変換コアは動かす。
	}
	defer pCoUninitialize.Call()
	parseArgs()
	app.initPaths()
	pid, _, _ := pGetCurrentProcessId.Call()
	app.currentPID = uint32(pid)
	if app.startedSafe {
		app.enabled = false
	}
	if !acquireSingleInstanceOrShow() {
		return
	}
	go app.logWriter()
	app.startWorkers()
	app.logf("starting %s %s core=dedicated-hook-thread+output-worker+watchdog args=%v", appName, appVersion, os.Args[1:])
	if err := app.ensureConfig(); err != nil {
		app.logf("ensure config error: %v", err)
	}
	if err := app.loadConfig(); err != nil {
		app.logf("load config error: %v", err)
		messageBox("設定読み込みエラー", fmt.Sprintf("設定を読み込めませんでした。\n%v\n\n安全のため、初期設定で停止状態起動します。", err))
		app.enabled = false
		_ = app.loadDefaultConfig()
	}
	app.startAutoSwitchWorker()
	app.syncControllerSubsystems()
	if err := app.startWebServer(); err != nil {
		app.logf("web UI init error: %v", err)
		messageBox("MouseButtonMapper", "設定画面の起動に失敗しました。\n"+err.Error())
	}
	if err := app.createMainWindow(!app.startTray); err != nil {
		app.logf("message window init error: %v", err)
		messageBox("MouseButtonMapper", "常駐ウィンドウ初期化に失敗しました。緊急停止CMDで終了できます。\n"+err.Error())
	}
	app.startHookSubsystem()
	app.messageLoop()
	app.cleanup()
}

func hasCommandLineFlag(flags ...string) bool {
	for _, arg := range os.Args[1:] {
		arg = strings.ToLower(strings.TrimSpace(arg))
		for _, flag := range flags {
			if arg == strings.ToLower(flag) {
				return true
			}
		}
	}
	return false
}

func runSelfTest() error {
	var cfg Config
	if err := json.Unmarshal([]byte(defaultConfigJSON), &cfg); err != nil {
		return fmt.Errorf("embedded default config: %w", err)
	}
	cfg = normalizeConfig(cfg)
	if len(cfg.Profiles) == 0 || strings.TrimSpace(cfg.ActiveProfileId) == "" {
		return fmt.Errorf("embedded default config has no usable profile")
	}
	if strings.TrimSpace(webHTML) == "" || !strings.Contains(webHTML, `id="autoEnabled"`) || !strings.Contains(webHTML, `id="ruleRows"`) || !strings.Contains(webHTML, `id="ruleLongEnabled"`) || !strings.Contains(webHTML, `id="recordLongOutput"`) {
		return fmt.Errorf("embedded web UI is incomplete")
	}
	probeRule := Rule{Enabled: true, Input: []Item{{Kind: "Mouse", Code: "X1"}}, Mode: "Tap", LongPressEnabled: true, LongPressMs: 500, LongPressAction: longPressActionCancel, Output: []Item{{Kind: "Key", Code: "65"}}}
	if err := validateLongPressRule(probeRule); err != nil {
		return fmt.Errorf("long press rule self-test failed: %w", err)
	}
	joyConProbe := Rule{Enabled: true, Input: []Item{{Kind: "JoyCon", Code: string(JoyConStickUp)}}, Mode: joyConRuleModeHold, Output: []Item{{Kind: "Key", Code: "W"}}}
	if err := validateJoyConHoldRule(joyConProbe); err != nil {
		return fmt.Errorf("Joy-Con hold rule self-test failed: %w", err)
	}
	xInputProbe := Rule{Enabled: true, Input: []Item{{Kind: "XInput", Code: "P1:LB"}}, Mode: joyConRuleModeHold, Output: []Item{{Kind: "Key", Code: "Q"}}}
	if err := validateJoyConHoldRule(xInputProbe); err != nil {
		return fmt.Errorf("XInput hold rule self-test failed: %w", err)
	}
	joyConReport := make([]byte, 12)
	joyConReport[0] = joyConReportFull
	joyConReport[5] = 1 << 7
	joyConReport[6] = 0xd0
	joyConReport[7] = 0x07
	joyConReport[8] = 0x7d
	if state, err := parseJoyConInputReport(joyConReport); err != nil || !state.Buttons[JoyConButtonZL] {
		return fmt.Errorf("Joy-Con report parser self-test failed: %v", err)
	}
	probe := AppBinding{Enabled: true, ProfileId: cfg.ActiveProfileId, ProcessName: "selftest.exe"}
	if !bindingMatches(probe, ForegroundAppInfo{ProcessName: "selftest"}) {
		return fmt.Errorf("automatic profile matcher self-test failed")
	}
	return nil
}

func parseArgs() {
	for _, arg := range os.Args[1:] {
		switch strings.ToLower(strings.TrimSpace(arg)) {
		case "--tray", "/tray":
			app.startTray = true
		case "--safe", "/safe":
			app.startedSafe = true
		case "--webview2", "/webview2":
			app.forceWebView2 = true
		}
	}
}

var singleInstanceMutexHandles []uintptr

func acquireSingleInstanceOrShow() bool {
	// v7.9.0: 多重起動を強めに拒否する。
	// 旧v7系のMutexも同時に押さえ、古い版との二重常駐も避ける。
	names := []string{
		"Local\\MouseButtonMapper_v7_single_instance",
		"Local\\MouseButtonMapper_single_instance",
	}
	for _, n := range names {
		name := syscall.StringToUTF16Ptr(n)
		h, _, _ := pCreateMutexW.Call(0, 1, uintptr(unsafe.Pointer(name)))
		if h == 0 {
			continue
		}
		gle, _, _ := pGetLastError.Call()
		if gle == 183 {
			signalExistingInstance()
			return false
		}
		singleInstanceMutexHandles = append(singleInstanceMutexHandles, h)
	}
	return true
}

func signalExistingInstance() {
	classes := []string{
		mainWindowClass,
		"MouseButtonMapperMainWindow_v810",
		"MouseButtonMapperMainWindow_v790",
		"MouseButtonMapperMainWindow_v780",
		"MouseButtonMapperMainWindow_v770",
		"MouseButtonMapperMainWindow_v760",
	}
	for _, c := range classes {
		cls := syscall.StringToUTF16Ptr(c)
		hwnd, _, _ := pFindWindowW.Call(uintptr(unsafe.Pointer(cls)), 0)
		if hwnd != 0 {
			pPostMessageW.Call(hwnd, showMsg, 0, 0)
			pShowWindow.Call(hwnd, SW_RESTORE)
			pSetForegroundWindow.Call(hwnd)
			return
		}
	}
	messageBox("MouseButtonMapper", "MouseButtonMapper はすでに起動しています。")
}

func (a *App) initPaths() {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		if d, err := os.UserConfigDir(); err == nil {
			base = d
		} else {
			base = "."
		}
	}
	dir := filepath.Join(base, appName)
	_ = os.MkdirAll(dir, 0755)
	a.configPath = filepath.Join(dir, "config.json")
	a.logPath = filepath.Join(dir, "MouseButtonMapper.log")
	if exe, err := os.Executable(); err == nil {
		a.iconPath = filepath.Join(filepath.Dir(exe), "MouseButtonMapper.ico")
	}
}

func (a *App) logWriter() {
	for msg := range a.logCh {
		f, err := os.OpenFile(a.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			_, _ = f.WriteString(time.Now().Format("2006-01-02 15:04:05.000 ") + msg + "\r\n")
			_ = f.Close()
		}
	}
}
func (a *App) logf(format string, args ...any) {
	select {
	case a.logCh <- fmt.Sprintf(format, args...):
	default:
	}
}

func (a *App) startWorkers() {
	a.workerWG.Add(2)
	go a.outputWorker()
	go a.configWriter()
}

func (a *App) startAutoSwitchWorker() {
	a.workerWG.Add(1)
	go a.autoSwitchWorker()
}

const processQueryLimitedInformation = 0x1000

var winEventCallback = syscall.NewCallback(winEventProc)

func winEventProc(hWinEventHook uintptr, event uintptr, hwnd uintptr, idObject uintptr, idChild uintptr, idEventThread uintptr, eventTime uintptr) uintptr {
	if event != EVENT_SYSTEM_FOREGROUND || hwnd == 0 {
		return 0
	}
	select {
	case app.foregroundEventCh <- hwnd:
	default:
		// Foreground changes can arrive in bursts. Keep the newest one rather
		// than blocking the WinEvent callback thread.
		select {
		case <-app.foregroundEventCh:
		default:
		}
		select {
		case app.foregroundEventCh <- hwnd:
		default:
		}
	}
	return 0
}

func foregroundWindowText(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	n, _, _ := pGetWindowTextLengthW.Call(hwnd)
	if n == 0 {
		return ""
	}
	buf := make([]uint16, int(n)+1)
	got, _, _ := pGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if got == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:int(got)])
}

func processImagePath(pid uint32) string {
	if pid == 0 {
		return ""
	}
	h, _, _ := pOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer pCloseHandle.Call(h)
	buf := make([]uint16, 32768)
	sz := uint32(len(buf))
	r, _, _ := pQueryFullProcessImageNameW.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&sz)))
	if r == 0 || sz == 0 || int(sz) > len(buf) {
		return ""
	}
	return syscall.UTF16ToString(buf[:sz])
}

func processNameFromSnapshot(pid uint32) string {
	const th32csSnapProcess = 0x00000002
	h, _, _ := pCreateToolhelp32Snapshot.Call(th32csSnapProcess, 0)
	if h == 0 || h == ^uintptr(0) {
		return ""
	}
	defer pCloseHandle.Call(h)
	var pe PROCESSENTRY32W
	pe.DwSize = uint32(unsafe.Sizeof(pe))
	r, _, _ := pProcess32FirstW.Call(h, uintptr(unsafe.Pointer(&pe)))
	for r != 0 {
		if pe.Th32ProcessID == pid {
			return syscall.UTF16ToString(pe.SzExeFile[:])
		}
		r, _, _ = pProcess32NextW.Call(h, uintptr(unsafe.Pointer(&pe)))
	}
	return ""
}

func readForegroundAppFromHWND(hwnd uintptr, previous ForegroundAppInfo, source string) ForegroundAppInfo {
	if hwnd == 0 {
		hwnd, _, _ = pGetForegroundWindow.Call()
	}
	if hwnd == 0 {
		return ForegroundAppInfo{}
	}
	var pid uint32
	pGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	info := ForegroundAppInfo{HWND: hwnd, PID: pid, Title: foregroundWindowText(hwnd), Source: source, SeenAt: time.Now()}
	if previous.PID == pid && previous.Path != "" {
		info.Path = previous.Path
		info.ProcessName = previous.ProcessName
	} else {
		info.Path = processImagePath(pid)
		info.ProcessName = baseNameAnySeparator(info.Path)
		if info.ProcessName == "" {
			info.ProcessName = processNameFromSnapshot(pid)
		}
	}
	return info
}

func readForegroundApp(previous ForegroundAppInfo) ForegroundAppInfo {
	return readForegroundAppFromHWND(0, previous, "定期確認")
}

func (a *App) isSettingsWindowLocked(info ForegroundAppInfo) bool {
	if info.PID != 0 && info.PID == a.currentPID {
		return true
	}
	return strings.Contains(strings.ToLower(info.Title), strings.ToLower(appName))
}

func pumpThreadMessages() {
	var msg MSG
	for {
		r, _, _ := pPeekMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, PM_REMOVE)
		if r == 0 {
			return
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (a *App) setForegroundMonitorStatus(status string) {
	a.mu.Lock()
	a.foregroundMonitorStatus = status
	a.mu.Unlock()
}

func (a *App) autoSwitchWorker() {
	defer a.workerWG.Done()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hook, _, hookErr := pSetWinEventHook.Call(
		EVENT_SYSTEM_FOREGROUND,
		EVENT_SYSTEM_FOREGROUND,
		0,
		winEventCallback,
		0,
		0,
		WINEVENT_OUTOFCONTEXT|WINEVENT_SKIPOWNPROCESS,
	)
	if hook != 0 {
		a.mu.Lock()
		a.foregroundHook = hook
		a.foregroundMonitorStatus = "WinEventHook（前面切替イベント）+ 1秒ごとの確認"
		a.mu.Unlock()
		a.logf("foreground monitor installed: WinEventHook=%x", hook)
		defer func() {
			pUnhookWinEvent.Call(hook)
			a.mu.Lock()
			a.foregroundHook = 0
			a.mu.Unlock()
		}()
	} else {
		status := fmt.Sprintf("1秒ごとの確認のみ（WinEventHook失敗: %v）", hookErr)
		a.setForegroundMonitorStatus(status)
		a.logf("foreground monitor fallback: %s", status)
	}

	pumpTicker := time.NewTicker(40 * time.Millisecond)
	fallbackTicker := time.NewTicker(1 * time.Second)
	defer pumpTicker.Stop()
	defer fallbackTicker.Stop()

	var previous ForegroundAppInfo
	capture := func(hwnd uintptr, source string) {
		info := readForegroundAppFromHWND(hwnd, previous, source)
		if !info.Valid() {
			return
		}
		previous = info
		a.mu.Lock()
		if source == "WinEventHook" {
			a.foregroundLastEventAt = info.SeenAt
		} else {
			a.foregroundFallbackPolls++
		}
		a.mu.Unlock()
		a.evaluateForegroundApp(info)
	}

	capture(0, "起動時確認")
	for {
		select {
		case <-a.shutdownCh:
			return
		case <-pumpTicker.C:
			pumpThreadMessages()
			var newest uintptr
		drainEvents:
			for {
				select {
				case newest = <-a.foregroundEventCh:
				default:
					break drainEvents
				}
			}
			if newest != 0 {
				capture(newest, "WinEventHook")
			}
		case <-fallbackTicker.C:
			capture(0, "定期確認")
		}
	}
}

func (a *App) evaluateBestKnownForeground() {
	a.mu.RLock()
	info := a.foregroundApp
	if a.isSettingsWindowLocked(info) && a.lastExternalApp.Valid() {
		info = a.lastExternalApp
	}
	a.mu.RUnlock()
	if !info.Valid() {
		info = readForegroundApp(ForegroundAppInfo{})
	}
	if info.Valid() {
		a.evaluateForegroundApp(info)
	}
}

func (a *App) updateAutoDecisionLocked(key, summary, detail string, now time.Time) {
	a.autoDecision = summary
	a.autoDecisionDetail = detail
	a.autoDecisionAt = now
	if key != a.autoLastDecisionKey {
		a.autoLastDecisionKey = key
		a.logf("auto decision: %s detail=%s", summary, detail)
	}
}

func (a *App) scheduleAutoRecheck(delay time.Duration) {
	if delay < 10*time.Millisecond {
		delay = 10 * time.Millisecond
	}
	time.AfterFunc(delay, func() {
		select {
		case <-a.shutdownCh:
			return
		default:
			a.evaluateBestKnownForeground()
		}
	})
}

func (a *App) evaluateForegroundApp(observed ForegroundAppInfo) {
	if !observed.Valid() {
		return
	}
	now := time.Now()
	a.mu.Lock()
	a.foregroundApp = observed

	isSettings := a.isSettingsWindowLocked(observed)
	if !isSettings && now.Before(a.settingsGraceUntil) {
		p := strings.ToLower(observed.ProcessName)
		isSettings = p == "msedge.exe" || p == "chrome.exe" || p == "firefox.exe"
	}

	matchInfo, usingLastExternal := foregroundMatchTarget(observed, a.lastExternalApp, isSettings)
	if isSettings {
		a.externalCandidate = ForegroundAppInfo{}
		a.externalCandidateSince = time.Time{}
		if !matchInfo.Valid() {
			a.updateAutoDecisionLocked("settings-no-history", "設定画面を表示中", "判定に使える直前のアプリがまだありません。対象アプリを一度前面に出してください。", now)
			a.mu.Unlock()
			return
		}
		// The settings UI itself must not cancel the app profile. Continue to
		// evaluate the last non-settings foreground app on every monitor tick.
		// This is also essential when the user enables auto switching while the
		// settings window is open: the debounce can then complete normally.
	} else {
		// A non-settings foreground window is already a valid observation. Keep it
		// immediately for the capture UI. Profile application uses the independent
		// DebounceMs candidate timer below.
		a.lastExternalApp = observed
		a.externalCandidate = observed
		a.externalCandidateSince = now
	}

	contextDetail := ""
	if usingLastExternal {
		contextDetail = fmt.Sprintf("設定画面を表示中のため、直前のアプリ %s を判定対象にしています。 ", matchInfo.ProcessName)
	}

	if a.recordingMode != "" {
		a.updateAutoDecisionLocked("recording", "入力記録中のため切替を保留", contextDetail+"記録終了後に再判定します。", now)
		a.mu.Unlock()
		return
	}
	if !a.config.AutoSwitch.Enabled {
		a.updateAutoDecisionLocked("disabled", "自動切替は無効", "「アプリに応じて自動切替する」をオンにすると判定を開始します。", now)
		if a.autoProfileID != "" || a.autoBindingID != "" {
			a.reconcilePhysicalStateLocked()
			if !a.physicalInputIdleLocked() {
				a.mu.Unlock()
				return
			}
			oldName := a.activeProfileNameLocked()
			a.clearAutoMatchLocked()
			a.rebuildRulesLocked()
			newName := a.activeProfileNameLocked()
			a.postUIRefreshLocked()
			a.mu.Unlock()
			a.logf("auto profile disabled: %s -> %s", oldName, newName)
			return
		}
		a.mu.Unlock()
		return
	}

	idx, binding, matched := firstMatchingBinding(a.config.AutoSwitch.Bindings, matchInfo, a.profileExistsLocked)
	candidateKey := "none"
	candidateProfile := ""
	candidateName := ""
	if matched {
		candidateKey = binding.Id + "|" + binding.ProfileId
		candidateProfile = binding.ProfileId
		candidateName = binding.Name
		detail := contextDetail + fmt.Sprintf("優先順位 %d「%s」が %s に一致しました。", idx+1, binding.Name, matchInfo.ProcessName)
		a.updateAutoDecisionLocked("match:"+candidateKey, "一致: "+binding.Name, detail, now)
	} else {
		detail := contextDetail + fmt.Sprintf("%s に一致する有効な自動切替ルールはありません。", matchInfo.ProcessName)
		a.updateAutoDecisionLocked("nomatch:"+strings.ToLower(matchInfo.ProcessName)+"|"+matchInfo.Title, "一致なし: 通常時プロファイルを使用", detail, now)
	}

	debounce := time.Duration(a.config.AutoSwitch.DebounceMs) * time.Millisecond
	if debounce < 0 {
		debounce = 0
	}
	if candidateKey != a.autoCandidateKey {
		a.autoCandidateKey = candidateKey
		a.autoCandidateSince = now
		a.autoBlockedSince = time.Time{}
		a.mu.Unlock()
		a.scheduleAutoRecheck(debounce + 20*time.Millisecond)
		return
	}
	if now.Sub(a.autoCandidateSince) < debounce {
		remaining := debounce - now.Sub(a.autoCandidateSince)
		a.mu.Unlock()
		a.scheduleAutoRecheck(remaining + 20*time.Millisecond)
		return
	}
	if a.autoBindingID == binding.Id && a.autoProfileID == candidateProfile {
		a.mu.Unlock()
		return
	}

	a.reconcilePhysicalStateLocked()
	if !a.physicalInputIdleLocked() {
		if a.autoBlockedSince.IsZero() {
			a.autoBlockedSince = now
		}
		if !strings.Contains(a.autoDecisionDetail, "すべて離された直後") {
			a.autoDecisionDetail += " キーまたはマウスボタンが押されているため、すべて離された直後に適用します。"
		}
		a.mu.Unlock()
		a.scheduleAutoRecheck(100 * time.Millisecond)
		return
	}

	a.autoBlockedSince = time.Time{}
	oldName := a.activeProfileNameLocked()
	if matched {
		a.autoProfileID = candidateProfile
		a.autoBindingID = binding.Id
		a.autoBindingName = candidateName
	} else {
		a.autoProfileID = ""
		a.autoBindingID = ""
		a.autoBindingName = ""
	}
	a.autoLastSwitchAt = now
	a.rebuildRulesLocked()
	newName := a.activeProfileNameLocked()
	a.postUIRefreshLocked()
	a.mu.Unlock()

	if matched {
		a.logf("auto profile switch: %s -> %s priority=%d binding=%q process=%q title=%q path=%q source=%q", oldName, newName, idx+1, candidateName, matchInfo.ProcessName, matchInfo.Title, matchInfo.Path, matchInfo.Source)
	} else {
		a.logf("auto profile fallback: %s -> %s process=%q title=%q source=%q", oldName, newName, matchInfo.ProcessName, matchInfo.Title, matchInfo.Source)
	}
}

func (a *App) outputWorker() {
	defer a.workerWG.Done()
	for {
		select {
		case <-a.shutdownCh:
			return
		default:
		}
		select {
		case <-a.shutdownCh:
			return
		case job := <-a.actionCh:
			if delay := time.Since(job.QueuedAt); delay > 100*time.Millisecond {
				a.logf("output queue delay: %s input=%s output=%s", delay.Round(time.Millisecond), itemsText(job.Rule.Input), itemsText(job.Rule.Output))
			}
			a.sendRule(job.Rule)
		}
	}
}

func (a *App) configWriter() {
	defer a.workerWG.Done()
	for data := range a.configSaveCh {
		// 連続操作時は最新状態だけを書き込む。各状態は完全なconfig.jsonなので中間を省いても情報は失われない。
	drain:
		for {
			select {
			case newer, ok := <-a.configSaveCh:
				if !ok {
					break drain
				}
				data = newer
			default:
				break drain
			}
		}
		if err := a.backupConfig(); err != nil {
			a.logf("backup config failed: %v", err)
		}
		tmp := a.configPath + ".tmp"
		if err := os.WriteFile(tmp, data, 0644); err != nil {
			a.logf("config temp write failed: %v", err)
			continue
		}
		if err := os.Rename(tmp, a.configPath); err != nil {
			// Windows上で置換Renameが拒否された場合は、従来と同じ直接書き込みへ退避する。
			if err2 := os.WriteFile(a.configPath, data, 0644); err2 != nil {
				a.logf("config write failed: rename=%v direct=%v", err, err2)
			}
			_ = os.Remove(tmp)
		}
	}
}

func cloneRule(r Rule) Rule {
	r.Input = append([]Item(nil), r.Input...)
	r.Output = append([]Item(nil), r.Output...)
	r.LongPressOutput = append([]Item(nil), r.LongPressOutput...)
	return r
}

func (a *App) enqueueRule(r Rule) bool {
	if a.shuttingDown.Load() {
		return false
	}
	job := outputJob{Rule: cloneRule(r), QueuedAt: time.Now()}
	select {
	case a.actionCh <- job:
		return true
	default:
		n := a.outputDropped.Add(1)
		a.logf("output queue full: dropped=%d input=%s output=%s", n, itemsText(r.Input), itemsText(r.Output))
		return false
	}
}

func (a *App) ensureConfig() error {
	if _, err := os.Stat(a.configPath); err == nil {
		return nil
	}
	return os.WriteFile(a.configPath, []byte(defaultConfigJSON), 0644)
}
func stripBOM(b []byte) []byte { return bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF}) }
func (a *App) loadDefaultConfig() error {
	var cfg Config
	if err := json.Unmarshal([]byte(defaultConfigJSON), &cfg); err != nil {
		return err
	}
	a.applyConfig(cfg)
	return nil
}
func (a *App) loadConfig() error {
	b, err := os.ReadFile(a.configPath)
	if err != nil {
		return err
	}
	var cfg Config
	if err := json.Unmarshal(stripBOM(b), &cfg); err != nil {
		return err
	}
	a.applyConfig(cfg)
	return nil
}

func normalizeConfig(cfg Config) Config {
	if len(cfg.Profiles) == 0 {
		// Do not call mustDefaultConfig here: it calls normalizeConfig itself.
		// Parse the embedded default directly so malformed/empty imports cannot
		// recurse through the normalizer.
		var fallback Config
		if err := json.Unmarshal([]byte(defaultConfigJSON), &fallback); err == nil && len(fallback.Profiles) > 0 {
			cfg = fallback
		} else {
			cfg.Profiles = []Profile{{Id: "default", Name: "既定", Rules: []Rule{}}}
			cfg.ActiveProfileId = "default"
		}
	}
	if cfg.Version < 10 {
		cfg.Version = 10
	}
	seenProfiles := map[string]bool{}
	for i := range cfg.Profiles {
		id := strings.TrimSpace(cfg.Profiles[i].Id)
		if id == "" || seenProfiles[id] {
			id = fmt.Sprintf("profile_%d_%d", time.Now().UnixNano(), i)
		}
		seenProfiles[id] = true
		cfg.Profiles[i].Id = id
		if strings.TrimSpace(cfg.Profiles[i].Name) == "" {
			cfg.Profiles[i].Name = fmt.Sprintf("プロファイル %d", i+1)
		}
		cfg.Profiles[i].JoyCon = normalizeJoyConProfileConfig(cfg.Profiles[i].JoyCon)
		for j := range cfg.Profiles[i].Rules {
			r := &cfg.Profiles[i].Rules[j]
			r.Mode = normalizeJoyConRuleMode(r.Mode)
			if r.LongPressEnabled {
				r.LongPressMs = normalizeLongPressMs(r.LongPressMs)
				r.LongPressAction = normalizeLongPressAction(r.LongPressAction)
				if r.LongPressAction == longPressActionCancel {
					r.LongPressOutput = nil
				}
			}
		}
	}
	if !seenProfiles[cfg.ActiveProfileId] {
		cfg.ActiveProfileId = cfg.Profiles[0].Id
	}
	if cfg.AutoSwitch.DebounceMs <= 0 {
		cfg.AutoSwitch.DebounceMs = 350
	} else if cfg.AutoSwitch.DebounceMs < 50 {
		cfg.AutoSwitch.DebounceMs = 50
	}
	if cfg.AutoSwitch.DebounceMs > 3000 {
		cfg.AutoSwitch.DebounceMs = 3000
	}
	seenBindings := map[string]bool{}
	for i := range cfg.AutoSwitch.Bindings {
		b := &cfg.AutoSwitch.Bindings[i]
		b.Id = strings.TrimSpace(b.Id)
		if b.Id == "" || seenBindings[b.Id] {
			b.Id = fmt.Sprintf("binding_%d_%d", time.Now().UnixNano(), i)
		}
		seenBindings[b.Id] = true
		b.Name = strings.TrimSpace(b.Name)
		b.ProfileId = strings.TrimSpace(b.ProfileId)
		b.ProcessName = baseNameAnySeparator(b.ProcessName)
		b.TitleContains = strings.TrimSpace(b.TitleContains)
		b.PathContains = strings.TrimSpace(b.PathContains)
		if b.Name == "" {
			b.Name = baseNameAnySeparator(b.ProcessName)
			if b.Name == "" {
				b.Name = fmt.Sprintf("アプリ設定 %d", i+1)
			}
		}
		if !seenProfiles[b.ProfileId] {
			b.Enabled = false
		}
	}
	return cfg
}

func (a *App) applyConfig(cfg Config) {
	a.mu.Lock()
	a.abortAllLongPressLocked("configuration reloaded", false)
	a.clearJoyConInputStateLocked("configuration reloaded")
	cfg = normalizeConfig(cfg)
	a.config = cfg
	a.editorProfileIndex = a.profileIndexByIDLocked(cfg.ActiveProfileId)
	if !cfg.AutoSwitch.Enabled || a.profileIndexByIDLocked(a.autoProfileID) < 0 {
		a.clearAutoMatchLocked()
	}
	a.rebuildRulesLocked()
	a.requestJoyConRescanLocked()
	a.logf("loaded config: effective=%s base=%s rules=%d auto=%v bindings=%d controller=%v", a.activeProfileNameLocked(), a.baseProfileNameLocked(), len(a.rules), cfg.AutoSwitch.Enabled, len(cfg.AutoSwitch.Bindings), cfg.Controller.Enabled)
	a.postUIRefreshLocked()
	a.mu.Unlock()
	a.releaseJoyConHeldOutputs()
	a.syncControllerSubsystems()
}

func mustDefaultConfig() Config {
	var cfg Config
	_ = json.Unmarshal([]byte(defaultConfigJSON), &cfg)
	return normalizeConfig(cfg)
}

// physicalInputIdleLocked reports whether it is safe to replace the active
// rule set. Switching profiles in the middle of a held mouse chord could make
// the corresponding UP event use a different profile and either fire the
// wrong single-button action or leave a suppressed button in an odd state.
// Automatic switching therefore waits for a clean input boundary.
func mouseButtonVK(btn string) uint32 {
	switch btn {
	case "Left":
		return VK_LBUTTON
	case "Right":
		return VK_RBUTTON
	case "Middle":
		return VK_MBUTTON
	case "X1":
		return VK_XBUTTON1
	case "X2":
		return VK_XBUTTON2
	}
	return 0
}

func asyncKeyDown(vk uint32) bool {
	if vk == 0 {
		return false
	}
	r, _, _ := pGetAsyncKeyState.Call(uintptr(vk))
	return uint16(r)&0x8000 != 0
}

// reconcilePhysicalStateLocked repairs a missed UP event before a profile
// switch. The low-level hooks remain the primary source of state; this is only
// a boundary check so one stale map entry cannot block automatic switching
// forever.
func (a *App) reconcilePhysicalStateLocked() {
	for btn := range a.mouseDown {
		if !asyncKeyDown(mouseButtonVK(btn)) {
			a.abortLongPressForTriggerLocked(Item{Kind: "Mouse", Code: btn}, "missed mouse-up repaired")
			delete(a.longPress, longPressKey(Item{Kind: "Mouse", Code: btn}))
			delete(a.mouseDown, btn)
			delete(a.mouseDownAt, btn)
			delete(a.pendingTap, btn)
			delete(a.consumedPrefix, btn)
			delete(a.suppressedDown, btn)
		}
	}
	for vk := range a.keyDown {
		if vk == VK_CONTROL || vk == VK_SHIFT || vk == VK_MENU {
			continue
		}
		if !asyncKeyDown(vk) {
			a.abortLongPressForTriggerLocked(Item{Kind: "Key", Code: strconv.Itoa(int(vk))}, "missed key-up repaired")
			delete(a.longPress, longPressKey(Item{Kind: "Key", Code: strconv.Itoa(int(vk))}))
			delete(a.keyDown, vk)
		}
	}
	if !asyncKeyDown(VK_LCONTROL) && !asyncKeyDown(VK_RCONTROL) {
		delete(a.keyDown, VK_CONTROL)
		delete(a.keyDown, VK_LCONTROL)
		delete(a.keyDown, VK_RCONTROL)
	}
	if !asyncKeyDown(VK_LSHIFT) && !asyncKeyDown(VK_RSHIFT) {
		delete(a.keyDown, VK_SHIFT)
		delete(a.keyDown, VK_LSHIFT)
		delete(a.keyDown, VK_RSHIFT)
	}
	if !asyncKeyDown(VK_LMENU) && !asyncKeyDown(VK_RMENU) {
		delete(a.keyDown, VK_MENU)
		delete(a.keyDown, VK_LMENU)
		delete(a.keyDown, VK_RMENU)
	}
}

func (a *App) physicalInputIdleLocked() bool {
	return len(a.mouseDown) == 0 && len(a.keyDown) == 0 && len(a.controllerDown) == 0
}

func (a *App) profileIndexByIDLocked(id string) int {
	for i := range a.config.Profiles {
		if a.config.Profiles[i].Id == id {
			return i
		}
	}
	return -1
}

func (a *App) profileExistsLocked(id string) bool {
	return a.profileIndexByIDLocked(id) >= 0
}

func (a *App) effectiveProfileIDLocked() string {
	if a.config.AutoSwitch.Enabled && a.autoProfileID != "" && a.profileExistsLocked(a.autoProfileID) {
		return a.autoProfileID
	}
	if a.profileExistsLocked(a.config.ActiveProfileId) {
		return a.config.ActiveProfileId
	}
	if len(a.config.Profiles) > 0 {
		return a.config.Profiles[0].Id
	}
	return ""
}

func (a *App) rebuildRulesLocked() {
	a.rebuildRulesLockedWithJoyConRescan(true)
}

func (a *App) rebuildRulesWithoutJoyConRescanLocked() {
	a.rebuildRulesLockedWithJoyConRescan(false)
}

func (a *App) rebuildRulesLockedWithJoyConRescan(rescanJoyCon bool) {
	idx := a.profileIndexByIDLocked(a.effectiveProfileIDLocked())
	if idx < 0 {
		idx = 0
	}
	if idx >= len(a.config.Profiles) {
		a.activeProfileIndex = 0
		a.rules = nil
		if rescanJoyCon {
			a.requestJoyConRescanLocked()
		}
		return
	}
	a.activeProfileIndex = idx
	prof := a.config.Profiles[idx]
	rules := make([]Rule, 0, len(prof.Rules))
	for _, r := range prof.Rules {
		r.Mode = normalizeJoyConRuleMode(r.Mode)
		if !r.Enabled || len(r.Input) == 0 || !ruleHasRunnableOutput(r) || isDangerousSingleReplacement(r) {
			continue
		}
		if !a.config.Controller.Enabled && ruleUsesControllerInput(r) {
			continue
		}
		if err := validateJoyConHoldRule(r); err != nil {
			continue
		}
		if err := validateLongPressRule(r); err != nil {
			continue
		}
		rules = append(rules, r)
	}
	a.rules = rules
	if rescanJoyCon {
		a.requestJoyConRescanLocked()
	}
}

func (a *App) activeProfileNameLocked() string {
	if len(a.config.Profiles) == 0 || a.activeProfileIndex < 0 || a.activeProfileIndex >= len(a.config.Profiles) {
		return "(なし)"
	}
	return a.config.Profiles[a.activeProfileIndex].Name
}

func (a *App) baseProfileNameLocked() string {
	idx := a.profileIndexByIDLocked(a.config.ActiveProfileId)
	if idx < 0 {
		return "(なし)"
	}
	return a.config.Profiles[idx].Name
}

func (a *App) editorProfileNameLocked() string {
	if a.editorProfileIndex < 0 || a.editorProfileIndex >= len(a.config.Profiles) {
		return "(なし)"
	}
	return a.config.Profiles[a.editorProfileIndex].Name
}

func (a *App) clearAutoMatchLocked() {
	a.autoProfileID = ""
	a.autoBindingID = ""
	a.autoBindingName = ""
	a.autoCandidateKey = ""
	a.autoCandidateSince = time.Time{}
}

func isDangerousSingleReplacement(r Rule) bool {
	if len(r.Input) != 1 {
		return false
	}
	it := r.Input[0]
	if strings.EqualFold(it.Kind, "Mouse") {
		c := normMouse(it.Code)
		return c == "Left" || c == "Right" || c == "Middle"
	}
	if strings.EqualFold(it.Kind, "Key") {
		vk, ok := parseVK(it.Code)
		if !ok {
			return false
		}
		switch vk {
		case VK_RETURN, VK_SPACE, VK_TAB, VK_ESCAPE, VK_BACK:
			return true
		}
	}
	return false
}

func (a *App) startHookSubsystem() {
	go a.hookThread()
	select {
	case <-a.hookReady:
	case <-time.After(3 * time.Second):
		a.logf("hook thread startup timeout")
		messageBox("MouseButtonMapper", "入力フック専用スレッドの起動がタイムアウトしました。")
	}
	go a.hookWatchdog()
}

func (a *App) hookThread() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	tid, _, _ := pGetCurrentThreadId.Call()
	a.hookThreadID.Store(uint32(tid))
	// PostThreadMessageを確実に受け取れるよう、このスレッドのメッセージキューを先に作る。
	var dummy MSG
	pPeekMessageW.Call(uintptr(unsafe.Pointer(&dummy)), 0, 0, 0, PM_NOREMOVE)
	a.keyCb = syscall.NewCallback(keyboardProc)
	a.mouseCb = syscall.NewCallback(mouseProc)
	a.installHooksCurrent("startup")
	close(a.hookReady)

	var msg MSG
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		if msg.Message == hookReinstallMsg {
			a.reinstallHooksCurrent("watchdog detected missing callbacks")
			continue
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	a.uninstallHooksCurrent()
	close(a.hookDone)
}

func (a *App) installHooksCurrent(reason string) {
	hmod, _, _ := pGetModuleHandleW.Call(0)
	mh, _, e1 := pSetWindowsHookExW.Call(uintptr(WH_MOUSE_LL), a.mouseCb, hmod, 0)
	kh, _, e2 := pSetWindowsHookExW.Call(uintptr(WH_KEYBOARD_LL), a.keyCb, hmod, 0)
	a.hookMu.Lock()
	a.mouseHook, a.keyHook = mh, kh
	a.hookMu.Unlock()
	gen := a.hookGeneration.Add(1)
	if mh == 0 {
		a.logf("mouse hook failed: reason=%s error=%v", reason, e1)
	}
	if kh == 0 {
		a.logf("keyboard hook failed: reason=%s error=%v", reason, e2)
	}
	a.logf("hooks installed generation=%d reason=%s mouse=%x keyboard=%x", gen, reason, mh, kh)
	a.postUIRefresh()
}

func (a *App) uninstallHooksCurrent() {
	a.hookMu.Lock()
	mh, kh := a.mouseHook, a.keyHook
	a.mouseHook, a.keyHook = 0, 0
	a.hookMu.Unlock()
	if mh != 0 {
		pUnhookWindowsHookEx.Call(mh)
	}
	if kh != 0 {
		pUnhookWindowsHookEx.Call(kh)
	}
}

func (a *App) reinstallHooksCurrent(reason string) {
	a.rehookPending.Store(false)
	a.hookReinstallCount.Add(1)
	a.logf("rehooking: %s", reason)
	a.uninstallHooksCurrent()
	a.mu.Lock()
	a.mouseDown = map[string]bool{}
	a.mouseDownAt = map[string]time.Time{}
	a.keyDown = map[uint32]bool{}
	a.pendingTap = map[string]bool{}
	a.consumedPrefix = map[string]bool{}
	a.suppressedDown = map[string]bool{}
	a.abortAllLongPressLocked("hooks reinstalled", true)
	a.clearJoyConInputStateLocked("hooks reinstalled")
	a.requestJoyConRescanLocked()
	if a.recordingMode != "" {
		a.recordingMode = ""
		a.recordingRuleIndex = -1
		a.recordingProfileID = ""
		a.recordedItems = nil
		a.recordHeld = map[string]bool{}
		a.logf("recording cancelled because hooks were reinstalled")
	}
	a.mu.Unlock()
	a.releaseJoyConHeldOutputs()
	time.Sleep(20 * time.Millisecond)
	a.installHooksCurrent(reason)
}

func (a *App) requestHookReinstall(reason string) {
	if !a.rehookPending.CompareAndSwap(false, true) {
		return
	}
	tid := a.hookThreadID.Load()
	if tid == 0 {
		a.rehookPending.Store(false)
		a.logf("cannot request rehook: hook thread id is zero")
		return
	}
	r, _, e := pPostThreadMessageW.Call(uintptr(tid), hookReinstallMsg, 0, 0)
	if r == 0 {
		a.rehookPending.Store(false)
		a.logf("PostThreadMessage(rehook) failed: reason=%s error=%v", reason, e)
		return
	}
	a.logf("rehook requested: %s", reason)
}

func (a *App) hookHealthText() string {
	a.hookMu.RLock()
	mh, kh := a.mouseHook, a.keyHook
	a.hookMu.RUnlock()
	return fmt.Sprintf("MouseHook:%v KeyboardHook:%v Gen:%d Rehook:%d Drop:%d", mh != 0, kh != 0, a.hookGeneration.Load(), a.hookReinstallCount.Load(), a.outputDropped.Load())
}

func getLastInputTick() (uint32, bool) {
	li := LASTINPUTINFO{CbSize: uint32(unsafe.Sizeof(LASTINPUTINFO{}))}
	r, _, _ := pGetLastInputInfo.Call(uintptr(unsafe.Pointer(&li)))
	return li.DwTime, r != 0
}

func (a *App) hookWatchdog() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	lastInput, _ := getLastInputTick()
	lastAnySeq := a.hookEventSeq.Load()
	lastMouseSeq := a.mouseEventSeq.Load()
	var lastCursor POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&lastCursor)))
	lastRequest := time.Time{}
	for {
		select {
		case <-a.hookDone:
			return
		case <-ticker.C:
		}
		a.hookMu.RLock()
		mh, kh := a.mouseHook, a.keyHook
		a.hookMu.RUnlock()
		if (mh == 0 || kh == 0) && time.Since(lastRequest) > 5*time.Second {
			lastRequest = time.Now()
			a.requestHookReinstall("hook handle is zero")
		}

		curInput, ok := getLastInputTick()
		curAnySeq := a.hookEventSeq.Load()
		curMouseSeq := a.mouseEventSeq.Load()
		var curCursor POINT
		pGetCursorPos.Call(uintptr(unsafe.Pointer(&curCursor)))
		cursorMoved := curCursor != lastCursor
		missingAll := ok && curInput != lastInput && curAnySeq == lastAnySeq
		missingMouse := cursorMoved && curMouseSeq == lastMouseSeq
		if (missingAll || missingMouse) && time.Since(lastRequest) > 5*time.Second {
			// OS側では入力またはカーソル移動が進んだのに、当該フックのコールバックが進んでいない。
			// タイムアウトによる無言解除を短く再確認してから再フックする。
			time.Sleep(80 * time.Millisecond)
			stillMissingAll := missingAll && a.hookEventSeq.Load() == curAnySeq
			stillMissingMouse := missingMouse && a.mouseEventSeq.Load() == curMouseSeq
			if stillMissingAll || stillMissingMouse {
				lastRequest = time.Now()
				reason := "system input advanced but hook callbacks did not"
				if stillMissingMouse {
					reason = "cursor moved but mouse hook callback did not"
				}
				a.requestHookReinstall(reason)
			}
		}
		lastInput = curInput
		lastAnySeq = a.hookEventSeq.Load()
		lastMouseSeq = a.mouseEventSeq.Load()
		lastCursor = curCursor
	}
}

func (a *App) stopHookSubsystem() {
	tid := a.hookThreadID.Load()
	if tid != 0 {
		pPostThreadMessageW.Call(uintptr(tid), WM_QUIT, 0, 0)
	}
	select {
	case <-a.hookDone:
	case <-time.After(2 * time.Second):
		a.logf("hook thread shutdown timeout")
	}
}

func (a *App) messageLoop() {
	var msg MSG
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
func (a *App) cleanup() {
	a.stopXInputSubsystem()
	a.stopJoyConSubsystem()
	// タイマーやUI処理が終了処理と競合しても、終了開始後に新しい入力を
	// 注入しない。actionChは閉じず、shutdownChでワーカーを停止する。
	// これにより、停止済みタイマーのコールバックが遅れて戻ってきても
	// closed channelへのsendでpanicしない。
	a.shuttingDown.Store(true)
	a.mu.Lock()
	a.abortAllLongPressLocked("application exit", true)
	a.clearJoyConInputStateLocked("application exit")
	a.mu.Unlock()
	a.sendMu.Lock()
	a.sendMu.Unlock()
	a.stopHookSubsystem()
	close(a.shutdownCh)
	close(a.configSaveCh)
	a.workerWG.Wait()
	a.removeTray()
	a.ReleaseModifiersNow()
	a.logf("exiting")
	time.Sleep(80 * time.Millisecond)
}

func keyboardProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode == HC_ACTION {
		app.hookEventSeq.Add(1)
		app.keyEventSeq.Add(1)
		k := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		if k.DwExtraInfo == extraInfoMarker {
			return callNext(0, nCode, wParam, lParam)
		}
		msg := uint32(wParam)
		down := msg == WM_KEYDOWN || msg == WM_SYSKEYDOWN
		up := msg == WM_KEYUP || msg == WM_SYSKEYUP
		if down || up {
			started := time.Now()
			suppress := app.handleKeyEvent(k.VkCode, down)
			if d := time.Since(started); d > 25*time.Millisecond {
				app.logf("slow keyboard hook callback: %s vk=%d down=%v", d.Round(time.Millisecond), k.VkCode, down)
			}
			if suppress {
				return 1
			}
		}
	}
	return callNext(0, nCode, wParam, lParam)
}
func mouseProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode == HC_ACTION {
		app.hookEventSeq.Add(1)
		app.mouseEventSeq.Add(1)
		m := (*MSLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		if m.DwExtraInfo == extraInfoMarker {
			return callNext(0, nCode, wParam, lParam)
		}
		msg := uint32(wParam)
		switch msg {
		case WM_LBUTTONDOWN, WM_LBUTTONUP, WM_RBUTTONDOWN, WM_RBUTTONUP,
			WM_MBUTTONDOWN, WM_MBUTTONUP, WM_XBUTTONDOWN, WM_XBUTTONUP, WM_MOUSEWHEEL:
			started := time.Now()
			suppress := app.handleMouseEvent(msg, m.MouseData)
			if d := time.Since(started); d > 25*time.Millisecond {
				app.logf("slow mouse hook callback: %s message=0x%x data=0x%x", d.Round(time.Millisecond), msg, m.MouseData)
			}
			if suppress {
				return 1
			}
		}
	}
	return callNext(0, nCode, wParam, lParam)
}
func callNext(hhk uintptr, nCode int, wParam uintptr, lParam uintptr) uintptr {
	r, _, _ := pCallNextHookEx.Call(hhk, uintptr(nCode), wParam, lParam)
	return r
}

func (a *App) handleKeyEvent(vk uint32, isDown bool) bool {
	a.mu.Lock()
	wasDown := a.keyDown[vk] || a.keyDown[genericVK(vk)]
	if isDown {
		a.keyDown[vk] = true
		a.keyDown[genericVK(vk)] = true
	} else {
		delete(a.keyDown, vk)
		if !a.anySpecificModDown(genericVK(vk)) {
			delete(a.keyDown, genericVK(vk))
		}
	}
	emergency := isDown && vk == VK_F12 && (a.keyDown[VK_CONTROL] || a.keyDown[VK_LCONTROL] || a.keyDown[VK_RCONTROL]) && (a.keyDown[VK_SHIFT] || a.keyDown[VK_LSHIFT] || a.keyDown[VK_RSHIFT]) && (a.keyDown[VK_MENU] || a.keyDown[VK_LMENU] || a.keyDown[VK_RMENU])
	if emergency {
		a.abortAllLongPressLocked("emergency stop", false)
		a.clearJoyConInputStateLocked("emergency stop")
		a.recordingMode = ""
		a.recordingRuleIndex = -1
		a.recordingProfileID = ""
		a.recordedItems = nil
		a.recordHeld = map[string]bool{}
		a.emergency = true
		a.enabled = false
		a.logf("emergency stop by Ctrl+Alt+Shift+F12")
		a.postUIRefreshLocked()
		a.mu.Unlock()
		go a.ReleaseModifiersNow()
		return true
	}
	it := Item{Kind: "Key", Code: strconv.Itoa(int(vk))}
	if a.recordingMode != "" {
		finish := false
		if isDown && !wasDown {
			a.recordDownLocked(it, "押下")
		} else if !isDown {
			finish = a.recordUpLocked(it, "離した")
		}
		a.postActivityRefreshLocked()
		a.mu.Unlock()
		if finish {
			go a.finishRecordingAuto()
		}
		return true
	}

	if !isDown {
		completion := a.finishLongPressLocked(it)
		a.mu.Unlock()
		if completion.HasRule {
			a.enqueueRuleGuaranteed(completion.Rule)
		}
		if completion.Handled {
			return completion.Suppress
		}
		return false
	}

	if wasDown {
		suppress := a.longPressSuppressedLocked(it)
		a.mu.Unlock()
		return suppress
	}
	a.noteLastInputLocked(it, "押下")
	if !a.enabled || a.emergency {
		a.mu.Unlock()
		return false
	}
	rule, ok := a.findBestTriggerLocked(it)
	if ok && len(rule.Input) > 1 {
		a.markPrefixesConsumedLocked(rule)
	}
	if ok && rule.LongPressEnabled {
		started := a.startLongPressLocked(rule, it)
		a.mu.Unlock()
		return started && rule.SuppressTrigger
	}
	a.mu.Unlock()
	if ok {
		if a.enqueueRule(rule) {
			return rule.SuppressTrigger
		}
		return false
	}
	return false
}
func (a *App) anySpecificModDown(generic uint32) bool {
	switch generic {
	case VK_CONTROL:
		return a.keyDown[VK_LCONTROL] || a.keyDown[VK_RCONTROL]
	case VK_SHIFT:
		return a.keyDown[VK_LSHIFT] || a.keyDown[VK_RSHIFT]
	case VK_MENU:
		return a.keyDown[VK_LMENU] || a.keyDown[VK_RMENU]
	}
	return false
}
func genericVK(vk uint32) uint32 {
	switch vk {
	case VK_LCONTROL, VK_RCONTROL:
		return VK_CONTROL
	case VK_LSHIFT, VK_RSHIFT:
		return VK_SHIFT
	case VK_LMENU, VK_RMENU:
		return VK_MENU
	}
	return vk
}

func (a *App) handleMouseEvent(msg uint32, mouseData uint32) bool {
	switch msg {
	case WM_LBUTTONDOWN:
		return a.buttonDown("Left")
	case WM_LBUTTONUP:
		return a.buttonUp("Left")
	case WM_RBUTTONDOWN:
		return a.buttonDown("Right")
	case WM_RBUTTONUP:
		return a.buttonUp("Right")
	case WM_MBUTTONDOWN:
		return a.buttonDown("Middle")
	case WM_MBUTTONUP:
		return a.buttonUp("Middle")
	case WM_XBUTTONDOWN:
		if hiword(mouseData) == 1 {
			return a.buttonDown("X1")
		}
		if hiword(mouseData) == 2 {
			return a.buttonDown("X2")
		}
	case WM_XBUTTONUP:
		if hiword(mouseData) == 1 {
			return a.buttonUp("X1")
		}
		if hiword(mouseData) == 2 {
			return a.buttonUp("X2")
		}
	case WM_MOUSEWHEEL:
		delta := int16(hiword(mouseData))
		if delta > 0 {
			return a.wheel("WheelUp")
		}
		if delta < 0 {
			return a.wheel("WheelDown")
		}
	}
	return false
}
func hiword(v uint32) uint16          { return uint16((v >> 16) & 0xFFFF) }
func isPrimaryButton(btn string) bool { return btn == "Left" || btn == "Right" || btn == "Middle" }

func (a *App) buttonDown(btn string) bool {
	a.mu.Lock()
	now := time.Now()
	trigger := Item{Kind: "Mouse", Code: btn}
	wasDown := a.mouseDown[btn]
	if wasDown {
		age := now.Sub(a.mouseDownAt[btn])
		staleAfter := 2 * time.Second
		if a.longPress[longPressKey(trigger)] != nil {
			// 長押し中のゲーミングマウスがDOWNを再送しても、正当な押下を
			// UP欠落と誤認しない。長押し状態がある場合だけ猶予を広げる。
			staleAfter = 10 * time.Second
		}
		if age > staleAfter {
			// UP欠落後に次のDOWNが来た場合、永久に「押下中」のまま固まらないよう自己修復する。
			a.abortLongPressForTriggerLocked(trigger, "stale mouse-down recovered")
			delete(a.longPress, longPressKey(trigger))
			delete(a.pendingTap, btn)
			delete(a.consumedPrefix, btn)
			delete(a.suppressedDown, btn)
			wasDown = false
			a.logf("recovered stale mouse-down state: button=%s age=%s", btn, age.Round(time.Millisecond))
		}
	}
	a.mouseDown[btn] = true
	a.mouseDownAt[btn] = now
	if wasDown {
		// 一部のゲーミングマウス/ドライバは押しっぱなし中にDOWNを再送する。
		// Tap/長押しルールを重複発火させず、既に抑制中のボタンだけ同じ扱いを続ける。
		suppress := !isPrimaryButton(btn) && (a.suppressedDown[btn] || a.pendingTap[btn] || a.consumedPrefix[btn] || a.longPressSuppressedLocked(trigger))
		a.mu.Unlock()
		return suppress
	}
	if a.recordingMode != "" {
		a.recordMouseDownLocked(btn)
		a.postActivityRefreshLocked()
		a.mu.Unlock()
		// 左/右/中クリックは、記録ボタンや中止ボタンを押せるように通す。
		return !isPrimaryButton(btn)
	}
	a.noteLastInputLocked(trigger, "押下")
	if !a.enabled || a.emergency {
		a.mu.Unlock()
		return false
	}

	rule, matched := a.findBestTriggerLocked(trigger)
	if matched && len(rule.Input) > 1 {
		a.markPrefixesConsumedLocked(rule)
	}
	if matched && rule.LongPressEnabled {
		started := a.startLongPressLocked(rule, trigger)
		if started && !isPrimaryButton(btn) && rule.SuppressTrigger {
			a.suppressedDown[btn] = true
		}
		a.mu.Unlock()
		return started && !isPrimaryButton(btn) && rule.SuppressTrigger
	}
	if matched && len(rule.Input) > 1 {
		a.mu.Unlock()
		queued := a.enqueueRule(rule)
		if isPrimaryButton(btn) || !queued {
			return false
		}
		return rule.SuppressTrigger
	}

	suppress := false
	if !isPrimaryButton(btn) {
		if rule, ok := a.singleTapRuleLocked(btn); ok && !rule.LongPressEnabled && rule.SuppressTrigger {
			a.pendingTap[btn] = true
			a.suppressedDown[btn] = true
			suppress = true
		} else if a.isSuppressPrefixButtonLocked(btn) {
			a.pendingTap[btn] = true
			a.suppressedDown[btn] = true
			suppress = true
		}
	}
	a.mu.Unlock()
	return suppress
}

func (a *App) buttonUp(btn string) bool {
	a.mu.Lock()
	trigger := Item{Kind: "Mouse", Code: btn}
	if a.recordingMode != "" {
		finish := a.recordUpLocked(trigger, "離した")
		delete(a.mouseDown, btn)
		delete(a.mouseDownAt, btn)
		a.postActivityRefreshLocked()
		a.mu.Unlock()
		if finish {
			go a.finishRecordingAuto()
		}
		return !isPrimaryButton(btn)
	}

	completion := a.finishLongPressLocked(trigger)
	if completion.Handled {
		suppressed := a.suppressedDown[btn]
		consumed := a.consumedPrefix[btn]
		delete(a.mouseDown, btn)
		delete(a.mouseDownAt, btn)
		delete(a.pendingTap, btn)
		delete(a.consumedPrefix, btn)
		delete(a.suppressedDown, btn)
		a.mu.Unlock()
		if completion.HasRule {
			a.enqueueRuleGuaranteed(completion.Rule)
		}
		if isPrimaryButton(btn) {
			return false
		}
		return completion.Suppress || suppressed || consumed
	}

	pending := a.pendingTap[btn]
	consumed := a.consumedPrefix[btn]
	suppressed := a.suppressedDown[btn]
	rule, single := a.singleTapRuleLocked(btn)
	delete(a.mouseDown, btn)
	delete(a.mouseDownAt, btn)
	delete(a.pendingTap, btn)
	delete(a.consumedPrefix, btn)
	delete(a.suppressedDown, btn)
	active := a.enabled && !a.emergency
	a.mu.Unlock()
	if pending && !consumed && single && active && !rule.LongPressEnabled {
		a.enqueueRuleGuaranteed(rule)
	}
	if isPrimaryButton(btn) {
		return false
	}
	return suppressed || pending || consumed
}
func (a *App) wheel(dir string) bool {
	a.mu.Lock()
	if a.recordingMode != "" {
		finish := a.recordWheelLocked(Item{Kind: "Mouse", Code: dir}, "検出")
		a.postActivityRefreshLocked()
		a.mu.Unlock()
		if finish {
			go a.finishRecordingAuto()
		}
		return true
	}
	a.noteLastInputLocked(Item{Kind: "Mouse", Code: dir}, "検出")
	if !a.enabled || a.emergency {
		a.mu.Unlock()
		return false
	}
	rule, ok := a.findBestTriggerLocked(Item{Kind: "Mouse", Code: dir})
	if ok {
		a.markPrefixesConsumedLocked(rule)
	}
	a.mu.Unlock()
	if ok {
		if a.enqueueRule(rule) {
			return rule.SuppressTrigger
		}
		return false
	}
	return false
}

func (a *App) singleTapRuleLocked(btn string) (Rule, bool) {
	for _, r := range a.rules {
		if len(r.Input) == 1 && strings.EqualFold(r.Input[0].Kind, "Mouse") && normMouse(r.Input[0].Code) == btn {
			return r, true
		}
	}
	return Rule{}, false
}
func (a *App) isSuppressPrefixButtonLocked(btn string) bool {
	for _, r := range a.rules {
		if len(r.Input) > 1 && r.SuppressPrefix {
			for i := 0; i < len(r.Input)-1; i++ {
				if strings.EqualFold(r.Input[i].Kind, "Mouse") && normMouse(r.Input[i].Code) == btn {
					return true
				}
			}
		}
	}
	return false
}
func (a *App) markPrefixesConsumedLocked(r Rule) {
	for i := 0; i < len(r.Input)-1; i++ {
		it := r.Input[i]
		a.abortLongPressForTriggerLocked(it, "input became a prefix of a longer rule")
		if strings.EqualFold(it.Kind, "Mouse") {
			a.consumedPrefix[normMouse(it.Code)] = true
		}
		if isControllerInputKind(it.Kind) {
			key := controllerInputKey(it)
			if key != "" {
				a.controllerConsumed[key] = true
				if holdRule, ok := a.controllerHoldRules[key]; ok {
					delete(a.controllerHoldRules, key)
					a.enqueueRuleGuaranteed(joyConHoldPhaseRule(holdRule, false))
				}
			}
		}
	}
}
func (a *App) noteLastInputLocked(it Item, phase string) {
	// この関数は a.mu を保持した状態で呼ぶ。
	a.lastInputText = itemsText([]Item{normalizeRecordedItem(it)}) + " " + phase
	a.lastInputAt = time.Now()
	a.postActivityRefreshLocked()
}

func recordItemKey(it Item) string {
	n := normalizeRecordedItem(it)
	if strings.EqualFold(n.Kind, "Mouse") {
		return "Mouse:" + normMouse(n.Code)
	}
	if strings.EqualFold(n.Kind, "Key") {
		if vk, ok := parseVK(n.Code); ok {
			return "Key:" + strconv.Itoa(int(genericVK(vk)))
		}
	}
	return n.Kind + ":" + n.Code
}

func (a *App) appendRecordedItemLocked(it Item) {
	n := normalizeRecordedItem(it)
	for _, ex := range a.recordedItems {
		if sameInput(ex, n) {
			return
		}
	}
	a.recordedItems = append(a.recordedItems, n)
}

func isOutputRecordingMode(mode string) bool {
	return mode == "output" || mode == "long-output"
}

func (a *App) recordMouseDownLocked(btn string) {
	it := Item{Kind: "Mouse", Code: btn}
	a.noteLastInputLocked(it, "押下")
	if isOutputRecordingMode(a.recordingMode) {
		return
	}
	if a.recordHeld == nil {
		a.recordHeld = map[string]bool{}
	}
	// 左/右/中クリックだけの誤登録を避ける。GUIの中止ボタン等を押せるようにするため。
	// ただし、既にサイドボタン等を記録済みなら「サイド + 左クリック」の最後の入力として記録する。
	if isPrimaryButton(btn) && len(a.recordedItems) == 0 {
		a.recordHeld[recordItemKey(it)] = true
		return
	}
	a.appendHeldMousePrefixesLocked()
	a.appendRecordedItemLocked(it)
	a.recordHeld[recordItemKey(it)] = true
}

func (a *App) appendHeldMousePrefixesLocked() {
	if a.recordingMode != "input" {
		return
	}
	if a.recordHeld == nil {
		a.recordHeld = map[string]bool{}
	}
	for _, btn := range []string{"Left", "Right", "Middle", "X1", "X2"} {
		if a.mouseDown[btn] {
			it := Item{Kind: "Mouse", Code: btn}
			a.appendRecordedItemLocked(it)
			a.recordHeld[recordItemKey(it)] = true
		}
	}
	controllerKeys := make([]string, 0, len(a.controllerDown))
	for key := range a.controllerDown {
		controllerKeys = append(controllerKeys, key)
	}
	sort.Strings(controllerKeys)
	for _, key := range controllerKeys {
		it, ok := controllerItemFromKey(key)
		if !ok {
			continue
		}
		a.appendRecordedItemLocked(it)
		a.recordHeld[recordItemKey(it)] = true
	}
}

func (a *App) recordDownLocked(it Item, phase string) {
	// この関数は a.mu を保持した状態で呼ぶ。
	a.noteLastInputLocked(it, phase)
	if isOutputRecordingMode(a.recordingMode) && !strings.EqualFold(it.Kind, "Key") {
		return
	}
	if a.recordingMode == "input" {
		a.appendHeldMousePrefixesLocked()
	}
	a.appendRecordedItemLocked(it)
	if a.recordHeld == nil {
		a.recordHeld = map[string]bool{}
	}
	a.recordHeld[recordItemKey(it)] = true
}

func (a *App) recordUpLocked(it Item, phase string) bool {
	// この関数は a.mu を保持した状態で呼ぶ。
	a.noteLastInputLocked(it, phase)
	if isOutputRecordingMode(a.recordingMode) && !strings.EqualFold(it.Kind, "Key") {
		return false
	}
	if a.recordHeld != nil {
		delete(a.recordHeld, recordItemKey(it))
	}
	return len(a.recordedItems) > 0 && len(a.recordHeld) == 0
}

func (a *App) recordWheelLocked(it Item, phase string) bool {
	// ホイールは「押下状態」を持たないため、単体なら即完了。
	// サイドボタン等を押しながら回した場合は、そのボタンを離した時点で完了。
	a.noteLastInputLocked(it, phase)
	if isOutputRecordingMode(a.recordingMode) {
		return false
	}
	a.appendHeldMousePrefixesLocked()
	a.appendRecordedItemLocked(it)
	return len(a.recordedItems) > 0 && len(a.recordHeld) == 0
}

func normalizeRecordedItem(it Item) Item {
	if strings.EqualFold(it.Kind, "Mouse") {
		return Item{Kind: "Mouse", Code: normMouse(it.Code)}
	}
	if strings.EqualFold(it.Kind, "Key") {
		if vk, ok := parseVK(it.Code); ok {
			return Item{Kind: "Key", Code: strconv.Itoa(int(genericVK(vk)))}
		}
	}
	if strings.EqualFold(it.Kind, "JoyCon") {
		return Item{Kind: "JoyCon", Code: normalizeJoyConCode(it.Code)}
	}
	if strings.EqualFold(it.Kind, "XInput") {
		return Item{Kind: "XInput", Code: normalizeXInputCode(it.Code)}
	}
	return it
}

func (a *App) findBestTriggerLocked(trigger Item) (Rule, bool) {
	bestLen := -1
	var best Rule
	for _, r := range a.rules {
		if len(r.Input) == 0 {
			continue
		}
		last := r.Input[len(r.Input)-1]
		if !sameInput(last, trigger) {
			continue
		}
		if len(r.Input) == 1 {
			if bestLen < 1 {
				bestLen = 1
				best = r
			}
			continue
		}
		ok := true
		for i := 0; i < len(r.Input)-1; i++ {
			if !a.isItemDownLocked(r.Input[i]) {
				ok = false
				break
			}
		}
		if ok && len(r.Input) > bestLen {
			bestLen = len(r.Input)
			best = r
		}
	}
	if bestLen >= 0 {
		return best, true
	}
	return Rule{}, false
}
func (a *App) isItemDownLocked(it Item) bool {
	if strings.EqualFold(it.Kind, "Mouse") {
		return a.mouseDown[normMouse(it.Code)]
	}
	if strings.EqualFold(it.Kind, "Key") {
		vk, ok := parseVK(it.Code)
		return ok && (a.keyDown[vk] || a.keyDown[genericVK(vk)])
	}
	if isControllerInputKind(it.Kind) {
		key := controllerInputKey(it)
		return key != "" && a.controllerDown[key]
	}
	return false
}
func sameInput(a Item, b Item) bool {
	if !strings.EqualFold(a.Kind, b.Kind) {
		return false
	}
	if strings.EqualFold(a.Kind, "Mouse") {
		return normMouse(a.Code) == normMouse(b.Code)
	}
	if strings.EqualFold(a.Kind, "Key") {
		av, aok := parseVK(a.Code)
		bv, bok := parseVK(b.Code)
		return aok && bok && genericVK(av) == genericVK(bv)
	}
	if strings.EqualFold(a.Kind, "JoyCon") {
		return normalizeJoyConCode(a.Code) == normalizeJoyConCode(b.Code)
	}
	if strings.EqualFold(a.Kind, "XInput") {
		return normalizeXInputCode(a.Code) == normalizeXInputCode(b.Code)
	}
	return false
}

func normMouse(code string) string {
	c := strings.TrimSpace(strings.ToLower(code))
	switch c {
	case "left", "lbutton", "左クリック":
		return "Left"
	case "right", "rbutton", "右クリック":
		return "Right"
	case "middle", "mbutton", "中クリック":
		return "Middle"
	case "x1", "back", "戻る", "side1":
		return "X1"
	case "x2", "forward", "進む", "side2":
		return "X2"
	case "wheelup", "wheel_up", "up":
		return "WheelUp"
	case "wheeldown", "wheel_down", "down":
		return "WheelDown"
	}
	return code
}
func parseVK(code string) (uint32, bool) {
	c := strings.TrimSpace(code)
	if n, err := strconv.Atoi(c); err == nil {
		return uint32(n), true
	}
	u := strings.ToUpper(c)
	if len(u) == 1 {
		ch := u[0]
		if ch >= 'A' && ch <= 'Z' {
			return uint32(ch), true
		}
		if ch >= '0' && ch <= '9' {
			return uint32(ch), true
		}
	}
	switch u {
	case "CTRL", "CONTROL":
		return VK_CONTROL, true
	case "SHIFT":
		return VK_SHIFT, true
	case "ALT", "MENU":
		return VK_MENU, true
	case "WIN", "LWIN":
		return VK_LWIN, true
	case "RWIN":
		return VK_RWIN, true
	case "ENTER", "RETURN":
		return VK_RETURN, true
	case "SPACE":
		return VK_SPACE, true
	case "TAB":
		return VK_TAB, true
	case "ESC", "ESCAPE":
		return VK_ESCAPE, true
	case "BACKSPACE", "BACK":
		return VK_BACK, true
	case "LEFT":
		return VK_LEFT, true
	case "RIGHT":
		return VK_RIGHT, true
	case "UP":
		return VK_UP, true
	case "DOWN":
		return VK_DOWN, true
	case "PAGEUP", "PGUP":
		return VK_PRIOR, true
	case "PAGEDOWN", "PGDN":
		return VK_NEXT, true
	case "PRINTSCREEN", "PRTSC":
		return VK_SNAPSHOT, true
	}
	if strings.HasPrefix(u, "F") {
		if n, err := strconv.Atoi(u[1:]); err == nil && n >= 1 && n <= 24 {
			return uint32(0x6F + n), true
		}
	}
	return 0, false
}

func (a *App) sendRule(r Rule) {
	a.sendJoyConRuleOutput(r)
}
func isModifier(vk uint32) bool {
	switch genericVK(vk) {
	case VK_CONTROL, VK_SHIFT, VK_MENU:
		return true
	}
	return vk == VK_LWIN || vk == VK_RWIN
}
func normalizeModifier(vk uint32) uint32 {
	switch genericVK(vk) {
	case VK_CONTROL:
		return VK_LCONTROL
	case VK_SHIFT:
		return VK_LSHIFT
	case VK_MENU:
		return VK_LMENU
	}
	if vk == VK_RWIN {
		return VK_RWIN
	}
	if vk == VK_LWIN {
		return VK_LWIN
	}
	return vk
}
func (a *App) sendShortcut(keys []uint32) {
	a.sendMu.Lock()
	defer a.sendMu.Unlock()
	if a.shuttingDown.Load() {
		return
	}
	mods := []uint32{}
	normals := []uint32{}
	seen := map[uint32]bool{}
	for _, vk := range keys {
		if isModifier(vk) {
			mv := normalizeModifier(vk)
			if !seen[mv] {
				mods = append(mods, mv)
				seen[mv] = true
			}
		} else {
			normals = append(normals, vk)
		}
	}
	inputs := []INPUT{}
	pressedByUs := []uint32{}
	for _, vk := range mods {
		// ユーザーが物理的に押している修飾キーは借りるだけにし、こちらからUPを送らない。
		// 旧実装はCtrl等を必ずUPしていたため、押しっぱなし中の物理状態を壊し得た。
		if a.physicalModDown(vk) {
			continue
		}
		inputs = append(inputs, makeKeyInput(vk, false))
		pressedByUs = append(pressedByUs, vk)
	}
	if len(normals) == 0 {
		for i := len(pressedByUs) - 1; i >= 0; i-- {
			inputs = append(inputs, makeKeyInput(pressedByUs[i], true))
		}
	} else {
		for _, vk := range normals {
			inputs = append(inputs, makeKeyInput(vk, false), makeKeyInput(vk, true))
		}
		for i := len(pressedByUs) - 1; i >= 0; i-- {
			inputs = append(inputs, makeKeyInput(pressedByUs[i], true))
		}
	}
	if ok := a.callSendInput(inputs); !ok && len(pressedByUs) > 0 {
		cleanup := make([]INPUT, 0, len(pressedByUs))
		for i := len(pressedByUs) - 1; i >= 0; i-- {
			cleanup = append(cleanup, makeKeyInput(pressedByUs[i], true))
		}
		a.callSendInput(cleanup)
	}
}
func makeKeyInput(vk uint32, keyup bool) INPUT {
	var in INPUT
	in.Type = INPUT_KEYBOARD
	ki := KEYBDINPUT{WVk: uint16(vk), DwExtraInfo: extraInfoMarker}
	if keyup {
		ki.DwFlags |= KEYEVENTF_KEYUP
	}
	if isExtendedVK(vk) {
		ki.DwFlags |= KEYEVENTF_EXTENDEDKEY
	}
	*(*KEYBDINPUT)(unsafe.Pointer(&in.Data[0])) = ki
	return in
}
func isExtendedVK(vk uint32) bool {
	switch vk {
	case VK_LEFT, VK_RIGHT, VK_UP, VK_DOWN, VK_PRIOR, VK_NEXT, VK_END, VK_HOME, VK_RCONTROL, VK_RMENU, VK_LWIN, VK_RWIN:
		return true
	}
	return false
}
func (a *App) callSendInput(inputs []INPUT) bool {
	if len(inputs) == 0 {
		return true
	}
	r, _, _ := pSendInput.Call(uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(inputs[0]))
	if r != uintptr(len(inputs)) {
		gle, _, _ := pGetLastError.Call()
		a.logf("SendInput partial: sent=%d expected=%d gle=%d", r, len(inputs), gle)
		return false
	}
	return true
}
func asyncDown(vk uint32) bool {
	r, _, _ := pGetAsyncKeyState.Call(uintptr(vk))
	return int16(r&0xffff) < 0
}
func (a *App) physicalModDown(vk uint32) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	switch vk {
	case VK_LCONTROL, VK_RCONTROL, VK_CONTROL:
		return a.keyDown[VK_LCONTROL] || a.keyDown[VK_RCONTROL] || a.keyDown[VK_CONTROL]
	case VK_LSHIFT, VK_RSHIFT, VK_SHIFT:
		return a.keyDown[VK_LSHIFT] || a.keyDown[VK_RSHIFT] || a.keyDown[VK_SHIFT]
	case VK_LMENU, VK_RMENU, VK_MENU:
		return a.keyDown[VK_LMENU] || a.keyDown[VK_RMENU] || a.keyDown[VK_MENU]
	case VK_LWIN, VK_RWIN:
		return a.keyDown[vk]
	}
	return a.keyDown[vk]
}
func (a *App) releaseStuckModifiersLocked() {
	mods := []uint32{VK_LCONTROL, VK_RCONTROL, VK_LSHIFT, VK_RSHIFT, VK_LMENU, VK_RMENU, VK_LWIN, VK_RWIN}
	inputs := []INPUT{}
	for _, vk := range mods {
		if asyncDown(vk) && !a.physicalModDown(vk) {
			inputs = append(inputs, makeKeyInput(vk, true))
		}
	}
	if len(inputs) > 0 {
		a.callSendInput(inputs)
		a.logf("released stuck modifiers: %d", len(inputs))
	}
}
func (a *App) ReleaseModifiersNow() {
	a.sendMu.Lock()
	defer a.sendMu.Unlock()
	mods := []uint32{VK_LCONTROL, VK_RCONTROL, VK_LSHIFT, VK_RSHIFT, VK_LMENU, VK_RMENU, VK_LWIN, VK_RWIN, VK_CONTROL, VK_SHIFT, VK_MENU}
	inputs := []INPUT{}
	for _, vk := range mods {
		inputs = append(inputs, makeKeyInput(vk, true))
	}
	inputs = a.appendJoyConHeldReleaseInputs(inputs)
	a.callSendInput(inputs)
	a.logf("manual modifier release")
}

func (a *App) createMainWindow(show bool) error {
	hinst, _, _ := pGetModuleHandleW.Call(0)
	icc := INITCOMMONCONTROLSEX{DwSize: uint32(unsafe.Sizeof(INITCOMMONCONTROLSEX{})), DwICC: ICC_LISTVIEW_CLASSES}
	pInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))
	a.wndCb = syscall.NewCallback(wndProc)
	a.font = createFont(-16, 400)
	if a.font == 0 {
		a.font, _, _ = pGetStockObject.Call(DEFAULT_GUI_FONT)
	}
	a.fontTitle = createFont(-24, 700)
	if a.fontTitle == 0 {
		a.fontTitle = a.font
	}
	a.fontSmall = createFont(-14, 400)
	if a.fontSmall == 0 {
		a.fontSmall = a.font
	}
	a.fontButton = createFont(-17, 700)
	if a.fontButton == 0 {
		a.fontButton = a.font
	}
	icon := a.loadAppIcon()
	className := syscall.StringToUTF16Ptr(mainWindowClass)
	wc := WNDCLASSEX{CbSize: uint32(unsafe.Sizeof(WNDCLASSEX{})), LpfnWndProc: a.wndCb, HInstance: hinst, HIcon: icon, HIconSm: icon, HbrBackground: 0, LpszClassName: className}
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	title := syscall.StringToUTF16Ptr(appName + " " + appVersion)
	hwnd, _, err := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)), uintptr(WS_OVERLAPPEDWINDOW), uintptr(CW_USEDEFAULT), uintptr(CW_USEDEFAULT), 1280, 820, 0, 0, hinst, 0)
	if hwnd == 0 {
		return err
	}
	a.hwnd = hwnd
	// v7.9.0: 既定ではEdgeのアプリウィンドウで設定GUIを開く。
	// --webview2 指定時だけ、WebView2Loader.dll同梱環境で内蔵WebView2を試す。
	a.addTray()
	if show {
		a.showMainWindow()
	}
	return nil
}

func wndProc(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	switch msg {
	case trayMsg:
		if lParam == WM_RBUTTONUP {
			app.showTrayMenu()
			return 0
		}
		if lParam == WM_LBUTTONDBLCLK || lParam == WM_LBUTTONUP {
			app.showMainWindow()
			return 0
		}
	case showMsg:
		app.showMainWindow()
		return 0
	case refreshMsg:
		app.refreshUI()
		return 0
	case activityMsg:
		app.updateStatus()
		app.updateButtons()
		app.updateActivityControls()
		return 0
	case WM_PAINT:
		app.paint(hwnd)
		return 0
	case WM_SIZE:
		app.resizeWebView2()
		pInvalidateRect.Call(hwnd, 0, 1)
		return 0
	case WM_NOTIFY:
		hdr := (*NMHDR)(unsafe.Pointer(lParam))
		if hdr != nil && hdr.IdFrom == ID_LIST_RULES {
			if hdr.Code == NM_CUSTOMDRAW {
				return app.handleRuleListCustomDraw(lParam)
			}
			if hdr.Code == NM_CLICK || hdr.Code == NM_DBLCLK {
				act := (*NMITEMACTIVATE)(unsafe.Pointer(lParam))
				if act != nil && app.handleRuleListCellClick(int(act.IItem), int(act.ISubItem)) {
					return 0
				}
				app.updateEditorFromSelection()
				return 0
			}
			if hdr.Code == LVN_ITEMCHANGED {
				app.updateEditorFromSelection()
				return 0
			}
		}
	case WM_COMMAND:
		id := uint32(wParam & 0xffff)
		code := uint32((wParam >> 16) & 0xffff)
		if id == ID_PROFILE_COMBO && code == CBN_SELCHANGE {
			app.changeProfileFromCombo()
			return 0
		}
		app.handleCommand(id)
		return 0
	case WM_CLOSE:
		pShowWindow.Call(hwnd, SW_HIDE)
		return 0
	case WM_DESTROY:
		app.closeWebView2()
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func (a *App) paint(hwnd uintptr) {
	var ps PAINTSTRUCT
	hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	var rc RECT
	pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	fill(hdc, rc, rgb(240, 240, 240))
	pSetBkMode.Call(hdc, TRANSPARENT)
	old, _, _ := pSelectObject.Call(hdc, a.fontTitle)
	pSetTextColor.Call(hdc, uintptr(rgb(0, 0, 0)))
	drawText(hdc, "マウスボタンの割り当て", RECT{22, 12, rc.Right - 24, 46}, DT_LEFT|DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS)
	pSelectObject.Call(hdc, a.fontSmall)
	pSetTextColor.Call(hdc, uintptr(rgb(65, 65, 65)))
	drawText(hdc, "使用中のプロファイル", RECT{22, 56, 150, 82}, DT_LEFT|DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS)
	drawText(hdc, "プロファイルを切り替えると、現在の編集内容を保存してから、選んだ割り当てをすぐに有効にします。入力と実行内容は、実際にマウスやキーボードを操作して記録できます。", RECT{22, 104, rc.Right - 24, 132}, DT_LEFT|DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS)
	pSetTextColor.Call(hdc, uintptr(rgb(0, 0, 0)))
	drawText(hdc, "最後に検出した入力", RECT{22, rc.Bottom - 182, rc.Right - 24, rc.Bottom - 158}, DT_LEFT|DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS)
	drawText(hdc, "動作状態", RECT{22, rc.Bottom - 112, rc.Right - 24, rc.Bottom - 88}, DT_LEFT|DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS)
	pSetTextColor.Call(hdc, uintptr(rgb(90, 90, 90)))
	drawText(hdc, "緊急停止：Ctrl＋Alt＋Shift＋F12　／　基本クリック・Enter・Space等の単体置換は禁止", RECT{22, rc.Bottom - 38, rc.Right - 24, rc.Bottom - 14}, DT_LEFT|DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS)
	if old != 0 {
		pSelectObject.Call(hdc, old)
	}
}

func fill(hdc uintptr, rc RECT, color uint32) {
	br := brush(color)
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), br)
	pDeleteObject.Call(br)
}
func brush(color uint32) uintptr { b, _, _ := pCreateSolidBrush.Call(uintptr(color)); return b }
func rgb(r, g, b byte) uint32    { return uint32(r) | uint32(g)<<8 | uint32(b)<<16 }
func drawText(hdc uintptr, text string, rc RECT, flags uint32) {
	t := syscall.StringToUTF16(text)
	pDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&t[0])), uintptr(len(t)-1), uintptr(unsafe.Pointer(&rc)), uintptr(flags))
}
func i32(v int32) uintptr { return uintptr(uint32(v)) }
func createFont(height int32, weight int32) uintptr {
	name := syscall.StringToUTF16Ptr("Segoe UI")
	h, _, _ := pCreateFontW.Call(i32(height), 0, 0, 0, uintptr(uint32(weight)), 0, 0, 0, 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(name)))
	return h
}

func (a *App) createControls() {
	a.ctrlStatus = createChild("STATIC", "", SS_LEFT, 0)
	a.ctrlMessage = createChild("STATIC", "", SS_LEFT, 0)
	a.ctrlLastInput = createChild("STATIC", "最後に検出した入力: （まだありません）", SS_LEFT, 0)
	a.ctrlProfile = createChild("COMBOBOX", "", CBS_DROPDOWNLIST|CBS_HASSTRINGS|WS_TABSTOP, ID_PROFILE_COMBO)
	a.ctrlList = createChildEx(WS_EX_CLIENTEDGE, "SysListView32", "", LVS_REPORT|LVS_SINGLESEL|LVS_SHOWSELALWAYS|LVS_NOSORTHEADER|WS_TABSTOP, ID_LIST_RULES)
	setupRuleListColumns(a.ctrlList)
	a.lblInput = createChild("STATIC", "入力", SS_LEFT, 0)
	a.editInput = createChildEx(WS_EX_CLIENTEDGE, "EDIT", "", ES_AUTOHSCROLL|WS_TABSTOP, ID_EDIT_INPUT)
	a.lblOutput = createChild("STATIC", "実行内容", SS_LEFT, 0)
	a.editOutput = createChildEx(WS_EX_CLIENTEDGE, "EDIT", "", ES_AUTOHSCROLL|WS_TABSTOP, ID_EDIT_OUTPUT)
	// 実行方法の編集プルダウンは置かない。現在の安定コアでは Tap 固定で、一覧に表示するだけにする。
	a.chkEnabled = createChild("BUTTON", "有効", BS_AUTOCHECKBOX|WS_TABSTOP, ID_CHK_ENABLED)
	a.chkSuppressTrigger = createChild("BUTTON", "最後の入力を通常動作させない", BS_AUTOCHECKBOX|WS_TABSTOP, ID_CHK_SUPPRESS_TRIGGER)
	a.chkSuppressPrefix = createChild("BUTTON", "サイド単押しを取り消す", BS_AUTOCHECKBOX|WS_TABSTOP, ID_CHK_SUPPRESS_PREFIX)

	a.btnStartStop = createChild("BUTTON", "", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_STARTSTOP)
	a.btnEmergency = createChild("BUTTON", "緊急停止", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_EMERGENCY)
	a.btnRelease = createChild("BUTTON", "修飾キー解放", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_RELEASE)
	a.btnReload = createChild("BUTTON", "設定再読み込み", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_RELOAD)
	a.btnSaveRule = createChild("BUTTON", "選択中の割り当てを保存", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_SAVE_RULE)
	a.btnAddRule = createChild("BUTTON", "ルールを追加", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_ADD_RULE)
	a.btnDupRule = createChild("BUTTON", "複製", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_DUP_RULE)
	a.btnDeleteRule = createChild("BUTTON", "削除", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_DELETE_RULE)
	a.btnMoveTop = createChild("BUTTON", "最上部へ", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_MOVE_TOP)
	a.btnMoveUp = createChild("BUTTON", "上へ", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_MOVE_UP)
	a.btnMoveDown = createChild("BUTTON", "下へ", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_MOVE_DOWN)
	a.btnMoveBottom = createChild("BUTTON", "最下部へ", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_MOVE_BOTTOM)
	a.btnTestOutput = createChild("BUTTON", "実行内容をテスト", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_TEST_OUTPUT)
	a.btnRecordInput = createChild("BUTTON", "★ 入力を記録", BS_DEFPUSHBUTTON|WS_TABSTOP, ID_BTN_RECORD_INPUT)
	a.btnRecordOutput = createChild("BUTTON", "★ 実行内容を記録", BS_DEFPUSHBUTTON|WS_TABSTOP, ID_BTN_RECORD_OUTPUT)
	a.btnRecordStop = createChild("BUTTON", "記録を中止", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_RECORD_STOP)
	a.btnDefaultRules = createChild("BUTTON", "既定ルールを表示", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_DEFAULT_RULES)
	a.btnOpenCfg = createChild("BUTTON", "JSONを開く", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_OPENCFG)
	a.btnOpenFolder = createChild("BUTTON", "設定フォルダー", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_OPENFOLDER)
	a.btnOpenLog = createChild("BUTTON", "ログを開く", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_OPENLOG)
	a.btnExport = createChild("BUTTON", "すべてエクスポート", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_EXPORT)
	a.btnImport = createChild("BUTTON", "設定をインポート", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_IMPORT)
	a.btnCopyDiag = createChild("BUTTON", "診断ログをコピー", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_COPY_DIAG)
	a.btnSafe = createChild("BUTTON", "変換を停止", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_SAFE)
	a.btnQuit = createChild("BUTTON", "終了", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_QUIT)
	controls := []uintptr{a.ctrlStatus, a.ctrlMessage, a.ctrlLastInput, a.ctrlProfile, a.ctrlList, a.lblInput, a.editInput, a.lblOutput, a.editOutput, a.chkEnabled, a.chkSuppressTrigger, a.chkSuppressPrefix, a.btnStartStop, a.btnEmergency, a.btnRelease, a.btnReload, a.btnSaveRule, a.btnAddRule, a.btnDupRule, a.btnDeleteRule, a.btnMoveTop, a.btnMoveUp, a.btnMoveDown, a.btnMoveBottom, a.btnTestOutput, a.btnRecordInput, a.btnRecordOutput, a.btnRecordStop, a.btnDefaultRules, a.btnOpenCfg, a.btnOpenFolder, a.btnOpenLog, a.btnExport, a.btnImport, a.btnCopyDiag, a.btnSafe, a.btnQuit}
	for _, h := range controls {
		setFont(h, a.font)
	}
	setFont(a.ctrlMessage, a.fontSmall)
	setFont(a.ctrlLastInput, a.fontSmall)
	setFont(a.btnRecordInput, a.fontButton)
	setFont(a.btnRecordOutput, a.fontButton)
}

func createChild(class, text string, style uint32, id uint32) uintptr {
	return createChildEx(0, class, text, style, id)
}
func createChildEx(exStyle uint32, class, text string, style uint32, id uint32) uintptr {
	hinst, _, _ := pGetModuleHandleW.Call(0)
	hwnd, _, _ := pCreateWindowExW.Call(uintptr(exStyle), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(class))), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))), uintptr(WS_CHILD|WS_VISIBLE|style), 0, 0, 10, 10, app.hwnd, uintptr(id), hinst, 0)
	return hwnd
}
func setFont(hwnd uintptr, font uintptr) {
	if hwnd != 0 && font != 0 {
		pSendMessageW.Call(hwnd, WM_SETFONT, font, 1)
	}
}
func move(hwnd uintptr, x, y, w, h int32) {
	if hwnd != 0 {
		pMoveWindow.Call(hwnd, uintptr(x), uintptr(y), uintptr(w), uintptr(h), 1)
	}
}

func setupRuleListColumns(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	pSendMessageW.Call(hwnd, LVM_SETEXTENDEDLISTVIEWSTYLE, 0, uintptr(LVS_EX_FULLROWSELECT|LVS_EX_GRIDLINES|LVS_EX_DOUBLEBUFFER))
	cols := []struct {
		title string
		width int32
	}{
		{"使う", 54}, {"この操作をしたら", 360}, {"実行方法", 170}, {"この操作を実行する", 360}, {"最後の入力を通常動作させない", 300}, {"サイド単押しを取り消す", 260},
	}
	for i, c := range cols {
		lvInsertColumn(hwnd, i, c.title, c.width)
	}
}
func lvInsertColumn(hwnd uintptr, index int, title string, width int32) {
	t := syscall.StringToUTF16(title)
	col := LVCOLUMNW{Mask: LVCF_FMT | LVCF_WIDTH | LVCF_TEXT | LVCF_SUBITEM, Cx: width, PszText: &t[0], CchTextMax: int32(len(t)), ISubItem: int32(index)}
	pSendMessageW.Call(hwnd, LVM_INSERTCOLUMNW, uintptr(index), uintptr(unsafe.Pointer(&col)))
}
func lvInsertRow(hwnd uintptr, row int, cols []string) {
	if len(cols) == 0 {
		return
	}
	t := syscall.StringToUTF16(cols[0])
	item := LVITEMW{Mask: LVIF_TEXT, IItem: int32(row), ISubItem: 0, PszText: &t[0], CchTextMax: int32(len(t))}
	pSendMessageW.Call(hwnd, LVM_INSERTITEMW, 0, uintptr(unsafe.Pointer(&item)))
	for i := 1; i < len(cols); i++ {
		lvSetSubItem(hwnd, row, i, cols[i])
	}
}
func lvSetSubItem(hwnd uintptr, row int, col int, text string) {
	t := syscall.StringToUTF16(text)
	item := LVITEMW{IItem: int32(row), ISubItem: int32(col), PszText: &t[0], CchTextMax: int32(len(t))}
	pSendMessageW.Call(hwnd, LVM_SETITEMTEXTW, uintptr(row), uintptr(unsafe.Pointer(&item)))
}

func (a *App) layoutControls() {
	if a.hwnd == 0 || a.ctrlList == 0 {
		return
	}
	var rc RECT
	pGetClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&rc)))
	w := rc.Right - rc.Left
	h := rc.Bottom - rc.Top
	if w < 1180 {
		w = 1180
	}
	if h < 760 {
		h = 760
	}
	m := int32(20)
	gap := int32(8)
	btnH := int32(36)
	rowH := int32(28)

	// 上段: 旧GUIと同じ「プロファイル → 操作ボタン → プロファイル状態」構成。
	move(a.ctrlProfile, 150, 54, 260, 240)
	move(a.btnAddRule, m, h-382, 125, btnH)
	move(a.btnDeleteRule, m+133, h-382, 80, btnH)
	move(a.btnMoveTop, m+221, h-382, 95, btnH)
	move(a.btnMoveUp, m+324, h-382, 72, btnH)
	move(a.btnMoveDown, m+404, h-382, 72, btnH)
	move(a.btnMoveBottom, m+484, h-382, 95, btnH)

	// 目立たせるべき主役。旧GUIより太め・広めにする。
	move(a.btnRecordInput, m+600, h-390, 190, btnH+8)
	move(a.btnRecordOutput, m+798, h-390, 230, btnH+8)
	move(a.btnRecordStop, m+1036, h-382, 120, btnH)
	move(a.btnDefaultRules, m+1164, h-382, 155, btnH)

	// ルール一覧。チェック列はクリックで即トグル・即保存する。
	gridTop := int32(140)
	gridBottom := h - 400
	if gridBottom < 400 {
		gridBottom = 400
	}
	move(a.ctrlList, m, gridTop, w-2*m, gridBottom-gridTop)

	// 下段操作ボタン。元GUIのボタン配置に寄せる。
	y2 := h - 340
	bx := m
	row2 := []struct {
		h uintptr
		w int32
	}{
		{a.btnExport, 155}, {a.btnImport, 145}, {a.btnOpenFolder, 135}, {a.btnOpenLog, 105}, {a.btnCopyDiag, 155}, {a.btnEmergency, 145}, {a.btnSaveRule, 160}, {a.btnStartStop, 160},
	}
	for _, b := range row2 {
		move(b.h, bx, y2, b.w, btnH)
		bx += b.w + gap
	}

	// 選択中ルールの確認・手修正欄。プルダウンは廃止し、実行方法は現在の安定コアに合わせて固定表示。
	y3 := h - 292
	move(a.chkEnabled, m, y3, 70, rowH)
	move(a.lblInput, m+82, y3+4, 48, rowH)
	editW := (w - 2*m - 590) / 2
	if editW < 220 {
		editW = 220
	}
	move(a.editInput, m+128, y3, editW, rowH)
	move(a.lblOutput, m+140+editW, y3+4, 72, rowH)
	move(a.editOutput, m+214+editW, y3, editW, rowH)
	move(a.btnTestOutput, w-170, y3-2, 150, btnH)

	y4 := h - 250
	move(a.chkSuppressTrigger, m, y4, 300, rowH)
	move(a.chkSuppressPrefix, m+320, y4, 260, rowH)
	move(a.btnRelease, m+600, y4-4, 116, btnH)
	move(a.btnReload, m+724, y4-4, 130, btnH)
	move(a.btnOpenCfg, m+862, y4-4, 105, btnH)
	move(a.btnSafe, m+975, y4-4, 104, btnH)
	move(a.btnQuit, m+1087, y4-4, 78, btnH)

	// capturePanel相当: 記録状態・保存結果などを広く出す。
	move(a.ctrlMessage, m, h-218, w-2*m, 50)
	move(a.ctrlLastInput, m, h-150, w-2*m, 36)
	move(a.ctrlStatus, m, h-80, w-2*m, 36)
}

func (a *App) showMainWindow() {
	url := a.webURL()
	if url == "" {
		messageBox("MouseButtonMapper", "設定画面がまだ起動していません。")
		return
	}
	// v7.9.0: 前版の空ウィンドウ事故を避けるため、既定では内蔵WebView2を使わない。
	// 内蔵WebView2を試す場合は、EXE隣にWebView2Loader.dllを置き、--webview2で起動する。
	if !a.forceWebView2 {
		a.mu.Lock()
		a.settingsGraceUntil = time.Now().Add(3 * time.Second)
		a.mu.Unlock()
		openEdgeAppOrBrowser(url)
		return
	}
	if !hasBundledWebView2Loader() {
		a.logf("--webview2 specified but bundled WebView2Loader.dll was not found; falling back to Edge app window")
		a.mu.Lock()
		a.settingsGraceUntil = time.Now().Add(3 * time.Second)
		a.mu.Unlock()
		openEdgeAppOrBrowser(url)
		return
	}
	pShowWindow.Call(a.hwnd, SW_RESTORE)
	pSetForegroundWindow.Call(a.hwnd)
	if a.showWebView2(url) {
		go a.fallbackIfWebView2Blank(url, 2500*time.Millisecond)
		return
	}
	openEdgeAppOrBrowser(url)
}

func hasBundledWebView2Loader() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(filepath.Dir(exe), "WebView2Loader.dll"))
	return err == nil
}

func (a *App) fallbackIfWebView2Blank(url string, d time.Duration) {
	time.Sleep(d)
	a.mu.RLock()
	ready := a.webviewReady && a.webviewCore != 0
	a.mu.RUnlock()
	if ready {
		return
	}
	a.logf("WebView2 did not become ready; falling back to Edge app window")
	openEdgeAppOrBrowser(url)
}
func (a *App) postUIRefreshLocked() {
	if a.hwnd != 0 {
		pPostMessageW.Call(a.hwnd, refreshMsg, 0, 0)
	}
}
func (a *App) postUIRefresh() {
	if a.hwnd != 0 {
		pPostMessageW.Call(a.hwnd, refreshMsg, 0, 0)
	}
}
func (a *App) postActivityRefreshLocked() {
	if a.hwnd != 0 {
		pPostMessageW.Call(a.hwnd, activityMsg, 0, 0)
	}
}

func (a *App) refreshUI() {
	a.updateTrayTip()
	a.updateStatus()
	a.updateButtons()
	a.updateActivityControls()
	a.mu.RLock()
	recording := a.recordingMode != ""
	a.mu.RUnlock()
	if recording {
		return
	}
	keep := a.selectedRuleIndex()
	a.refreshProfileCombo()
	a.refreshRuleList(keep)
	a.updateEditorFromSelection()
}
func (a *App) updateStatus() {
	a.mu.RLock()
	status := "実行中"
	if a.emergency {
		status = "緊急停止中"
	} else if !a.enabled {
		status = "停止中"
	}
	prof := a.activeProfileNameLocked()
	rules := len(a.rules)
	rec := a.recordingMode
	cfg := a.configPath
	a.mu.RUnlock()
	hook := a.hookHealthText()
	recText := ""
	if rec == "input" {
		recText = "    記録: 入力"
	} else if rec == "output" {
		recText = "    記録: 実行内容"
	}
	txt := fmt.Sprintf("状態: %s%s    プロファイル: %s    有効ルール: %d    %s", status, recText, prof, rules, hook)
	setText(a.ctrlStatus, txt)
	setText(a.ctrlMessage, "設定: "+cfg+"    緊急停止: Ctrl+Alt+Shift+F12")
}
func (a *App) updateButtons() {
	a.mu.RLock()
	running := a.enabled && !a.emergency
	rec := a.recordingMode
	a.mu.RUnlock()
	if running {
		setText(a.btnStartStop, "変換を停止")
		setText(a.btnSafe, "変換を停止")
	} else {
		setText(a.btnStartStop, "変換を開始")
		setText(a.btnSafe, "停止状態")
	}
	if rec == "input" {
		setText(a.btnRecordInput, "● 入力を記録中")
		setText(a.btnRecordOutput, "★ 実行内容を記録")
	} else if rec == "output" {
		setText(a.btnRecordInput, "★ 入力を記録")
		setText(a.btnRecordOutput, "● 実行内容を記録中")
	} else {
		setText(a.btnRecordInput, "★ 入力を記録")
		setText(a.btnRecordOutput, "★ 実行内容を記録")
	}
}

func (a *App) updateActivityControls() {
	a.mu.RLock()
	last := a.lastInputText
	at := a.lastInputAt
	rec := a.recordingMode
	items := append([]Item(nil), a.recordedItems...)
	a.mu.RUnlock()
	if last == "" {
		last = "（まだありません）"
	} else if !at.IsZero() {
		last = fmt.Sprintf("%s    %s", last, at.Format("15:04:05"))
	}
	setText(a.ctrlLastInput, "最後に検出した入力: "+last)
	if rec == "input" {
		setText(a.editInput, itemsText(items))
		setText(a.ctrlMessage, "入力を記録中です。例: サイド1を押したままホイール上。すべて離すと自動で登録・保存します。")
	} else if rec == "output" {
		setText(a.editOutput, itemsText(items))
		setText(a.ctrlMessage, "実行内容を記録中です。例: Ctrl+Win+←。すべて離すと自動で登録・保存します。")
	}
}
func setText(hwnd uintptr, text string) {
	if hwnd != 0 {
		pSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))))
	}
}
func getText(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	n, _, _ := pSendMessageW.Call(hwnd, WM_GETTEXTLENGTH, 0, 0)
	buf := make([]uint16, int(n)+2)
	pSendMessageW.Call(hwnd, WM_GETTEXT, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
}
func setCheck(hwnd uintptr, checked bool) {
	v := uintptr(BST_UNCHECKED)
	if checked {
		v = BST_CHECKED
	}
	pSendMessageW.Call(hwnd, BM_SETCHECK, v, 0)
}
func getCheck(hwnd uintptr) bool {
	r, _, _ := pSendMessageW.Call(hwnd, BM_GETCHECK, 0, 0)
	return r == BST_CHECKED
}
func comboAdd(hwnd uintptr, text string) {
	pSendMessageW.Call(hwnd, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))))
}

func (a *App) refreshProfileCombo() {
	if a.ctrlProfile == 0 {
		return
	}
	cur, _, _ := pSendMessageW.Call(a.ctrlProfile, CB_GETCURSEL, 0, 0)
	pSendMessageW.Call(a.ctrlProfile, CB_RESETCONTENT, 0, 0)
	a.mu.RLock()
	idx := a.activeProfileIndex
	profiles := append([]Profile(nil), a.config.Profiles...)
	a.mu.RUnlock()
	for _, p := range profiles {
		comboAdd(a.ctrlProfile, p.Name)
	}
	if idx >= 0 && idx < len(profiles) {
		pSendMessageW.Call(a.ctrlProfile, CB_SETCURSEL, uintptr(idx), 0)
	} else if cur != ^uintptr(0) {
		pSendMessageW.Call(a.ctrlProfile, CB_SETCURSEL, cur, 0)
	}
}
func (a *App) changeProfileFromCombo() {
	r, _, _ := pSendMessageW.Call(a.ctrlProfile, CB_GETCURSEL, 0, 0)
	idx := int(r)
	a.mu.Lock()
	if idx >= 0 && idx < len(a.config.Profiles) {
		a.config.ActiveProfileId = a.config.Profiles[idx].Id
		err := a.saveConfigLocked()
		a.rebuildRulesLocked()
		a.postUIRefreshLocked()
		a.mu.Unlock()
		if err != nil {
			messageBox("保存失敗", err.Error())
		}
		return
	}
	a.mu.Unlock()
}

func (a *App) handleRuleListCustomDraw(lParam uintptr) uintptr {
	cd := (*NMLVCUSTOMDRAW)(unsafe.Pointer(lParam))
	if cd == nil {
		return CDRF_DODEFAULT
	}
	switch cd.Nmcd.DwDrawStage {
	case CDDS_PREPAINT:
		return CDRF_NOTIFYITEMDRAW
	case CDDS_ITEMPREPAINT:
		return CDRF_NOTIFYSUBITEMDRAW
	case CDDS_ITEMPREPAINT | CDDS_SUBITEM:
		col := int(cd.ISubItem)
		if col != 0 && col != 4 && col != 5 {
			return CDRF_DODEFAULT
		}
		row := int(cd.Nmcd.DwItemSpec)
		checked, enabled := a.checkStateForCell(row, col)
		a.drawCheckboxCell(cd.Nmcd.Hdc, cd.Nmcd.Rc, checked, enabled, (cd.Nmcd.UItemState&CDIS_SELECTED) != 0)
		return CDRF_SKIPDEFAULT
	}
	return CDRF_DODEFAULT
}

func (a *App) checkStateForCell(row, col int) (bool, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.config.Profiles) == 0 || a.activeProfileIndex >= len(a.config.Profiles) || row < 0 || row >= len(a.config.Profiles[a.activeProfileIndex].Rules) {
		return false, false
	}
	r := a.config.Profiles[a.activeProfileIndex].Rules[row]
	switch col {
	case 0:
		return r.Enabled, true
	case 4:
		if isLastInputPrimaryMouse(r.Input) {
			return false, false
		}
		return r.SuppressTrigger, true
	case 5:
		if !hasSidePrefix(r.Input) {
			return false, false
		}
		return r.SuppressPrefix, true
	}
	return false, false
}

func (a *App) drawCheckboxCell(hdc uintptr, rc RECT, checked bool, enabled bool, selected bool) {
	bg := uint32(pGetSysColorValue(COLOR_WINDOW))
	if selected {
		bg = uint32(pGetSysColorValue(COLOR_HIGHLIGHT))
	}
	fill(hdc, rc, bg)
	// セル境界。ListView既定描画をスキップするため、薄い罫線だけこちらで戻す。
	gridBrush := brush(rgb(210, 210, 210))
	pFrameRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), gridBrush)
	pDeleteObject.Call(gridBrush)
	box := int32(14)
	cellW := rc.Right - rc.Left
	cellH := rc.Bottom - rc.Top
	if cellW < box || cellH < box {
		return
	}
	brc := RECT{}
	brc.Left = rc.Left + (cellW-box)/2
	brc.Top = rc.Top + (cellH-box)/2
	brc.Right = brc.Left + box
	brc.Bottom = brc.Top + box
	state := uint32(DFCS_BUTTONCHECK)
	if checked {
		state |= DFCS_CHECKED
	}
	if !enabled {
		state |= DFCS_INACTIVE
	}
	pDrawFrameControl.Call(hdc, uintptr(unsafe.Pointer(&brc)), DFC_BUTTON, uintptr(state))
}

func pGetSysColorValue(index int) uintptr {
	r, _, _ := pGetSysColor.Call(uintptr(index))
	return r
}

func (a *App) handleRuleListCellClick(row, col int) bool {
	if row < 0 {
		return false
	}
	if col != 0 && col != 4 && col != 5 {
		return false
	}
	a.mu.Lock()
	rules := a.activeRulesSliceLocked()
	if rules == nil || row >= len(*rules) {
		a.mu.Unlock()
		return false
	}
	r := &(*rules)[row]
	switch col {
	case 0:
		r.Enabled = !r.Enabled
	case 4:
		if isLastInputPrimaryMouse(r.Input) {
			r.SuppressTrigger = false
		} else {
			r.SuppressTrigger = !r.SuppressTrigger
		}
	case 5:
		if hasSidePrefix(r.Input) {
			r.SuppressPrefix = !r.SuppressPrefix
		} else {
			r.SuppressPrefix = false
		}
	}
	err := a.saveConfigLocked()
	a.rebuildRulesWithoutJoyConRescanLocked()
	a.postUIRefreshLocked()
	a.mu.Unlock()
	if err != nil {
		messageBox("保存失敗", err.Error())
		return true
	}
	a.refreshRuleList(row)
	a.updateEditorFromSelection()
	setText(a.ctrlMessage, "チェックを切り替えて保存しました。")
	return true
}

func isLastInputPrimaryMouse(items []Item) bool {
	if len(items) == 0 {
		return false
	}
	last := items[len(items)-1]
	return strings.EqualFold(last.Kind, "Mouse") && isPrimaryButton(normMouse(last.Code))
}

func hasSidePrefix(items []Item) bool {
	if len(items) < 2 {
		return false
	}
	for i := 0; i < len(items)-1; i++ {
		if strings.EqualFold(items[i].Kind, "Mouse") {
			c := normMouse(items[i].Code)
			if c == "X1" || c == "X2" {
				return true
			}
		}
	}
	return false
}

func (a *App) updateSelectedFlagsFromEditor() {
	idx := a.selectedRuleIndex()
	if idx < 0 {
		return
	}
	a.mu.Lock()
	rules := a.activeRulesSliceLocked()
	if rules == nil || idx >= len(*rules) {
		a.mu.Unlock()
		return
	}
	r := &(*rules)[idx]
	r.Enabled = getCheck(a.chkEnabled)
	if isLastInputPrimaryMouse(r.Input) {
		r.SuppressTrigger = false
	} else {
		r.SuppressTrigger = getCheck(a.chkSuppressTrigger)
	}
	if hasSidePrefix(r.Input) {
		r.SuppressPrefix = getCheck(a.chkSuppressPrefix)
	} else {
		r.SuppressPrefix = false
	}
	err := a.saveConfigLocked()
	a.rebuildRulesWithoutJoyConRescanLocked()
	a.postUIRefreshLocked()
	a.mu.Unlock()
	if err != nil {
		messageBox("保存失敗", err.Error())
		return
	}
	a.refreshRuleList(idx)
	a.updateEditorFromSelection()
	setText(a.ctrlMessage, "チェックを切り替えて保存しました。")
}

func (a *App) selectedRuleIndex() int {
	if a.ctrlList == 0 {
		return -1
	}
	r, _, _ := pSendMessageW.Call(a.ctrlList, LVM_GETNEXTITEM, ^uintptr(0), uintptr(LVNI_SELECTED))
	if int32(r) < 0 {
		return -1
	}
	return int(r)
}
func (a *App) setSelectedRuleIndex(idx int) {
	if a.ctrlList == 0 || idx < 0 {
		return
	}
	item := LVITEMW{State: LVIS_SELECTED | LVIS_FOCUSED, StateMask: LVIS_SELECTED | LVIS_FOCUSED}
	pSendMessageW.Call(a.ctrlList, LVM_SETITEMSTATE, uintptr(idx), uintptr(unsafe.Pointer(&item)))
	pSendMessageW.Call(a.ctrlList, LVM_ENSUREVISIBLE, uintptr(idx), 0)
}
func (a *App) refreshRuleList(keep int) {
	if a.ctrlList == 0 {
		return
	}
	pSendMessageW.Call(a.ctrlList, LVM_DELETEALLITEMS, 0, 0)
	a.mu.RLock()
	rules := []Rule{}
	if len(a.config.Profiles) > 0 && a.activeProfileIndex < len(a.config.Profiles) {
		rules = append(rules, a.config.Profiles[a.activeProfileIndex].Rules...)
	}
	a.mu.RUnlock()
	for i, r := range rules {
		cols := []string{"", itemsText(r.Input), modeText(r.Mode), itemsText(r.Output), "", ""}
		lvInsertRow(a.ctrlList, i, cols)
	}
	if keep >= 0 && keep < len(rules) {
		a.setSelectedRuleIndex(keep)
	} else if len(rules) > 0 {
		a.setSelectedRuleIndex(0)
	}
}
func onoff(b bool) string {
	if b {
		return "☑"
	}
	return "☐"
}
func yesno(b bool) string {
	if b {
		return "☑"
	}
	return "☐"
}
func modeText(m string) string {
	if strings.EqualFold(m, joyConRuleModeHold) {
		return "押している間キーを保持する"
	}
	if strings.EqualFold(m, "Tap") || m == "" {
		return "1回だけ実行する"
	}
	return m
}

func (a *App) updateEditorFromSelection() {
	idx := a.selectedRuleIndex()
	a.mu.RLock()
	if len(a.config.Profiles) == 0 || a.activeProfileIndex >= len(a.config.Profiles) || idx < 0 || idx >= len(a.config.Profiles[a.activeProfileIndex].Rules) {
		a.mu.RUnlock()
		setText(a.editInput, "")
		setText(a.editOutput, "")
		setCheck(a.chkEnabled, false)
		setCheck(a.chkSuppressTrigger, false)
		setCheck(a.chkSuppressPrefix, false)
		return
	}
	r := a.config.Profiles[a.activeProfileIndex].Rules[idx]
	a.mu.RUnlock()
	setCheck(a.chkEnabled, r.Enabled)
	setText(a.editInput, itemsText(r.Input))
	setText(a.editOutput, itemsText(r.Output))
	setCheck(a.chkSuppressTrigger, r.SuppressTrigger)
	setCheck(a.chkSuppressPrefix, r.SuppressPrefix)
}
func itemsText(items []Item) string {
	out := []string{}
	for _, it := range items {
		if strings.EqualFold(it.Kind, "Mouse") {
			out = append(out, mouseName(normMouse(it.Code)))
		} else if strings.EqualFold(it.Kind, "Key") {
			if vk, ok := parseVK(it.Code); ok {
				out = append(out, vkName(vk))
			} else {
				out = append(out, "Key("+it.Code+")")
			}
		} else if strings.EqualFold(it.Kind, "JoyCon") {
			out = append(out, "Joy-Con "+joyConCodeText(it.Code))
		} else if strings.EqualFold(it.Kind, "XInput") {
			out = append(out, "XInput "+xInputCodeText(it.Code))
		} else {
			out = append(out, it.Kind+"("+it.Code+")")
		}
	}
	return strings.Join(out, " + ")
}
func mouseName(code string) string {
	switch code {
	case "Left":
		return "左クリック"
	case "Right":
		return "右クリック"
	case "Middle":
		return "中クリック"
	case "X1":
		return "サイド1"
	case "X2":
		return "サイド2"
	case "WheelUp":
		return "ホイール上"
	case "WheelDown":
		return "ホイール下"
	}
	return code
}
func vkName(vk uint32) string {
	switch genericVK(vk) {
	case VK_CONTROL:
		return "Ctrl"
	case VK_SHIFT:
		return "Shift"
	case VK_MENU:
		return "Alt"
	}
	switch vk {
	case VK_LWIN, VK_RWIN:
		return "Win"
	case VK_LEFT:
		return "←"
	case VK_RIGHT:
		return "→"
	case VK_UP:
		return "↑"
	case VK_DOWN:
		return "↓"
	case VK_PRIOR:
		return "PageUp"
	case VK_NEXT:
		return "PageDown"
	case VK_RETURN:
		return "Enter"
	case VK_SPACE:
		return "Space"
	case VK_TAB:
		return "Tab"
	case VK_ESCAPE:
		return "Esc"
	case VK_BACK:
		return "Backspace"
	case VK_SNAPSHOT:
		return "PrintScreen"
	}
	if vk >= 'A' && vk <= 'Z' {
		return string(rune(vk))
	}
	if vk >= '0' && vk <= '9' {
		return string(rune(vk))
	}
	if vk >= 0x70 && vk <= 0x87 {
		return fmt.Sprintf("F%d", vk-0x6F)
	}
	return fmt.Sprintf("VK%d", vk)
}

func (a *App) handleCommand(id uint32) {
	switch id {
	case ID_BTN_STARTSTOP, ID_TRAY_STARTSTOP:
		a.mu.Lock()
		if a.enabled && !a.emergency {
			a.enabled = false
			a.abortAllLongPressLocked("conversion stopped", false)
			a.clearJoyConInputStateLocked("conversion stopped")
		} else {
			a.enabled = true
			a.emergency = false
		}
		a.postUIRefreshLocked()
		a.mu.Unlock()
		a.releaseJoyConHeldOutputs()
		a.logf("toggle running")
	case ID_BTN_EMERGENCY, ID_TRAY_EMERGENCY, ID_BTN_SAFE:
		a.mu.Lock()
		a.enabled = false
		a.emergency = true
		a.abortAllLongPressLocked("emergency stop from UI", false)
		a.clearJoyConInputStateLocked("emergency stop from UI")
		a.postUIRefreshLocked()
		a.mu.Unlock()
		a.ReleaseModifiersNow()
		a.logf("emergency stop from UI")
	case ID_BTN_RELEASE, ID_TRAY_RELEASE:
		a.ReleaseModifiersNow()
		messageBox("MouseButtonMapper", "修飾キー解放を送信しました。")
	case ID_BTN_RELOAD, ID_TRAY_RELOAD, ID_BTN_REVERT_RULE:
		if err := a.loadConfig(); err != nil {
			messageBox("設定再読み込みエラー", err.Error())
		} else {
			a.refreshUI()
			messageBox("MouseButtonMapper", "設定を再読み込みしました。")
		}
	case ID_BTN_OPENCFG, ID_TRAY_OPENCFG:
		a.openConfig()
	case ID_BTN_OPENFOLDER, ID_TRAY_OPENFOLDER:
		_ = exec.Command("explorer.exe", filepath.Dir(a.configPath)).Start()
	case ID_BTN_OPENLOG:
		_ = exec.Command("notepad.exe", a.logPath).Start()
	case ID_BTN_ENABLE_RULE:
		a.setSelectedRuleEnabled(true)
	case ID_BTN_DISABLE_RULE:
		a.setSelectedRuleEnabled(false)
	case ID_BTN_SAVE_RULE:
		a.saveSelectedRuleFromEditor()
	case ID_BTN_ADD_RULE:
		a.addRule()
	case ID_BTN_DUP_RULE:
		a.duplicateRule()
	case ID_BTN_DELETE_RULE:
		a.deleteSelectedRule()
	case ID_BTN_MOVE_TOP:
		a.moveSelectedRuleTo(0)
	case ID_BTN_MOVE_UP:
		a.moveSelectedRule(-1)
	case ID_BTN_MOVE_DOWN:
		a.moveSelectedRule(1)
	case ID_BTN_MOVE_BOTTOM:
		a.moveSelectedRuleTo(1 << 30)
	case ID_BTN_TEST_OUTPUT:
		a.testOutputFromEditor()
	case ID_CHK_ENABLED, ID_CHK_SUPPRESS_TRIGGER, ID_CHK_SUPPRESS_PREFIX:
		a.updateSelectedFlagsFromEditor()
	case ID_BTN_RECORD_INPUT:
		a.startRecording("input")
	case ID_BTN_RECORD_OUTPUT:
		a.startRecording("output")
	case ID_BTN_RECORD_STOP:
		a.stopRecording()
	case ID_BTN_DEFAULT_RULES:
		a.showDefaultRules()
	case ID_BTN_EXPORT:
		if p, err := a.exportConfig(); err != nil {
			messageBox("エクスポート失敗", err.Error())
		} else {
			messageBox("MouseButtonMapper", "設定をエクスポートしました。\n"+p)
		}
	case ID_BTN_IMPORT:
		if err := a.importConfigFromClipboardPath(); err != nil {
			messageBox("インポート", err.Error())
		}
	case ID_BTN_COPY_DIAG:
		if err := a.copyDiagnosticLogToClipboard(); err != nil {
			messageBox("診断ログ", err.Error())
		} else {
			setText(a.ctrlMessage, "診断ログをクリップボードにコピーしました。")
		}
	case ID_TRAY_SHOW:
		a.showMainWindow()
	case ID_TRAY_ABOUT:
		messageBox("MouseButtonMapper", appName+" "+appVersion+"\n\nGo製の単体EXE版です。\nGUI: EXEを二重起動またはトレイ左クリックで表示\n緊急停止: Ctrl+Alt+Shift+F12\n設定: "+a.configPath+"\nログ: "+a.logPath)
	case ID_BTN_QUIT, ID_TRAY_EXIT:
		if a.hwnd != 0 {
			pDestroyWindow.Call(a.hwnd)
		}
	}
}

func (a *App) startRecording(mode string) {
	idx := a.selectedRuleIndex()
	if idx < 0 {
		messageBox("MouseButtonMapper", "先に記録先のルールを選択してください。新規なら『ルールを追加』を押してください。")
		return
	}
	a.mu.Lock()
	a.recordingMode = mode
	a.recordingRuleIndex = idx
	if a.activeProfileIndex >= 0 && a.activeProfileIndex < len(a.config.Profiles) {
		a.recordingProfileID = a.config.Profiles[a.activeProfileIndex].Id
	}
	a.recordedItems = nil
	a.recordHeld = map[string]bool{}
	a.lastInputText = ""
	a.lastInputAt = time.Time{}
	a.postUIRefreshLocked()
	a.mu.Unlock()
	if mode == "input" {
		setText(a.ctrlMessage, "入力の記録を開始しました。サイドボタン、ホイール、キーを実際に操作してください。すべて離すと自動登録します。")
	} else {
		setText(a.ctrlMessage, "実行内容の記録を開始しました。Ctrl+V のように、送りたいショートカットを押してください。すべて離すと自動登録します。")
	}
}

func (a *App) stopRecording() {
	a.mu.Lock()
	mode := a.recordingMode
	a.recordingMode = ""
	a.recordingRuleIndex = -1
	a.recordingProfileID = ""
	a.recordedItems = nil
	a.recordHeld = map[string]bool{}
	a.postUIRefreshLocked()
	a.mu.Unlock()
	if mode == "" {
		setText(a.ctrlMessage, "記録中ではありません。")
	} else {
		setText(a.ctrlMessage, "記録を中止しました。")
	}
}

func (a *App) finishRecordingAuto() {
	// フック処理を重くしないため、hook側では完了判定だけ行い、保存は別goroutineで実行する。
	a.mu.Lock()
	mode := a.recordingMode
	idx := a.recordingRuleIndex
	profileID := a.recordingProfileID
	items := append([]Item(nil), a.recordedItems...)
	reset := func() {
		a.recordingMode = ""
		a.recordingRuleIndex = -1
		a.recordingProfileID = ""
		a.recordedItems = nil
		a.recordHeld = map[string]bool{}
		a.postUIRefreshLocked()
	}
	if mode == "" || idx < 0 || len(items) == 0 {
		reset()
		a.mu.Unlock()
		return
	}
	profileIdx := a.profileIndexByIDLocked(profileID)
	if profileIdx < 0 || idx >= len(a.config.Profiles[profileIdx].Rules) {
		reset()
		a.mu.Unlock()
		return
	}
	rules := &a.config.Profiles[profileIdx].Rules
	updated := cloneRule((*rules)[idx])
	if mode == "input" {
		updated.Input = items
	} else if mode == "output" {
		updated.Output = items
	} else if mode == "long-output" {
		updated.LongPressEnabled = true
		updated.LongPressAction = longPressActionExecute
		updated.LongPressMs = normalizeLongPressMs(updated.LongPressMs)
		updated.LongPressOutput = items
	}
	if err := validateJoyConHoldRule(updated); err != nil {
		reset()
		a.mu.Unlock()
		messageBox("記録内容を保存できません", err.Error())
		return
	}
	if err := validateLongPressRule(updated); err != nil {
		reset()
		a.mu.Unlock()
		messageBox("記録内容を保存できません", err.Error())
		return
	}
	(*rules)[idx] = updated
	err := a.saveConfigLocked()
	a.rebuildRulesWithoutJoyConRescanLocked()
	reset()
	a.mu.Unlock()
	if err != nil {
		messageBox("記録の保存に失敗", err.Error())
		return
	}
	if mode == "input" {
		setText(a.ctrlMessage, "入力を記録して保存しました: "+itemsText(items))
	} else if mode == "long-output" {
		setText(a.ctrlMessage, "長押し時の実行内容を記録して保存しました: "+itemsText(items))
	} else {
		setText(a.ctrlMessage, "短押し時の実行内容を記録して保存しました: "+itemsText(items))
	}
}

func (a *App) showDefaultRules() {
	cfg := mustDefaultConfig()
	lines := []string{"既定ルール（表示のみ・現在設定は上書きしません）", ""}
	if len(cfg.Profiles) > 0 {
		for i, r := range cfg.Profiles[0].Rules {
			lines = append(lines, fmt.Sprintf("%02d. %s  →  %s", i+1, itemsText(r.Input), itemsText(r.Output)))
		}
	}
	msg := strings.Join(lines, "\n")
	if len(msg) > 1800 {
		msg = msg[:1800] + "\n..."
	}
	messageBox("既定ルール", msg)
}

func (a *App) activeRulesSliceLocked() *[]Rule {
	if len(a.config.Profiles) == 0 || a.activeProfileIndex >= len(a.config.Profiles) {
		return nil
	}
	return &a.config.Profiles[a.activeProfileIndex].Rules
}
func (a *App) editorRulesSliceLocked() *[]Rule {
	if len(a.config.Profiles) == 0 || a.editorProfileIndex < 0 || a.editorProfileIndex >= len(a.config.Profiles) {
		return nil
	}
	return &a.config.Profiles[a.editorProfileIndex].Rules
}

func (a *App) saveSelectedRuleFromEditor() {
	idx := a.selectedRuleIndex()
	if idx < 0 {
		messageBox("MouseButtonMapper", "保存するルールを選択してください。")
		return
	}
	input, err := parseItemsText(getText(a.editInput), true, true)
	if err != nil {
		messageBox("入力の解釈に失敗", err.Error())
		return
	}
	output, err := parseItemsText(getText(a.editOutput), false, true)
	if err != nil {
		messageBox("出力の解釈に失敗", err.Error())
		return
	}
	if len(input) == 0 || len(output) == 0 {
		messageBox("MouseButtonMapper", "入力と出力はどちらも1つ以上必要です。")
		return
	}
	r := Rule{Enabled: getCheck(a.chkEnabled), Input: input, Mode: "Tap", Output: output, SuppressTrigger: getCheck(a.chkSuppressTrigger), SuppressPrefix: getCheck(a.chkSuppressPrefix)}
	if isLastInputPrimaryMouse(r.Input) {
		r.SuppressTrigger = false
	}
	if !hasSidePrefix(r.Input) {
		r.SuppressPrefix = false
	}
	a.mu.Lock()
	a.recordingMode = ""
	a.recordedItems = nil
	rules := a.activeRulesSliceLocked()
	if rules == nil || idx >= len(*rules) {
		a.mu.Unlock()
		return
	}
	// 旧Win32編集画面には長押し欄がないため、そこで通常項目を保存しても
	// Web GUIで設定した長押し内容を失わせない。
	r.LongPressEnabled = (*rules)[idx].LongPressEnabled
	r.LongPressMs = (*rules)[idx].LongPressMs
	r.LongPressAction = (*rules)[idx].LongPressAction
	r.LongPressOutput = append([]Item(nil), (*rules)[idx].LongPressOutput...)
	(*rules)[idx] = r
	err = a.saveConfigLocked()
	a.rebuildRulesWithoutJoyConRescanLocked()
	a.postUIRefreshLocked()
	a.mu.Unlock()
	if err != nil {
		messageBox("保存失敗", err.Error())
	} else {
		a.refreshRuleList(idx)
		setText(a.ctrlMessage, "選択中の割り当てを保存しました。")
	}
}
func (a *App) addRule() {
	r := Rule{Enabled: true, Input: []Item{{Kind: "Mouse", Code: "X1"}}, Mode: "Tap", Output: []Item{{Kind: "Key", Code: strconv.Itoa(int(VK_CONTROL))}, {Kind: "Key", Code: "86"}}, SuppressTrigger: true, SuppressPrefix: false}
	a.mu.Lock()
	rules := a.activeRulesSliceLocked()
	if rules == nil {
		a.mu.Unlock()
		return
	}
	*rules = append(*rules, r)
	idx := len(*rules) - 1
	err := a.saveConfigLocked()
	a.rebuildRulesWithoutJoyConRescanLocked()
	a.postUIRefreshLocked()
	a.mu.Unlock()
	if err != nil {
		messageBox("保存失敗", err.Error())
	} else {
		a.refreshRuleList(idx)
		a.updateEditorFromSelection()
		setText(a.ctrlMessage, "ルールを追加しました。")
	}
}
func (a *App) duplicateRule() {
	idx := a.selectedRuleIndex()
	if idx < 0 {
		return
	}
	a.mu.Lock()
	rules := a.activeRulesSliceLocked()
	if rules == nil || idx >= len(*rules) {
		a.mu.Unlock()
		return
	}
	r := (*rules)[idx]
	insert := idx + 1
	*rules = append((*rules)[:insert], append([]Rule{r}, (*rules)[insert:]...)...)
	err := a.saveConfigLocked()
	a.rebuildRulesWithoutJoyConRescanLocked()
	a.postUIRefreshLocked()
	a.mu.Unlock()
	if err != nil {
		messageBox("保存失敗", err.Error())
	} else {
		a.refreshRuleList(insert)
		a.updateEditorFromSelection()
	}
}
func (a *App) deleteSelectedRule() {
	idx := a.selectedRuleIndex()
	if idx < 0 {
		return
	}
	a.mu.Lock()
	rules := a.activeRulesSliceLocked()
	if rules == nil || idx >= len(*rules) {
		a.mu.Unlock()
		return
	}
	*rules = append((*rules)[:idx], (*rules)[idx+1:]...)
	if idx >= len(*rules) {
		idx = len(*rules) - 1
	}
	err := a.saveConfigLocked()
	a.rebuildRulesWithoutJoyConRescanLocked()
	a.postUIRefreshLocked()
	a.mu.Unlock()
	if err != nil {
		messageBox("保存失敗", err.Error())
	} else {
		a.refreshRuleList(idx)
		a.updateEditorFromSelection()
	}
}

func (a *App) moveSelectedRuleTo(target int) {
	idx := a.selectedRuleIndex()
	if idx < 0 {
		return
	}
	a.mu.Lock()
	rules := a.activeRulesSliceLocked()
	if rules == nil || idx >= len(*rules) {
		a.mu.Unlock()
		return
	}
	if len(*rules) == 0 {
		a.mu.Unlock()
		return
	}
	if target < 0 {
		target = 0
	}
	if target >= len(*rules) {
		target = len(*rules) - 1
	}
	if idx == target {
		a.mu.Unlock()
		return
	}
	r := (*rules)[idx]
	without := append((*rules)[:idx], (*rules)[idx+1:]...)
	if target > len(without) {
		target = len(without)
	}
	newRules := append(without[:target], append([]Rule{r}, without[target:]...)...)
	*rules = newRules
	err := a.saveConfigLocked()
	a.rebuildRulesWithoutJoyConRescanLocked()
	a.postUIRefreshLocked()
	a.mu.Unlock()
	if err != nil {
		messageBox("保存失敗", err.Error())
	} else {
		a.refreshRuleList(target)
		a.updateEditorFromSelection()
	}
}

func (a *App) importConfigFromClipboardPath() error {
	// Go単体EXEのまま、標準のファイル選択だけPowerShellのWinFormsに任せる。
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "Add-Type -AssemblyName System.Windows.Forms; $d=New-Object System.Windows.Forms.OpenFileDialog; $d.Filter='JSON (*.json)|*.json|All files (*.*)|*.*'; $d.Title='MouseButtonMapper 設定をインポート'; if($d.ShowDialog() -eq 'OK'){ [Console]::Write($d.FileName) }")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ファイル選択を開けませんでした: %w", err)
	}
	src := strings.TrimSpace(string(out))
	if src == "" {
		return fmt.Errorf("インポートをキャンセルしました。")
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	var cfg Config
	if err := json.Unmarshal(stripBOM(b), &cfg); err != nil {
		return err
	}
	if len(cfg.Profiles) == 0 {
		return fmt.Errorf("プロファイルが入っていないためインポートできません。")
	}
	if err := a.backupConfig(); err != nil {
		a.logf("backup before import failed: %v", err)
	}
	a.mu.Lock()
	a.abortAllLongPressLocked("configuration imported", false)
	a.clearControllerInputStateLocked("configuration imported")
	a.config = normalizeConfig(cfg)
	a.editorProfileIndex = a.profileIndexByIDLocked(a.config.ActiveProfileId)
	a.clearAutoMatchLocked()
	if err := a.saveConfigLocked(); err != nil {
		a.mu.Unlock()
		return err
	}
	a.rebuildRulesLocked()
	a.postUIRefreshLocked()
	a.mu.Unlock()
	a.releaseJoyConHeldOutputs()
	a.syncControllerSubsystems()
	setText(a.ctrlMessage, "設定をインポートして適用しました: "+src)
	return nil
}

func (a *App) copyDiagnosticLogToClipboard() error {
	a.mu.RLock()
	lines := []string{
		appName + " " + appVersion,
		"Config: " + a.configPath,
		"Log: " + a.logPath,
		fmt.Sprintf("Enabled=%v Emergency=%v Profile=%s ActiveRules=%d", a.enabled, a.emergency, a.activeProfileNameLocked(), len(a.rules)),
		fmt.Sprintf("ControllerFeature=%v JoyConWorker=%v XInputWorker=%v", a.config.Controller.Enabled, a.joyConWorker != nil, a.xInputCancel != nil),
		"JoyConStatus: " + a.joyConStatusTextLocked(),
		"XInputStatus: " + a.xInputStatusTextLocked(),
		"HookHealth: " + a.hookHealthText(),
		"",
		"Rules:",
	}
	if len(a.config.Profiles) > 0 && a.activeProfileIndex < len(a.config.Profiles) {
		for i, r := range a.config.Profiles[a.activeProfileIndex].Rules {
			lines = append(lines, fmt.Sprintf("%02d enabled=%v mode=%s suppressLast=%v cancelSideSingle=%v input=%s shortOutput=%s longPress=%v longMs=%d longAction=%s longOutput=%s", i+1, r.Enabled, r.Mode, r.SuppressTrigger, r.SuppressPrefix, itemsText(r.Input), itemsText(r.Output), r.LongPressEnabled, normalizeLongPressMs(r.LongPressMs), normalizeLongPressAction(r.LongPressAction), itemsText(r.LongPressOutput)))
		}
	}
	a.mu.RUnlock()
	if b, err := os.ReadFile(a.logPath); err == nil {
		lines = append(lines, "", "Log tail:", tailString(string(b), 12000))
	}
	cmd := exec.Command("clip.exe")
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\r\n"))
	return cmd.Run()
}

func tailString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[len(s)-max:]
}

func (a *App) moveSelectedRule(delta int) {
	idx := a.selectedRuleIndex()
	if idx < 0 {
		return
	}
	ni := idx + delta
	a.mu.Lock()
	rules := a.activeRulesSliceLocked()
	if rules == nil || idx >= len(*rules) || ni < 0 || ni >= len(*rules) {
		a.mu.Unlock()
		return
	}
	(*rules)[idx], (*rules)[ni] = (*rules)[ni], (*rules)[idx]
	err := a.saveConfigLocked()
	a.rebuildRulesWithoutJoyConRescanLocked()
	a.postUIRefreshLocked()
	a.mu.Unlock()
	if err != nil {
		messageBox("保存失敗", err.Error())
	} else {
		a.refreshRuleList(ni)
		a.updateEditorFromSelection()
	}
}
func (a *App) testOutputFromEditor() {
	output, err := parseItemsText(getText(a.editOutput), false, true)
	if err != nil {
		messageBox("出力の解釈に失敗", err.Error())
		return
	}
	keys := []uint32{}
	for _, it := range output {
		if vk, ok := parseVK(it.Code); ok {
			keys = append(keys, vk)
		}
	}
	if len(keys) > 0 {
		a.sendShortcut(keys)
	}
}

func parseItemsText(text string, allowMouse bool, allowKey bool) ([]Item, error) {
	repl := strings.NewReplacer("＋", "+", "，", ",", "、", ",", "／", "/", "←", "Left", "→", "Right", "↑", "Up", "↓", "Down")
	text = repl.Replace(text)
	text = strings.ReplaceAll(text, ",", "+")
	parts := strings.Split(text, "+")
	items := []Item{}
	for _, part := range parts {
		tok := strings.TrimSpace(part)
		if tok == "" {
			continue
		}
		if allowMouse {
			if xinput, ok := parseXInputToken(tok); ok {
				items = append(items, Item{Kind: "XInput", Code: xinput})
				continue
			}
			if joy, ok := parseJoyConToken(tok); ok {
				items = append(items, Item{Kind: "JoyCon", Code: joy})
				continue
			}
			if m, ok := parseMouseToken(tok); ok {
				items = append(items, Item{Kind: "Mouse", Code: m})
				continue
			}
		}
		if allowKey {
			if vk, ok := parseVK(tok); ok {
				items = append(items, Item{Kind: "Key", Code: strconv.Itoa(int(vk))})
				continue
			}
		}
		return nil, fmt.Errorf("%q を解釈できません。例: サイド1 + ホイール上 / Ctrl + Win + ←", tok)
	}
	return items, nil
}
func parseXInputToken(tok string) (string, bool) {
	clean := strings.ToLower(strings.TrimSpace(tok))
	clean = strings.ReplaceAll(clean, " ", "")
	hasPrefix := strings.HasPrefix(clean, "xinput") || strings.HasPrefix(clean, "gamepad") || strings.HasPrefix(clean, "パッド")
	if !hasPrefix {
		return "", false
	}
	clean = strings.TrimPrefix(clean, "xinput")
	clean = strings.TrimPrefix(clean, "gamepad")
	clean = strings.TrimPrefix(clean, "パッド")
	clean = strings.TrimLeft(clean, ":/：")
	code := normalizeXInputCode(clean)
	if isKnownXInputCode(code) {
		return code, true
	}
	return "", false
}

func parseJoyConToken(tok string) (string, bool) {
	clean := strings.ToLower(strings.TrimSpace(tok))
	clean = strings.ReplaceAll(clean, " ", "")
	hasPrefix := strings.Contains(clean, "joy-con") || strings.Contains(clean, "joycon") || strings.HasPrefix(clean, "左joy")
	if !hasPrefix {
		return "", false
	}
	clean = strings.ReplaceAll(clean, "joy-con", "")
	clean = strings.ReplaceAll(clean, "joycon", "")
	clean = strings.TrimPrefix(clean, "(l)")
	clean = strings.TrimPrefix(clean, "（l）")
	clean = strings.TrimPrefix(clean, "左")
	code := normalizeJoyConCode(clean)
	if isKnownJoyConCode(code) {
		return code, true
	}
	return "", false
}

func parseMouseToken(tok string) (string, bool) {
	c := strings.ToLower(strings.TrimSpace(tok))
	c = strings.ReplaceAll(c, " ", "")
	switch c {
	case "left", "lbutton", "左クリック", "左":
		return "Left", true
	case "right", "rbutton", "右クリック", "右":
		return "Right", true
	case "middle", "mbutton", "中クリック", "中":
		return "Middle", true
	case "x1", "back", "戻る", "side1", "サイド1", "サイドボタン1":
		return "X1", true
	case "x2", "forward", "進む", "side2", "サイド2", "サイドボタン2":
		return "X2", true
	case "wheelup", "wheel_up", "ホイール上", "上ホイール":
		return "WheelUp", true
	case "wheeldown", "wheel_down", "ホイール下", "下ホイール":
		return "WheelDown", true
	}
	return "", false
}

func (a *App) backupConfig() error {
	if _, err := os.Stat(a.configPath); err != nil {
		return nil
	}
	bakDir := filepath.Join(filepath.Dir(a.configPath), "backups")
	_ = os.MkdirAll(bakDir, 0755)
	dst := filepath.Join(bakDir, "config_backup_"+time.Now().Format("20060102_150405")+".json")
	return copyFile(a.configPath, dst)
}
func (a *App) saveConfigLocked() error {
	// フックコールバックと共有するa.muを保持したままディスクI/Oをしない。
	// ここでは完全なJSONスナップショットだけを作り、専用ワーカーへ渡す。
	if a.config.Version < 10 {
		a.config.Version = 10
	}
	a.config.SavedBy = appVersion
	a.config.SavedAt = time.Now().Format(time.RFC3339)
	b, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		return err
	}
	data := append([]byte(nil), b...)
	select {
	case a.configSaveCh <- data:
		return nil
	default:
		// 最新スナップショットが完全な状態を含むため、古い未書き込み分を1件捨てて差し替える。
		select {
		case <-a.configSaveCh:
		default:
		}
		select {
		case a.configSaveCh <- data:
			return nil
		default:
			return fmt.Errorf("設定保存キューが混雑しています")
		}
	}
}
func (a *App) setSelectedRuleEnabled(enabled bool) {
	idx := a.selectedRuleIndex()
	if idx < 0 {
		return
	}
	a.mu.Lock()
	if len(a.config.Profiles) == 0 || a.activeProfileIndex >= len(a.config.Profiles) || idx >= len(a.config.Profiles[a.activeProfileIndex].Rules) {
		a.mu.Unlock()
		return
	}
	a.config.Profiles[a.activeProfileIndex].Rules[idx].Enabled = enabled
	err := a.saveConfigLocked()
	a.rebuildRulesWithoutJoyConRescanLocked()
	a.postUIRefreshLocked()
	a.mu.Unlock()
	if err != nil {
		messageBox("保存失敗", err.Error())
	} else {
		a.refreshRuleList(idx)
		a.updateEditorFromSelection()
	}
}
func (a *App) exportConfig() (string, error) {
	dir := filepath.Join(filepath.Dir(a.configPath), "exports")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	dst := filepath.Join(dir, "config_export_"+time.Now().Format("20060102_150405")+".json")
	if err := copyFile(a.configPath, dst); err != nil {
		return "", err
	}
	_ = exec.Command("explorer.exe", "/select,", dst).Start()
	return dst, nil
}
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
func (a *App) openConfig() { _ = exec.Command("notepad.exe", a.configPath).Start() }

// ---- Browser-based settings UI (v7.7.0+) ----------------------------------
// Win32のListViewでDataGridView相当を無理に再現すると、チェックボックス・列幅・DPIで壊れやすい。
// そこで入力変換コアはそのまま、設定画面だけを localhost のHTML UIへ分離する。
// 表のチェック欄は本物の <input type="checkbox"> で、記録ボタンも旧GUIの導線に寄せる。

type webRule struct {
	Index                   int    `json:"index"`
	Enabled                 bool   `json:"enabled"`
	Input                   string `json:"input"`
	Mode                    string `json:"mode"`
	Output                  string `json:"output"`
	LongPressEnabled        bool   `json:"longPressEnabled"`
	LongPressMs             int    `json:"longPressMs"`
	LongPressAction         string `json:"longPressAction"`
	LongPressOutput         string `json:"longPressOutput"`
	LongPressSummary        string `json:"longPressSummary"`
	SuppressTrigger         bool   `json:"suppressTrigger"`
	SuppressPrefix          bool   `json:"suppressPrefix"`
	SuppressTriggerEditable bool   `json:"suppressTriggerEditable"`
	SuppressPrefixEditable  bool   `json:"suppressPrefixEditable"`
}

type webProfile struct {
	Index     int    `json:"index"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Active    bool   `json:"active"`
	Effective bool   `json:"effective"`
}

type webForegroundApp struct {
	PID         uint32 `json:"pid"`
	ProcessName string `json:"processName"`
	Path        string `json:"path"`
	Title       string `json:"title"`
	Source      string `json:"source"`
	SeenAt      string `json:"seenAt"`
}

type webAutoBinding struct {
	Index               int    `json:"index"`
	ID                  string `json:"id"`
	Enabled             bool   `json:"enabled"`
	Name                string `json:"name"`
	ProfileID           string `json:"profileId"`
	ProfileName         string `json:"profileName"`
	ProcessName         string `json:"processName"`
	TitleContains       string `json:"titleContains"`
	PathContains        string `json:"pathContains"`
	Matched             bool   `json:"matched"`
	MatchesLastExternal bool   `json:"matchesLastExternal"`
	MatchSummary        string `json:"matchSummary"`
}

type webState struct {
	Version              string           `json:"version"`
	ConfigPath           string           `json:"configPath"`
	LogPath              string           `json:"logPath"`
	UIURL                string           `json:"uiUrl"`
	Enabled              bool             `json:"enabled"`
	Emergency            bool             `json:"emergency"`
	Status               string           `json:"status"`
	ProfileName          string           `json:"profileName"`
	BaseProfileName      string           `json:"baseProfileName"`
	EditorProfileName    string           `json:"editorProfileName"`
	ActiveProfile        int              `json:"activeProfile"`
	BaseProfile          int              `json:"baseProfile"`
	EffectiveProfile     int              `json:"effectiveProfile"`
	Profiles             []webProfile     `json:"profiles"`
	Rules                []webRule        `json:"rules"`
	AutoSwitchEnabled    bool             `json:"autoSwitchEnabled"`
	ControllerEnabled    bool             `json:"controllerEnabled"`
	AutoDebounceMs       int              `json:"autoDebounceMs"`
	AutoBindings         []webAutoBinding `json:"autoBindings"`
	AutoMatchedBindingID string           `json:"autoMatchedBindingId"`
	AutoMatchedName      string           `json:"autoMatchedName"`
	AutoMonitorStatus    string           `json:"autoMonitorStatus"`
	AutoDecision         string           `json:"autoDecision"`
	AutoDecisionDetail   string           `json:"autoDecisionDetail"`
	AutoDecisionAt       string           `json:"autoDecisionAt"`
	AutoLastSwitchAt     string           `json:"autoLastSwitchAt"`
	ForegroundApp        webForegroundApp `json:"foregroundApp"`
	LastExternalApp      webForegroundApp `json:"lastExternalApp"`
	RecordingMode        string           `json:"recordingMode"`
	RecordedText         string           `json:"recordedText"`
	LastInput            string           `json:"lastInput"`
	LastInputAt          string           `json:"lastInputAt"`
	HookStatus           string           `json:"hookStatus"`
}

func (a *App) startWebServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.webIndex)
	mux.HandleFunc("/api/state", a.webAPIState)
	mux.HandleFunc("/api/action", a.webAPIAction)
	mux.HandleFunc("/api/rule", a.webAPIRule)
	mux.HandleFunc("/api/profile", a.webAPIProfile)
	mux.HandleFunc("/api/autoswitch", a.webAPIAutoSwitch)
	mux.HandleFunc("/api/controller", a.webAPIControllerFeature)
	mux.HandleFunc("/api/joycon", a.webAPIJoyCon)
	mux.HandleFunc("/joycon-ui.js", a.webJoyConUIJS)
	mux.HandleFunc("/api/import-json", a.webAPIImportJSON)
	mux.HandleFunc("/api/default-rules", a.webAPIDefaultRules)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	a.httpAddr = ln.Addr().String()
	go func() {
		err := http.Serve(ln, mux)
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "closed") {
			a.logf("web ui server stopped: %v", err)
		}
	}()
	a.logf("web ui: http://%s/", a.httpAddr)
	return nil
}

func (a *App) webURL() string {
	if a.httpAddr == "" {
		return ""
	}
	return "http://" + a.httpAddr + "/"
}

func openEdgeAppOrBrowser(url string) {
	if url == "" {
		return
	}
	// フォールバックでも通常タブではなく、まずEdgeのアプリウィンドウを試す。
	if err := exec.Command("msedge.exe", "--app="+url).Start(); err == nil {
		return
	}
	if err := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url).Start(); err != nil {
		_ = exec.Command("cmd.exe", "/C", "start", "", url).Start()
	}
}

type webView2EnvCompletedHandler struct {
	lpVtbl *webView2EnvCompletedHandlerVtbl
	ref    uint32
}
type webView2EnvCompletedHandlerVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Invoke         uintptr
}
type webView2ControllerCompletedHandler struct {
	lpVtbl *webView2ControllerCompletedHandlerVtbl
	ref    uint32
}
type webView2ControllerCompletedHandlerVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Invoke         uintptr
}

var (
	webView2EnvCompletedVtbl = &webView2EnvCompletedHandlerVtbl{
		QueryInterface: syscall.NewCallback(comQueryInterface),
		AddRef:         syscall.NewCallback(comAddRef),
		Release:        syscall.NewCallback(comRelease),
		Invoke:         syscall.NewCallback(webView2EnvCompletedInvoke),
	}
	webView2ControllerCompletedVtbl = &webView2ControllerCompletedHandlerVtbl{
		QueryInterface: syscall.NewCallback(comQueryInterface),
		AddRef:         syscall.NewCallback(comAddRef),
		Release:        syscall.NewCallback(comRelease),
		Invoke:         syscall.NewCallback(webView2ControllerCompletedInvoke),
	}
)

func comQueryInterface(this uintptr, riid uintptr, ppv uintptr) uintptr {
	if ppv != 0 {
		*(*uintptr)(unsafe.Pointer(ppv)) = this
	}
	return 0
}
func comAddRef(this uintptr) uintptr  { return 1 }
func comRelease(this uintptr) uintptr { return 1 }

func comVtbl(obj uintptr) *[128]uintptr {
	return (*[128]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(obj))))
}
func hresultFailed(hr uintptr) bool { return int32(hr) < 0 }

func (a *App) showWebView2(url string) bool {
	a.mu.Lock()
	if a.webviewReady {
		a.webviewURL = url
		core := a.webviewCore
		a.mu.Unlock()
		a.navigateWebView2(core, url)
		a.resizeWebView2()
		return true
	}
	if a.webviewLoading {
		a.webviewURL = url
		a.mu.Unlock()
		return true
	}
	a.webviewLoading = true
	a.webviewURL = url
	a.webviewEnvHandler = &webView2EnvCompletedHandler{lpVtbl: webView2EnvCompletedVtbl, ref: 1}
	a.webviewCtlHandler = &webView2ControllerCompletedHandler{lpVtbl: webView2ControllerCompletedVtbl, ref: 1}
	a.mu.Unlock()

	createProc := loadWebView2CreateProc()
	if createProc == 0 {
		a.logf("WebView2Loader.dll not found; falling back to Edge app window")
		a.mu.Lock()
		a.webviewLoading = false
		a.mu.Unlock()
		return false
	}
	userDataDir := filepath.Join(filepath.Dir(a.configPath), "WebView2UserData")
	_ = os.MkdirAll(userDataDir, 0755)
	ud := syscall.StringToUTF16Ptr(userDataDir)
	hr, _, _ := syscall.SyscallN(createProc, 0, uintptr(unsafe.Pointer(ud)), 0, uintptr(unsafe.Pointer(a.webviewEnvHandler)))
	if hresultFailed(hr) {
		a.logf("CreateCoreWebView2EnvironmentWithOptions failed: hr=0x%x", hr)
		a.mu.Lock()
		a.webviewLoading = false
		a.mu.Unlock()
		return false
	}
	return true
}

func loadWebView2CreateProc() uintptr {
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "WebView2Loader.dll"))
	}
	for _, cand := range candidates {
		name := syscall.StringToUTF16Ptr(cand)
		h, _, _ := pLoadLibraryW.Call(uintptr(unsafe.Pointer(name)))
		if h == 0 {
			continue
		}
		procName := append([]byte("CreateCoreWebView2EnvironmentWithOptions"), 0)
		p, _, _ := pGetProcAddress.Call(h, uintptr(unsafe.Pointer(&procName[0])))
		if p != 0 {
			app.logf("loaded WebView2Loader: %s", cand)
			return p
		}
	}
	return 0
}

func webView2EnvCompletedInvoke(this uintptr, result uintptr, env uintptr) uintptr {
	if hresultFailed(result) || env == 0 {
		app.logf("WebView2 environment callback failed: hr=0x%x env=%x", result, env)
		app.mu.Lock()
		app.webviewLoading = false
		app.mu.Unlock()
		openEdgeAppOrBrowser(app.webURL())
		return 0
	}
	vt := comVtbl(env)
	// ICoreWebView2Environment::CreateCoreWebView2Controller(HWND, handler) はIUnknownの直後、vtable index 3。
	hr, _, _ := syscall.SyscallN(vt[3], env, app.hwnd, uintptr(unsafe.Pointer(app.webviewCtlHandler)))
	if hresultFailed(hr) {
		app.logf("CreateCoreWebView2Controller failed: hr=0x%x", hr)
		app.mu.Lock()
		app.webviewLoading = false
		app.mu.Unlock()
		openEdgeAppOrBrowser(app.webURL())
	}
	return 0
}

func webView2ControllerCompletedInvoke(this uintptr, result uintptr, controller uintptr) uintptr {
	if hresultFailed(result) || controller == 0 {
		app.logf("WebView2 controller callback failed: hr=0x%x controller=%x", result, controller)
		app.mu.Lock()
		app.webviewLoading = false
		app.mu.Unlock()
		openEdgeAppOrBrowser(app.webURL())
		return 0
	}
	app.mu.Lock()
	app.webviewController = controller
	app.webviewReady = true
	app.webviewLoading = false
	url := app.webviewURL
	app.mu.Unlock()

	vt := comVtbl(controller)
	// ICoreWebView2Controller::put_IsVisible は vtable index 4。
	syscall.SyscallN(vt[4], controller, uintptr(1))
	app.resizeWebView2()
	var core uintptr
	// ICoreWebView2Controller::get_CoreWebView2 は vtable index 25。
	hr, _, _ := syscall.SyscallN(vt[25], controller, uintptr(unsafe.Pointer(&core)))
	if !hresultFailed(hr) && core != 0 {
		app.mu.Lock()
		app.webviewCore = core
		app.mu.Unlock()
		app.navigateWebView2(core, url)
	} else {
		app.logf("CoreWebView2 get_CoreWebView2 failed: hr=0x%x core=%x; falling back to Edge app window", hr, core)
		openEdgeAppOrBrowser(app.webURL())
	}
	return 0
}

func (a *App) navigateWebView2(core uintptr, url string) {
	if core == 0 || url == "" {
		return
	}
	vt := comVtbl(core)
	u := syscall.StringToUTF16Ptr(url)
	// ICoreWebView2::Navigate は vtable index 5。
	hr, _, _ := syscall.SyscallN(vt[5], core, uintptr(unsafe.Pointer(u)))
	if hresultFailed(hr) {
		a.logf("CoreWebView2.Navigate failed: hr=0x%x", hr)
	}
}

func (a *App) resizeWebView2() {
	a.mu.RLock()
	controller := a.webviewController
	a.mu.RUnlock()
	if controller == 0 || a.hwnd == 0 {
		return
	}
	var rc RECT
	pGetClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&rc)))
	vt := comVtbl(controller)
	// ICoreWebView2Controller::put_Bounds は vtable index 6。
	syscall.SyscallN(vt[6], controller, uintptr(unsafe.Pointer(&rc)))
}

func (a *App) closeWebView2() {
	a.mu.RLock()
	controller := a.webviewController
	a.mu.RUnlock()
	if controller == 0 {
		return
	}
	vt := comVtbl(controller)
	// ICoreWebView2Controller::Close は vtable index 24。
	syscall.SyscallN(vt[24], controller)
}

func webForeground(info ForegroundAppInfo) webForegroundApp {
	seen := ""
	if !info.SeenAt.IsZero() {
		seen = info.SeenAt.Format("15:04:05")
	}
	return webForegroundApp{PID: info.PID, ProcessName: info.ProcessName, Path: info.Path, Title: info.Title, Source: info.Source, SeenAt: seen}
}

func (a *App) buildWebState() webState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	status := "実行中"
	if a.emergency {
		status = "緊急停止中"
	} else if !a.enabled {
		status = "停止中"
	}
	profiles := []webProfile{}
	for i, p := range a.config.Profiles {
		profiles = append(profiles, webProfile{Index: i, ID: p.Id, Name: p.Name, Active: i == a.editorProfileIndex, Effective: i == a.activeProfileIndex})
	}
	rules := []webRule{}
	if len(a.config.Profiles) > 0 && a.editorProfileIndex >= 0 && a.editorProfileIndex < len(a.config.Profiles) {
		for i, r := range a.config.Profiles[a.editorProfileIndex].Rules {
			rules = append(rules, webRule{
				Index:                   i,
				Enabled:                 r.Enabled,
				Input:                   itemsText(r.Input),
				Mode:                    normalizeJoyConRuleMode(r.Mode),
				Output:                  itemsText(r.Output),
				LongPressEnabled:        r.LongPressEnabled,
				LongPressMs:             normalizeLongPressMs(r.LongPressMs),
				LongPressAction:         normalizeLongPressAction(r.LongPressAction),
				LongPressOutput:         itemsText(r.LongPressOutput),
				LongPressSummary:        ruleLongPressSummary(r),
				SuppressTrigger:         r.SuppressTrigger,
				SuppressPrefix:          r.SuppressPrefix,
				SuppressTriggerEditable: !isLastInputPrimaryMouse(r.Input),
				SuppressPrefixEditable:  hasSidePrefix(r.Input),
			})
		}
	}
	autoBindings := make([]webAutoBinding, 0, len(a.config.AutoSwitch.Bindings))
	for i, b := range a.config.AutoSwitch.Bindings {
		profileName := "（削除済み）"
		if idx := a.profileIndexByIDLocked(b.ProfileId); idx >= 0 {
			profileName = a.config.Profiles[idx].Name
		}
		matchResult := evaluateBindingMatch(b, a.lastExternalApp)
		matchSummary := strings.Join(matchResult.Reasons, " / ")
		autoBindings = append(autoBindings, webAutoBinding{
			Index: i, ID: b.Id, Enabled: b.Enabled, Name: b.Name,
			ProfileID: b.ProfileId, ProfileName: profileName,
			ProcessName: b.ProcessName, TitleContains: b.TitleContains, PathContains: b.PathContains,
			Matched:             b.Id != "" && b.Id == a.autoBindingID,
			MatchesLastExternal: matchResult.Matches,
			MatchSummary:        matchSummary,
		})
	}
	lastAt := ""
	if !a.lastInputAt.IsZero() {
		lastAt = a.lastInputAt.Format("15:04:05")
	}
	last := a.lastInputText
	if last == "" {
		last = "（まだありません）"
	}
	recorded := itemsText(a.recordedItems)
	hook := a.hookHealthText()
	autoDecisionAt := ""
	if !a.autoDecisionAt.IsZero() {
		autoDecisionAt = a.autoDecisionAt.Format("15:04:05")
	}
	autoLastSwitchAt := ""
	if !a.autoLastSwitchAt.IsZero() {
		autoLastSwitchAt = a.autoLastSwitchAt.Format("15:04:05")
	}
	return webState{
		Version:              appVersion,
		ConfigPath:           a.configPath,
		LogPath:              a.logPath,
		UIURL:                a.webURL(),
		Enabled:              a.enabled,
		Emergency:            a.emergency,
		Status:               status,
		ProfileName:          a.activeProfileNameLocked(),
		BaseProfileName:      a.baseProfileNameLocked(),
		EditorProfileName:    a.editorProfileNameLocked(),
		ActiveProfile:        a.editorProfileIndex,
		BaseProfile:          a.profileIndexByIDLocked(a.config.ActiveProfileId),
		EffectiveProfile:     a.activeProfileIndex,
		Profiles:             profiles,
		Rules:                rules,
		AutoSwitchEnabled:    a.config.AutoSwitch.Enabled,
		ControllerEnabled:    a.config.Controller.Enabled,
		AutoDebounceMs:       a.config.AutoSwitch.DebounceMs,
		AutoBindings:         autoBindings,
		AutoMatchedBindingID: a.autoBindingID,
		AutoMatchedName:      a.autoBindingName,
		AutoMonitorStatus:    a.foregroundMonitorStatus,
		AutoDecision:         a.autoDecision,
		AutoDecisionDetail:   a.autoDecisionDetail,
		AutoDecisionAt:       autoDecisionAt,
		AutoLastSwitchAt:     autoLastSwitchAt,
		ForegroundApp:        webForeground(a.foregroundApp),
		LastExternalApp:      webForeground(a.lastExternalApp),
		RecordingMode:        a.recordingMode,
		RecordedText:         recorded,
		LastInput:            last,
		LastInputAt:          lastAt,
		HookStatus:           hook,
	}
}

func (a *App) webIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(webHTML))
}

func (a *App) webAPIState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.buildWebState())
}

type actionReq struct {
	Action     string `json:"action"`
	Index      int    `json:"index"`
	Mode       string `json:"mode"`
	Output     string `json:"output"`
	LongOutput string `json:"longOutput"`
}

func (a *App) webAPIAction(w http.ResponseWriter, r *http.Request) {
	var req actionReq
	if !decodeJSON(w, r, &req) {
		return
	}
	sendOK := func(msg string) { writeJSON(w, map[string]any{"ok": true, "message": msg, "state": a.buildWebState()}) }
	switch req.Action {
	case "toggle-running":
		a.mu.Lock()
		if a.enabled && !a.emergency {
			a.enabled = false
			a.abortAllLongPressLocked("conversion stopped", false)
			a.clearJoyConInputStateLocked("conversion stopped")
		} else {
			a.enabled = true
			a.emergency = false
		}
		a.mu.Unlock()
		a.releaseJoyConHeldOutputs()
		sendOK("変換状態を切り替えました。")
	case "emergency":
		a.mu.Lock()
		a.enabled = false
		a.emergency = true
		a.abortAllLongPressLocked("emergency stop from web UI", false)
		a.clearJoyConInputStateLocked("emergency stop from web UI")
		a.recordingMode = ""
		a.recordHeld = map[string]bool{}
		a.mu.Unlock()
		a.ReleaseModifiersNow()
		sendOK("緊急停止しました。")
	case "release":
		a.ReleaseModifiersNow()
		sendOK("修飾キー解放を送信しました。")
	case "reload":
		if err := a.loadConfig(); err != nil {
			writeError(w, err)
			return
		}
		sendOK("設定を再読み込みしました。")
	case "open-config":
		a.openConfig()
		sendOK("config.jsonを開きました。")
	case "open-folder":
		_ = exec.Command("explorer.exe", filepath.Dir(a.configPath)).Start()
		sendOK("設定フォルダーを開きました。")
	case "open-log":
		_ = exec.Command("notepad.exe", a.logPath).Start()
		sendOK("ログを開きました。")
	case "export":
		p, err := a.exportConfig()
		if err != nil {
			writeError(w, err)
			return
		}
		sendOK("エクスポートしました: " + p)
	case "record-input", "record-output", "record-long-output":
		mode := "input"
		if req.Action == "record-output" {
			mode = "output"
		} else if req.Action == "record-long-output" {
			mode = "long-output"
		}
		if err := a.startRecordingAt(mode, req.Index); err != nil {
			writeError(w, err)
			return
		}
		sendOK("記録を開始しました。")
	case "record-stop":
		a.mu.Lock()
		a.recordingMode = ""
		a.recordingRuleIndex = -1
		a.recordingProfileID = ""
		a.recordedItems = nil
		a.recordHeld = map[string]bool{}
		a.mu.Unlock()
		sendOK("記録を中止しました。")
	case "test-output", "test-long-output":
		text := req.Output
		if req.Action == "test-long-output" {
			text = req.LongOutput
		}
		items, err := parseItemsText(text, true, true)
		if err != nil {
			writeError(w, err)
			return
		}
		if len(items) == 0 {
			writeError(w, fmt.Errorf("テストする実行内容を入力してください。"))
			return
		}
		if err := validateExecutableOutputItems(items); err != nil {
			writeError(w, err)
			return
		}
		a.enqueueRuleGuaranteed(Rule{Enabled: true, Mode: "Tap", Output: items})
		sendOK("実行内容をテストしました。")
	case "quit":
		go func() {
			time.Sleep(150 * time.Millisecond)
			if a.hwnd != 0 {
				pDestroyWindow.Call(a.hwnd)
			}
		}()
		sendOK("終了します。")
	default:
		writeError(w, fmt.Errorf("未知の操作です: %s", req.Action))
	}
}

func (a *App) startRecordingAt(mode string, idx int) error {
	if mode != "input" && mode != "output" && mode != "long-output" {
		return fmt.Errorf("記録種別が不正です。")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	rules := a.editorRulesSliceLocked()
	if rules == nil || idx < 0 || idx >= len(*rules) {
		return fmt.Errorf("先に記録先のルールを選択してください。")
	}
	a.recordingMode = mode
	a.recordingRuleIndex = idx
	if a.editorProfileIndex >= 0 && a.editorProfileIndex < len(a.config.Profiles) {
		a.recordingProfileID = a.config.Profiles[a.editorProfileIndex].Id
	}
	a.recordedItems = nil
	a.recordHeld = map[string]bool{}
	a.lastInputText = ""
	a.lastInputAt = time.Time{}
	return nil
}

type ruleReq struct {
	Op               string `json:"op"`
	Index            int    `json:"index"`
	Target           int    `json:"target"`
	Delta            int    `json:"delta"`
	Field            string `json:"field"`
	Enabled          bool   `json:"enabled"`
	Input            string `json:"input"`
	Mode             string `json:"mode"`
	Output           string `json:"output"`
	SuppressTrigger  bool   `json:"suppressTrigger"`
	SuppressPrefix   bool   `json:"suppressPrefix"`
	LongPressEnabled bool   `json:"longPressEnabled"`
	LongPressMs      int    `json:"longPressMs"`
	LongPressAction  string `json:"longPressAction"`
	LongPressOutput  string `json:"longPressOutput"`
}

func (a *App) webAPIRule(w http.ResponseWriter, r *http.Request) {
	var req ruleReq
	if !decodeJSON(w, r, &req) {
		return
	}
	var msg string
	var err error
	switch req.Op {
	case "toggle":
		err = a.webToggleRuleCell(req.Index, req.Field)
		msg = "チェックを切り替えました。"
	case "save":
		err = a.webSaveRule(req)
		msg = "選択中の割り当てを保存しました。現在適用中のプロファイルなら、動作にも即時反映されています。"
	case "add":
		err = a.webAddRule()
		msg = "ルールを追加しました。"
	case "duplicate":
		err = a.webDuplicateRule(req.Index)
		msg = "ルールを複製しました。"
	case "delete":
		err = a.webDeleteRule(req.Index)
		msg = "ルールを削除しました。"
	case "move":
		err = a.webMoveRule(req.Index, req.Target, req.Delta)
		msg = "ルールを移動しました。"
	default:
		err = fmt.Errorf("未知のルール操作です: %s", req.Op)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "message": msg, "state": a.buildWebState()})
}

func (a *App) webToggleRuleCell(idx int, field string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	rules := a.editorRulesSliceLocked()
	if rules == nil || idx < 0 || idx >= len(*rules) {
		return fmt.Errorf("対象ルールがありません。")
	}
	r := &(*rules)[idx]
	switch field {
	case "enabled":
		r.Enabled = !r.Enabled
	case "suppressTrigger":
		if isLastInputPrimaryMouse(r.Input) {
			r.SuppressTrigger = false
		} else {
			r.SuppressTrigger = !r.SuppressTrigger
		}
	case "suppressPrefix":
		if hasSidePrefix(r.Input) {
			r.SuppressPrefix = !r.SuppressPrefix
		} else {
			r.SuppressPrefix = false
		}
	default:
		return fmt.Errorf("未知のチェック欄です: %s", field)
	}
	if err := a.saveConfigLocked(); err != nil {
		return err
	}
	a.rebuildRulesWithoutJoyConRescanLocked()
	return nil
}

func (a *App) webSaveRule(req ruleReq) error {
	input, err := parseItemsText(req.Input, true, true)
	if err != nil {
		return fmt.Errorf("入力の解釈に失敗: %w", err)
	}
	output, err := parseItemsText(req.Output, true, true)
	if err != nil {
		return fmt.Errorf("短押し時の実行内容の解釈に失敗: %w", err)
	}
	if err := validateExecutableOutputItems(output); err != nil {
		return fmt.Errorf("短押し時の実行内容が不正です: %w", err)
	}
	if len(input) == 0 {
		return fmt.Errorf("入力は1つ以上必要です。")
	}

	action := normalizeLongPressAction(req.LongPressAction)
	longOutput := []Item(nil)
	if req.LongPressEnabled && action == longPressActionExecute {
		longOutput, err = parseItemsText(req.LongPressOutput, true, true)
		if err != nil {
			return fmt.Errorf("長押し時の実行内容の解釈に失敗: %w", err)
		}
		if err := validateExecutableOutputItems(longOutput); err != nil {
			return fmt.Errorf("長押し時の実行内容が不正です: %w", err)
		}
	}
	if !req.LongPressEnabled && len(output) == 0 {
		return fmt.Errorf("長押し判定を使わない場合は、実行内容が1つ以上必要です。")
	}

	r := Rule{
		Enabled:          req.Enabled,
		Input:            input,
		Mode:             normalizeJoyConRuleMode(req.Mode),
		Output:           output,
		SuppressTrigger:  req.SuppressTrigger,
		SuppressPrefix:   req.SuppressPrefix,
		LongPressEnabled: req.LongPressEnabled,
		LongPressMs:      normalizeLongPressMs(req.LongPressMs),
		LongPressAction:  action,
		LongPressOutput:  longOutput,
	}
	if isLastInputPrimaryMouse(r.Input) {
		r.SuppressTrigger = false
	}
	if !hasSidePrefix(r.Input) {
		r.SuppressPrefix = false
	}
	if err := validateJoyConHoldRule(r); err != nil {
		return err
	}
	if err := validateLongPressRule(r); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	rules := a.editorRulesSliceLocked()
	if rules == nil || req.Index < 0 || req.Index >= len(*rules) {
		return fmt.Errorf("対象ルールがありません。")
	}
	(*rules)[req.Index] = r
	if err := a.saveConfigLocked(); err != nil {
		return err
	}
	a.rebuildRulesWithoutJoyConRescanLocked()
	return nil
}

func (a *App) webAddRule() error {
	r := Rule{Enabled: true, Input: []Item{{Kind: "Mouse", Code: "X1"}}, Mode: "Tap", Output: []Item{{Kind: "Key", Code: strconv.Itoa(int(VK_CONTROL))}, {Kind: "Key", Code: "86"}}, SuppressTrigger: true}
	a.mu.Lock()
	defer a.mu.Unlock()
	rules := a.editorRulesSliceLocked()
	if rules == nil {
		return fmt.Errorf("アクティブなプロファイルがありません。")
	}
	*rules = append(*rules, r)
	if err := a.saveConfigLocked(); err != nil {
		return err
	}
	a.rebuildRulesWithoutJoyConRescanLocked()
	return nil
}

func (a *App) webDuplicateRule(idx int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	rules := a.editorRulesSliceLocked()
	if rules == nil || idx < 0 || idx >= len(*rules) {
		return fmt.Errorf("対象ルールがありません。")
	}
	r := (*rules)[idx]
	insert := idx + 1
	*rules = append((*rules)[:insert], append([]Rule{r}, (*rules)[insert:]...)...)
	if err := a.saveConfigLocked(); err != nil {
		return err
	}
	a.rebuildRulesWithoutJoyConRescanLocked()
	return nil
}

func (a *App) webDeleteRule(idx int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	rules := a.editorRulesSliceLocked()
	if rules == nil || idx < 0 || idx >= len(*rules) {
		return fmt.Errorf("対象ルールがありません。")
	}
	*rules = append((*rules)[:idx], (*rules)[idx+1:]...)
	if err := a.saveConfigLocked(); err != nil {
		return err
	}
	a.rebuildRulesWithoutJoyConRescanLocked()
	return nil
}

func (a *App) webMoveRule(idx, target, delta int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	rules := a.editorRulesSliceLocked()
	if rules == nil || idx < 0 || idx >= len(*rules) {
		return fmt.Errorf("対象ルールがありません。")
	}
	if target < 0 {
		target = idx + delta
	}
	if target < 0 {
		target = 0
	}
	if target >= len(*rules) {
		target = len(*rules) - 1
	}
	if target == idx {
		return nil
	}
	r := (*rules)[idx]
	without := append((*rules)[:idx], (*rules)[idx+1:]...)
	if target > len(without) {
		target = len(without)
	}
	*rules = append(without[:target], append([]Rule{r}, without[target:]...)...)
	if err := a.saveConfigLocked(); err != nil {
		return err
	}
	a.rebuildRulesWithoutJoyConRescanLocked()
	return nil
}

type profileReq struct {
	Op    string `json:"op"`
	Index int    `json:"index"`
	Name  string `json:"name"`
}

func (a *App) webAPIProfile(w http.ResponseWriter, r *http.Request) {
	var req profileReq
	if !decodeJSON(w, r, &req) {
		return
	}
	var err error
	switch req.Op {
	case "switch": // v8.1互換: 編集対象と通常時プロファイルを同時に変更
		err = a.webSwitchProfile(req.Index)
	case "edit":
		err = a.webEditProfile(req.Index)
	case "set-base":
		err = a.webSetBaseProfile(req.Index)
	case "new":
		err = a.webNewProfile(req.Name)
	case "duplicate":
		err = a.webDuplicateProfile(req.Index, req.Name)
	case "rename":
		err = a.webRenameProfile(req.Index, req.Name)
	case "delete":
		err = a.webDeleteProfile(req.Index)
	default:
		err = fmt.Errorf("未知のプロファイル操作です: %s", req.Op)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "state": a.buildWebState()})
}

func (a *App) webEditProfile(idx int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if idx < 0 || idx >= len(a.config.Profiles) {
		return fmt.Errorf("プロファイルがありません。")
	}
	a.editorProfileIndex = idx
	a.postUIRefreshLocked()
	return nil
}

func (a *App) webSetBaseProfile(idx int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if idx < 0 || idx >= len(a.config.Profiles) {
		return fmt.Errorf("プロファイルがありません。")
	}
	a.config.ActiveProfileId = a.config.Profiles[idx].Id
	if err := a.saveConfigLocked(); err != nil {
		return err
	}
	a.rebuildRulesLocked()
	a.postUIRefreshLocked()
	return nil
}

func (a *App) webSwitchProfile(idx int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if idx < 0 || idx >= len(a.config.Profiles) {
		return fmt.Errorf("プロファイルがありません。")
	}
	a.editorProfileIndex = idx
	a.config.ActiveProfileId = a.config.Profiles[idx].Id
	if err := a.saveConfigLocked(); err != nil {
		return err
	}
	a.rebuildRulesLocked()
	return nil
}

func (a *App) webNewProfile(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "新規プロファイル"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	id := fmt.Sprintf("profile_%d", time.Now().UnixNano())
	a.config.Profiles = append(a.config.Profiles, Profile{Id: id, Name: name, Rules: []Rule{}})
	a.editorProfileIndex = len(a.config.Profiles) - 1
	if err := a.saveConfigLocked(); err != nil {
		return err
	}
	a.rebuildRulesLocked()
	return nil
}

func (a *App) webDuplicateProfile(idx int, name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if idx < 0 || idx >= len(a.config.Profiles) {
		return fmt.Errorf("プロファイルがありません。")
	}
	p := a.config.Profiles[idx]
	if strings.TrimSpace(name) == "" {
		name = p.Name + " のコピー"
	}
	p.Id = fmt.Sprintf("profile_%d", time.Now().UnixNano())
	p.Name = name
	clonedRules := make([]Rule, len(p.Rules))
	for i := range p.Rules {
		clonedRules[i] = cloneRule(p.Rules[i])
	}
	p.Rules = clonedRules
	a.config.Profiles = append(a.config.Profiles, p)
	a.editorProfileIndex = len(a.config.Profiles) - 1
	if err := a.saveConfigLocked(); err != nil {
		return err
	}
	a.rebuildRulesLocked()
	return nil
}

func (a *App) webRenameProfile(idx int, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("名前が空です。")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if idx < 0 || idx >= len(a.config.Profiles) {
		return fmt.Errorf("プロファイルがありません。")
	}
	a.config.Profiles[idx].Name = name
	if err := a.saveConfigLocked(); err != nil {
		return err
	}
	a.rebuildRulesLocked()
	return nil
}

func (a *App) webDeleteProfile(idx int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.config.Profiles) <= 1 {
		return fmt.Errorf("最後のプロファイルは削除できません。")
	}
	if idx < 0 || idx >= len(a.config.Profiles) {
		return fmt.Errorf("プロファイルがありません。")
	}
	id := a.config.Profiles[idx].Id
	for _, b := range a.config.AutoSwitch.Bindings {
		if b.ProfileId == id {
			return fmt.Errorf("このプロファイルはアプリ自動切替の「%s」で使用中です。先に自動切替設定を削除または変更してください。", b.Name)
		}
	}
	deletedBase := a.config.ActiveProfileId == id
	a.config.Profiles = append(a.config.Profiles[:idx], a.config.Profiles[idx+1:]...)
	if a.editorProfileIndex >= len(a.config.Profiles) {
		a.editorProfileIndex = len(a.config.Profiles) - 1
	} else if idx < a.editorProfileIndex {
		a.editorProfileIndex--
	}
	if a.editorProfileIndex < 0 {
		a.editorProfileIndex = 0
	}
	if deletedBase || a.profileIndexByIDLocked(a.config.ActiveProfileId) < 0 {
		a.config.ActiveProfileId = a.config.Profiles[0].Id
	}
	if a.autoProfileID == id {
		a.clearAutoMatchLocked()
	}
	if err := a.saveConfigLocked(); err != nil {
		return err
	}
	a.rebuildRulesLocked()
	return nil
}

type autoSwitchReq struct {
	Op            string `json:"op"`
	Index         int    `json:"index"`
	Target        int    `json:"target"`
	Delta         int    `json:"delta"`
	Enabled       bool   `json:"enabled"`
	DebounceMs    int    `json:"debounceMs"`
	Name          string `json:"name"`
	ProfileID     string `json:"profileId"`
	ProcessName   string `json:"processName"`
	TitleContains string `json:"titleContains"`
	PathContains  string `json:"pathContains"`
}

func (a *App) webAPIAutoSwitch(w http.ResponseWriter, r *http.Request) {
	var req autoSwitchReq
	if !decodeJSON(w, r, &req) {
		return
	}
	var err error
	message := "自動切替設定を保存しました。"
	switch req.Op {
	case "settings":
		err = a.webSetAutoSwitch(req.Enabled, req.DebounceMs)
	case "add-captured":
		err = a.webAddCapturedBinding(req.ProfileID)
		message = "直前に使用していたアプリを登録しました。"
	case "add-empty":
		err = a.webAddEmptyBinding(req.ProfileID)
		message = "空の自動切替設定を追加しました。"
	case "save":
		err = a.webSaveBinding(req)
		message = "選択中の自動切替ルールを保存しました。"
	case "toggle":
		err = a.webToggleBinding(req.Index)
		message = "自動切替ルールの有効状態を変更しました。"
	case "duplicate":
		err = a.webDuplicateBinding(req.Index)
		message = "自動切替設定を複製しました。"
	case "delete":
		err = a.webDeleteBinding(req.Index)
		message = "自動切替設定を削除しました。"
	case "move":
		err = a.webMoveBinding(req.Index, req.Target, req.Delta)
		message = "自動切替の優先順位を変更しました。"
	case "recheck":
		a.evaluateBestKnownForeground()
		message = "直前のアプリを使って自動切替を再判定しました。"
	default:
		err = fmt.Errorf("未知の自動切替操作です: %s", req.Op)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	if req.Op != "recheck" && req.Op != "settings" {
		a.evaluateBestKnownForeground()
	}
	writeJSON(w, map[string]any{"ok": true, "message": message, "state": a.buildWebState()})
}

func clampDebounce(ms int) int {
	if ms < 50 {
		return 50
	}
	if ms > 3000 {
		return 3000
	}
	return ms
}

func (a *App) webSetAutoSwitch(enabled bool, debounceMs int) error {
	a.mu.Lock()
	a.config.AutoSwitch.Enabled = enabled
	a.config.AutoSwitch.DebounceMs = clampDebounce(debounceMs)
	if !enabled {
		a.clearAutoMatchLocked()
	}
	a.autoCandidateKey = ""
	a.autoCandidateSince = time.Time{}
	a.autoDecisionAt = time.Time{}
	a.rebuildRulesLocked()
	err := a.saveConfigLocked()
	a.mu.Unlock()
	if err == nil {
		a.evaluateBestKnownForeground()
	}
	return err
}

func (a *App) resolveBindingProfileLocked(profileID string) (string, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" && a.editorProfileIndex >= 0 && a.editorProfileIndex < len(a.config.Profiles) {
		profileID = a.config.Profiles[a.editorProfileIndex].Id
	}
	if !a.profileExistsLocked(profileID) {
		return "", fmt.Errorf("割り当て先のプロファイルがありません。")
	}
	return profileID, nil
}

func newBindingID() string {
	return fmt.Sprintf("binding_%d", time.Now().UnixNano())
}

func defaultBindingName(info ForegroundAppInfo) string {
	name := strings.TrimSuffix(baseNameAnySeparator(info.ProcessName), ".exe")
	if name == "" {
		name = strings.TrimSpace(info.Title)
	}
	if name == "" {
		name = "新しいアプリ設定"
	}
	return name
}

func (a *App) webAddCapturedBinding(profileID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	profileID, err := a.resolveBindingProfileLocked(profileID)
	if err != nil {
		return err
	}
	info := a.lastExternalApp
	if strings.TrimSpace(info.ProcessName) == "" && strings.TrimSpace(info.Title) == "" {
		return fmt.Errorf("直前に使用していたアプリを取得できませんでした。対象アプリを一度前面に出してから、設定画面へ戻ってください。")
	}
	b := AppBinding{
		Id:          newBindingID(),
		Enabled:     true,
		Name:        defaultBindingName(info),
		ProfileId:   profileID,
		ProcessName: baseNameAnySeparator(info.ProcessName),
	}
	if b.ProcessName == "" {
		b.TitleContains = strings.TrimSpace(info.Title)
	}
	a.config.AutoSwitch.Bindings = append(a.config.AutoSwitch.Bindings, b)
	return a.saveConfigLocked()
}

func (a *App) webAddEmptyBinding(profileID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	profileID, err := a.resolveBindingProfileLocked(profileID)
	if err != nil {
		return err
	}
	a.config.AutoSwitch.Bindings = append(a.config.AutoSwitch.Bindings, AppBinding{
		Id:        newBindingID(),
		Enabled:   true,
		Name:      "新しいアプリ設定",
		ProfileId: profileID,
	})
	return a.saveConfigLocked()
}

func normalizeBindingInput(b AppBinding) AppBinding {
	b.Name = strings.TrimSpace(b.Name)
	b.ProfileId = strings.TrimSpace(b.ProfileId)
	b.ProcessName = baseNameAnySeparator(b.ProcessName)
	b.TitleContains = strings.TrimSpace(b.TitleContains)
	b.PathContains = strings.TrimSpace(b.PathContains)
	if b.Name == "" {
		b.Name = strings.TrimSuffix(b.ProcessName, ".exe")
	}
	if b.Name == "" {
		b.Name = "アプリ設定"
	}
	return b
}

func validateBinding(b AppBinding, profileExists func(string) bool) error {
	if !profileExists(b.ProfileId) {
		return fmt.Errorf("割り当て先のプロファイルがありません。")
	}
	if strings.TrimSpace(b.ProcessName) == "" && strings.TrimSpace(b.TitleContains) == "" && strings.TrimSpace(b.PathContains) == "" {
		return fmt.Errorf("プロセス名・ウィンドウタイトル・実行ファイルパスのうち、少なくとも1つを指定してください。")
	}
	if strings.ContainsAny(b.ProcessName, "\\/") {
		return fmt.Errorf("プロセス名にはファイル名だけを指定してください。例: game.exe")
	}
	if hasGlobMeta(b.ProcessName) {
		if _, err := path.Match(strings.ToLower(b.ProcessName), "example.exe"); err != nil {
			return fmt.Errorf("プロセス名のワイルドカード形式が不正です: %v", err)
		}
	}
	return nil
}

func (a *App) webToggleBinding(index int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if index < 0 || index >= len(a.config.AutoSwitch.Bindings) {
		return fmt.Errorf("対象の自動切替ルールがありません。")
	}
	a.config.AutoSwitch.Bindings[index].Enabled = !a.config.AutoSwitch.Bindings[index].Enabled
	a.autoCandidateKey = ""
	a.autoCandidateSince = time.Time{}
	return a.saveConfigLocked()
}

func (a *App) webSaveBinding(req autoSwitchReq) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if req.Index < 0 || req.Index >= len(a.config.AutoSwitch.Bindings) {
		return fmt.Errorf("対象の自動切替設定がありません。")
	}
	old := a.config.AutoSwitch.Bindings[req.Index]
	b := normalizeBindingInput(AppBinding{
		Id: old.Id, Enabled: req.Enabled, Name: req.Name, ProfileId: req.ProfileID,
		ProcessName: req.ProcessName, TitleContains: req.TitleContains, PathContains: req.PathContains,
	})
	if err := validateBinding(b, a.profileExistsLocked); err != nil {
		return err
	}
	a.config.AutoSwitch.Bindings[req.Index] = b
	if a.autoBindingID == b.Id {
		a.autoProfileID = b.ProfileId
		a.autoBindingName = b.Name
	}
	a.autoCandidateKey = ""
	a.autoCandidateSince = time.Time{}
	a.rebuildRulesLocked()
	return a.saveConfigLocked()
}

func (a *App) webDuplicateBinding(index int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if index < 0 || index >= len(a.config.AutoSwitch.Bindings) {
		return fmt.Errorf("対象の自動切替設定がありません。")
	}
	b := a.config.AutoSwitch.Bindings[index]
	b.Id = newBindingID()
	b.Name += " のコピー"
	insert := index + 1
	old := a.config.AutoSwitch.Bindings
	next := make([]AppBinding, 0, len(old)+1)
	next = append(next, old[:insert]...)
	next = append(next, b)
	next = append(next, old[insert:]...)
	a.config.AutoSwitch.Bindings = next
	return a.saveConfigLocked()
}

func (a *App) webDeleteBinding(index int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if index < 0 || index >= len(a.config.AutoSwitch.Bindings) {
		return fmt.Errorf("対象の自動切替設定がありません。")
	}
	id := a.config.AutoSwitch.Bindings[index].Id
	a.config.AutoSwitch.Bindings = append(a.config.AutoSwitch.Bindings[:index], a.config.AutoSwitch.Bindings[index+1:]...)
	if a.autoBindingID == id {
		a.clearAutoMatchLocked()
		a.rebuildRulesLocked()
	}
	a.autoCandidateKey = ""
	a.autoCandidateSince = time.Time{}
	return a.saveConfigLocked()
}

func (a *App) webMoveBinding(index, target, delta int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	bindings := &a.config.AutoSwitch.Bindings
	if index < 0 || index >= len(*bindings) {
		return fmt.Errorf("対象の自動切替設定がありません。")
	}
	if target < 0 {
		target = index + delta
	}
	if target < 0 {
		target = 0
	}
	if target >= len(*bindings) {
		target = len(*bindings) - 1
	}
	if target == index {
		return nil
	}
	b := (*bindings)[index]
	without := make([]AppBinding, 0, len(*bindings)-1)
	without = append(without, (*bindings)[:index]...)
	without = append(without, (*bindings)[index+1:]...)
	if target > len(without) {
		target = len(without)
	}
	next := make([]AppBinding, 0, len(without)+1)
	next = append(next, without[:target]...)
	next = append(next, b)
	next = append(next, without[target:]...)
	*bindings = next
	a.autoCandidateKey = ""
	a.autoCandidateSince = time.Time{}
	return a.saveConfigLocked()
}

func (a *App) webAPIImportJSON(w http.ResponseWriter, r *http.Request) {
	var cfg Config
	if !decodeJSON(w, r, &cfg) {
		return
	}
	if len(cfg.Profiles) == 0 {
		writeError(w, fmt.Errorf("プロファイルが入っていません。"))
		return
	}
	if err := a.backupConfig(); err != nil {
		a.logf("backup before import failed: %v", err)
	}
	a.mu.Lock()
	a.config = normalizeConfig(cfg)
	a.editorProfileIndex = a.profileIndexByIDLocked(a.config.ActiveProfileId)
	a.clearAutoMatchLocked()
	if err := a.saveConfigLocked(); err != nil {
		a.mu.Unlock()
		writeError(w, err)
		return
	}
	a.rebuildRulesLocked()
	a.mu.Unlock()
	writeJSON(w, map[string]any{"ok": true, "state": a.buildWebState()})
}

func (a *App) webAPIDefaultRules(w http.ResponseWriter, r *http.Request) {
	cfg := mustDefaultConfig()
	out := []webRule{}
	if len(cfg.Profiles) > 0 {
		for i, rr := range cfg.Profiles[0].Rules {
			out = append(out, webRule{Index: i, Enabled: rr.Enabled, Input: itemsText(rr.Input), Mode: normalizeJoyConRuleMode(rr.Mode), Output: itemsText(rr.Output), SuppressTrigger: rr.SuppressTrigger, SuppressPrefix: rr.SuppressPrefix})
		}
	}
	writeJSON(w, map[string]any{"rules": out})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return false
	}
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 4<<20))
	if err := dec.Decode(dst); err != nil {
		writeError(w, err)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
}

func (a *App) loadAppIcon() uintptr {
	if a.appIcon != 0 {
		return a.appIcon
	}
	candidates := []string{}
	if a.iconPath != "" {
		candidates = append(candidates, a.iconPath)
	}
	candidates = append(candidates, filepath.Join(filepath.Dir(a.configPath), "MouseButtonMapper.ico"))
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			icon, _, _ := pLoadImageW.Call(0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(p))), IMAGE_ICON, 0, 0, LR_LOADFROMFILE|LR_DEFAULTSIZE)
			if icon != 0 {
				a.appIcon = icon
				return icon
			}
		}
	}
	icon, _, _ := pLoadIconW.Call(0, uintptr(32512))
	a.appIcon = icon
	return icon
}
func (a *App) trayData() NOTIFYICONDATA {
	var nid NOTIFYICONDATA
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = a.hwnd
	nid.UID = 1
	nid.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	nid.UCallbackMessage = trayMsg
	nid.HIcon = a.loadAppIcon()
	tip := appName + " " + appVersion
	a.mu.RLock()
	if a.emergency {
		tip += " - Emergency Stop"
	} else if !a.enabled {
		tip += " - Stopped"
	} else {
		tip += " - Running"
	}
	a.mu.RUnlock()
	copy(nid.SzTip[:], syscall.StringToUTF16(tip))
	return nid
}
func (a *App) addTray() {
	if a.hwnd == 0 {
		return
	}
	nid := a.trayData()
	r, _, e := pShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))
	if r == 0 {
		a.logf("Shell_NotifyIcon ADD failed: %v", e)
	}
}
func (a *App) updateTrayTip() {
	if a.hwnd == 0 {
		return
	}
	nid := a.trayData()
	pShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
}
func (a *App) removeTray() {
	if a.hwnd != 0 {
		nid := a.trayData()
		pShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
	}
}
func (a *App) showTrayMenu() {
	menu, _, _ := pCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	a.mu.RLock()
	running := a.enabled && !a.emergency
	a.mu.RUnlock()
	appendMenu(menu, MF_STRING, ID_TRAY_SHOW, "GUIを開く")
	if running {
		appendMenu(menu, MF_STRING, ID_TRAY_STARTSTOP, "停止")
	} else {
		appendMenu(menu, MF_STRING, ID_TRAY_STARTSTOP, "開始")
	}
	appendMenu(menu, MF_STRING, ID_TRAY_EMERGENCY, "緊急停止")
	appendMenu(menu, MF_STRING, ID_TRAY_RELEASE, "修飾キー解放")
	appendMenu(menu, MF_STRING, ID_TRAY_RELOAD, "設定再読み込み")
	appendMenu(menu, MF_SEPARATOR, 0, "")
	appendMenu(menu, MF_STRING, ID_TRAY_OPENCFG, "config.jsonを開く")
	appendMenu(menu, MF_STRING, ID_TRAY_OPENFOLDER, "設定フォルダーを開く")
	appendMenu(menu, MF_STRING, ID_TRAY_ABOUT, "情報")
	appendMenu(menu, MF_SEPARATOR, 0, "")
	appendMenu(menu, MF_STRING, ID_TRAY_EXIT, "終了")
	var pt POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	pSetForegroundWindow.Call(a.hwnd)
	pTrackPopupMenu.Call(menu, TPM_RIGHTBUTTON|TPM_BOTTOMALIGN, uintptr(pt.X), uintptr(pt.Y), 0, a.hwnd, 0)
	pDestroyMenu.Call(menu)
}
func appendMenu(menu uintptr, flags uint32, id uint32, text string) {
	pAppendMenuW.Call(menu, uintptr(flags), uintptr(id), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))))
}
func messageBox(title, text string) {
	pMessageBoxW.Call(0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))), 0)
}
