from pathlib import Path


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise RuntimeError(f"{label}: expected exactly one anchor, found {count}")
    return text.replace(old, new, 1)


main_path = Path("main.go")
main = main_path.read_text(encoding="utf-8")

main = replace_once(
    main,
    '''type Profile struct {
\tId    string `json:"Id"`
\tName  string `json:"Name"`
\tRules []Rule `json:"Rules"`
}''',
    '''type Profile struct {
\tId     string              `json:"Id"`
\tName   string              `json:"Name"`
\tRules  []Rule              `json:"Rules"`
\tJoyCon JoyConProfileConfig `json:"JoyCon,omitempty"`
}''',
    "Profile JoyCon field",
)

main = replace_once(
    main,
    '''\tmouseDown      map[string]bool
\tmouseDownAt    map[string]time.Time
\tkeyDown        map[uint32]bool
\tpendingTap     map[string]bool''',
    '''\tmouseDown       map[string]bool
\tmouseDownAt     map[string]time.Time
\tkeyDown         map[uint32]bool
\tjoyConDown      map[string]bool
\tjoyConPending   map[string]bool
\tjoyConConsumed  map[string]bool
\tjoyConHoldRules map[string]Rule
\tpendingTap      map[string]bool''',
    "App JoyCon input maps",
)

main = replace_once(
    main,
    '''\tlongPress      map[string]*longPressState
\tlongPressSeq   uint64

\tsendMu''',
    '''\tlongPress      map[string]*longPressState
\tlongPressSeq   uint64

\tjoyConWorker     *JoyConWorker
\tjoyConCancel     func()
\tjoyConDone       chan struct{}
\tjoyConStatus     JoyConConnectionStatus
\tjoyConOutputRefs map[uint32]joyConOutputReference

\tsendMu''',
    "App JoyCon runtime fields",
)

main = replace_once(
    main,
    'keyDown: map[uint32]bool{}, pendingTap:',
    'keyDown: map[uint32]bool{}, joyConDown: map[string]bool{}, joyConPending: map[string]bool{}, joyConConsumed: map[string]bool{}, joyConHoldRules: map[string]Rule{}, pendingTap:',
    "global App JoyCon map initialization",
)

main = replace_once(
    main,
    'longPress: map[string]*longPressState{}, logCh:',
    'longPress: map[string]*longPressState{}, joyConOutputRefs: map[uint32]joyConOutputReference{}, joyConStatus: JoyConConnectionStatus{BatteryPercent: -1}, logCh:',
    "global App JoyCon output initialization",
)

main = replace_once(
    main,
    '''\tapp.startAutoSwitchWorker()
\tif err := app.startWebServer(); err != nil {''',
    '''\tapp.startAutoSwitchWorker()
\tapp.startJoyConSubsystem()
\tif err := app.startWebServer(); err != nil {''',
    "start JoyCon subsystem",
)

main = replace_once(
    main,
    '''\tprobeRule := Rule{Enabled: true, Input: []Item{{Kind: "Mouse", Code: "X1"}}, Mode: "Tap", LongPressEnabled: true, LongPressMs: 500, LongPressAction: longPressActionCancel, Output: []Item{{Kind: "Key", Code: "65"}}}
\tif err := validateLongPressRule(probeRule); err != nil {
\t\treturn fmt.Errorf("long press rule self-test failed: %w", err)
\t}
\tprobe := AppBinding''',
    '''\tprobeRule := Rule{Enabled: true, Input: []Item{{Kind: "Mouse", Code: "X1"}}, Mode: "Tap", LongPressEnabled: true, LongPressMs: 500, LongPressAction: longPressActionCancel, Output: []Item{{Kind: "Key", Code: "65"}}}
\tif err := validateLongPressRule(probeRule); err != nil {
\t\treturn fmt.Errorf("long press rule self-test failed: %w", err)
\t}
\tjoyConProbe := Rule{Enabled: true, Input: []Item{{Kind: "JoyCon", Code: string(JoyConStickUp)}}, Mode: joyConRuleModeHold, Output: []Item{{Kind: "Key", Code: "W"}}}
\tif err := validateJoyConHoldRule(joyConProbe); err != nil {
\t\treturn fmt.Errorf("Joy-Con hold rule self-test failed: %w", err)
\t}
\tjoyConReport := make([]byte, 12)
\tjoyConReport[0] = joyConReportFull
\tjoyConReport[5] = 1 << 7
\tjoyConReport[6] = 0xd0
\tjoyConReport[7] = 0x07
\tjoyConReport[8] = 0x7d
\tif state, err := parseJoyConInputReport(joyConReport); err != nil || !state.Buttons[JoyConButtonZL] {
\t\treturn fmt.Errorf("Joy-Con report parser self-test failed: %v", err)
\t}
\tprobe := AppBinding''',
    "JoyCon self test",
)

main = replace_once(
    main,
    '''\t\tif strings.TrimSpace(cfg.Profiles[i].Name) == "" {
\t\t\tcfg.Profiles[i].Name = fmt.Sprintf("プロファイル %d", i+1)
\t\t}
\t\tfor j := range cfg.Profiles[i].Rules {
\t\t\tr := &cfg.Profiles[i].Rules[j]
\t\t\tif r.LongPressEnabled {''',
    '''\t\tif strings.TrimSpace(cfg.Profiles[i].Name) == "" {
\t\t\tcfg.Profiles[i].Name = fmt.Sprintf("プロファイル %d", i+1)
\t\t}
\t\tcfg.Profiles[i].JoyCon = normalizeJoyConProfileConfig(cfg.Profiles[i].JoyCon)
\t\tfor j := range cfg.Profiles[i].Rules {
\t\t\tr := &cfg.Profiles[i].Rules[j]
\t\t\tr.Mode = normalizeJoyConRuleMode(r.Mode)
\t\t\tif r.LongPressEnabled {''',
    "normalize profile JoyCon settings",
)

main = replace_once(
    main,
    '''\ta.rebuildRulesLocked()
\ta.logf("loaded config:''',
    '''\ta.rebuildRulesLocked()
\ta.requestJoyConRescanLocked()
\ta.logf("loaded config:''',
    "rescan JoyCon after config apply",
)

main = replace_once(
    main,
    '''func (a *App) physicalInputIdleLocked() bool {
\treturn len(a.mouseDown) == 0 && len(a.keyDown) == 0
}''',
    '''func (a *App) physicalInputIdleLocked() bool {
\treturn len(a.mouseDown) == 0 && len(a.keyDown) == 0 && len(a.joyConDown) == 0
}''',
    "JoyCon profile switch boundary",
)

main = replace_once(
    main,
    '''\tif idx >= len(a.config.Profiles) {
\t\ta.activeProfileIndex = 0
\t\ta.rules = nil
\t\treturn
\t}''',
    '''\tif idx >= len(a.config.Profiles) {
\t\ta.activeProfileIndex = 0
\t\ta.rules = nil
\t\ta.requestJoyConRescanLocked()
\t\treturn
\t}''',
    "JoyCon rescan on empty effective profile",
)

main = replace_once(
    main,
    '''\tfor _, r := range prof.Rules {
\t\tif !r.Enabled || !strings.EqualFold(r.Mode, "Tap") || len(r.Input) == 0 || !ruleHasRunnableOutput(r) || isDangerousSingleReplacement(r) {
\t\t\tcontinue
\t\t}
\t\tif err := validateLongPressRule(r); err != nil {
\t\t\tcontinue
\t\t}
\t\trules = append(rules, r)
\t}
\ta.rules = rules''',
    '''\tfor _, r := range prof.Rules {
\t\tr.Mode = normalizeJoyConRuleMode(r.Mode)
\t\tif !r.Enabled || len(r.Input) == 0 || !ruleHasRunnableOutput(r) || isDangerousSingleReplacement(r) {
\t\t\tcontinue
\t\t}
\t\tif err := validateJoyConHoldRule(r); err != nil {
\t\t\tcontinue
\t\t}
\t\tif err := validateLongPressRule(r); err != nil {
\t\t\tcontinue
\t\t}
\t\trules = append(rules, r)
\t}
\ta.rules = rules
\ta.requestJoyConRescanLocked()''',
    "activate JoyCon hold rules",
)

main = replace_once(
    main,
    '''func (a *App) cleanup() {
\t// タイマーやUI処理が終了処理と競合しても、終了開始後に新しい入力を''',
    '''func (a *App) cleanup() {
\ta.stopJoyConSubsystem()
\t// タイマーやUI処理が終了処理と競合しても、終了開始後に新しい入力を''',
    "stop JoyCon before shutdown",
)

main = replace_once(
    main,
    '''\ta.mu.Lock()
\ta.abortAllLongPressLocked("application exit", true)
\ta.mu.Unlock()''',
    '''\ta.mu.Lock()
\ta.abortAllLongPressLocked("application exit", true)
\ta.clearJoyConInputStateLocked("application exit")
\ta.mu.Unlock()''',
    "clear JoyCon on exit",
)

main = replace_once(
    main,
    '''\tif emergency {
\t\ta.abortAllLongPressLocked("emergency stop", false)
\t\ta.recordingMode = ""''',
    '''\tif emergency {
\t\ta.abortAllLongPressLocked("emergency stop", false)
\t\ta.clearJoyConInputStateLocked("emergency stop")
\t\ta.recordingMode = ""''',
    "clear JoyCon on emergency",
)

main = replace_once(
    main,
    '''\tfor _, btn := range []string{"Left", "Right", "Middle", "X1", "X2"} {
\t\tif a.mouseDown[btn] {
\t\t\tit := Item{Kind: "Mouse", Code: btn}
\t\t\ta.appendRecordedItemLocked(it)
\t\t\ta.recordHeld[recordItemKey(it)] = true
\t\t}
\t}
}''',
    '''\tfor _, btn := range []string{"Left", "Right", "Middle", "X1", "X2"} {
\t\tif a.mouseDown[btn] {
\t\t\tit := Item{Kind: "Mouse", Code: btn}
\t\t\ta.appendRecordedItemLocked(it)
\t\t\ta.recordHeld[recordItemKey(it)] = true
\t\t}
\t}
\tfor _, code := range append(append([]JoyConButton(nil), joyConLeftPhysicalButtons...), joyConStickDirections...) {
\t\tif a.joyConDown[string(code)] {
\t\t\tit := Item{Kind: "JoyCon", Code: string(code)}
\t\t\ta.appendRecordedItemLocked(it)
\t\t\ta.recordHeld[recordItemKey(it)] = true
\t\t}
\t}
}''',
    "record held JoyCon prefixes",
)

main = replace_once(
    main,
    '''\tif strings.EqualFold(it.Kind, "Key") {
\t\tif vk, ok := parseVK(it.Code); ok {
\t\t\treturn Item{Kind: "Key", Code: strconv.Itoa(int(genericVK(vk)))}
\t\t}
\t}
\treturn it
}''',
    '''\tif strings.EqualFold(it.Kind, "Key") {
\t\tif vk, ok := parseVK(it.Code); ok {
\t\t\treturn Item{Kind: "Key", Code: strconv.Itoa(int(genericVK(vk)))}
\t\t}
\t}
\tif strings.EqualFold(it.Kind, "JoyCon") {
\t\treturn Item{Kind: "JoyCon", Code: normalizeJoyConCode(it.Code)}
\t}
\treturn it
}''',
    "normalize recorded JoyCon input",
)

main = replace_once(
    main,
    '''\tif strings.EqualFold(it.Kind, "Key") {
\t\tvk, ok := parseVK(it.Code)
\t\treturn ok && (a.keyDown[vk] || a.keyDown[genericVK(vk)])
\t}
\treturn false
}''',
    '''\tif strings.EqualFold(it.Kind, "Key") {
\t\tvk, ok := parseVK(it.Code)
\t\treturn ok && (a.keyDown[vk] || a.keyDown[genericVK(vk)])
\t}
\tif strings.EqualFold(it.Kind, "JoyCon") {
\t\treturn a.joyConDown[normalizeJoyConCode(it.Code)]
\t}
\treturn false
}''',
    "shared JoyCon pressed state",
)

main = replace_once(
    main,
    '''\tif strings.EqualFold(a.Kind, "Key") {
\t\tav, aok := parseVK(a.Code)
\t\tbv, bok := parseVK(b.Code)
\t\treturn aok && bok && genericVK(av) == genericVK(bv)
\t}
\treturn false
}''',
    '''\tif strings.EqualFold(a.Kind, "Key") {
\t\tav, aok := parseVK(a.Code)
\t\tbv, bok := parseVK(b.Code)
\t\treturn aok && bok && genericVK(av) == genericVK(bv)
\t}
\tif strings.EqualFold(a.Kind, "JoyCon") {
\t\treturn normalizeJoyConCode(a.Code) == normalizeJoyConCode(b.Code)
\t}
\treturn false
}''',
    "compare JoyCon inputs",
)

main = replace_once(
    main,
    '''func (a *App) markPrefixesConsumedLocked(r Rule) {
\tfor i := 0; i < len(r.Input)-1; i++ {
\t\tit := r.Input[i]
\t\ta.abortLongPressForTriggerLocked(it, "input became a prefix of a longer rule")
\t\tif strings.EqualFold(it.Kind, "Mouse") {
\t\t\ta.consumedPrefix[normMouse(it.Code)] = true
\t\t}
\t}
}''',
    '''func (a *App) markPrefixesConsumedLocked(r Rule) {
\tfor i := 0; i < len(r.Input)-1; i++ {
\t\tit := r.Input[i]
\t\ta.abortLongPressForTriggerLocked(it, "input became a prefix of a longer rule")
\t\tif strings.EqualFold(it.Kind, "Mouse") {
\t\t\ta.consumedPrefix[normMouse(it.Code)] = true
\t\t}
\t\tif strings.EqualFold(it.Kind, "JoyCon") {
\t\t\tcode := normalizeJoyConCode(it.Code)
\t\t\ta.joyConConsumed[code] = true
\t\t\tif holdRule, ok := a.joyConHoldRules[code]; ok {
\t\t\t\tdelete(a.joyConHoldRules, code)
\t\t\t\ta.enqueueRuleGuaranteed(joyConHoldPhaseRule(holdRule, false))
\t\t\t}
\t\t}
\t}
}''',
    "consume JoyCon prefixes",
)

main = replace_once(
    main,
    '''func (a *App) sendRule(r Rule) {
\tkeys := make([]uint32, 0, len(r.Output))
\tunsupported := []string{}
\tfor _, it := range r.Output {
\t\tif strings.EqualFold(it.Kind, "Key") {
\t\t\tif vk, ok := parseVK(it.Code); ok {
\t\t\t\tkeys = append(keys, vk)
\t\t\t} else {
\t\t\t\tunsupported = append(unsupported, it.Code)
\t\t\t}
\t\t} else {
\t\t\tunsupported = append(unsupported, it.Kind+":"+it.Code)
\t\t}
\t}
\tif len(unsupported) > 0 {
\t\ta.logf("unsupported output ignored: %v", unsupported)
\t}
\tif len(keys) > 0 {
\t\ta.sendShortcut(keys)
\t}
}''',
    '''func (a *App) sendRule(r Rule) {
\ta.sendJoyConRuleOutput(r)
}''',
    "route outputs through shared JoyCon-capable sender",
)

main = replace_once(
    main,
    '''\tfor _, vk := range mods {
\t\tinputs = append(inputs, makeKeyInput(vk, true))
\t}
\ta.callSendInput(inputs)''',
    '''\tfor _, vk := range mods {
\t\tinputs = append(inputs, makeKeyInput(vk, true))
\t}
\tinputs = a.appendJoyConHeldReleaseInputs(inputs)
\ta.callSendInput(inputs)''',
    "release JoyCon held outputs",
)

main = replace_once(
    main,
    '''\ta.abortAllLongPressLocked("hooks reinstalled", true)
\tif a.recordingMode != "" {''',
    '''\ta.abortAllLongPressLocked("hooks reinstalled", true)
\ta.requestJoyConRescanLocked()
\tif a.recordingMode != "" {''',
    "rescan JoyCon after hook reset",
)

main_path.write_text(main, encoding="utf-8")

long_path = Path("longpress_windows.go")
longpress = long_path.read_text(encoding="utf-8")

longpress = replace_once(
    longpress,
    '''\tif strings.EqualFold(it.Kind, "Key") {
\t\tif vk, ok := parseVK(it.Code); ok {
\t\t\treturn "key:" + strconv.Itoa(int(genericVK(vk)))
\t\t}
\t}
\treturn ""
}''',
    '''\tif strings.EqualFold(it.Kind, "Key") {
\t\tif vk, ok := parseVK(it.Code); ok {
\t\t\treturn "key:" + strconv.Itoa(int(genericVK(vk)))
\t\t}
\t}
\tif strings.EqualFold(it.Kind, "JoyCon") {
\t\tcode := normalizeJoyConCode(it.Code)
\t\tif isKnownJoyConCode(code) {
\t\t\treturn "joycon:" + strings.ToLower(code)
\t\t}
\t}
\treturn ""
}''',
    "JoyCon long press key",
)

longpress = replace_once(
    longpress,
    '''func isHoldableLongPressTrigger(it Item) bool {
\tif strings.EqualFold(it.Kind, "Key") {
\t\tvk, ok := parseVK(it.Code)
\t\treturn ok && !isModifier(vk)
\t}
\tif !strings.EqualFold(it.Kind, "Mouse") {
\t\treturn false
\t}''',
    '''func isHoldableLongPressTrigger(it Item) bool {
\tif strings.EqualFold(it.Kind, "Key") {
\t\tvk, ok := parseVK(it.Code)
\t\treturn ok && !isModifier(vk)
\t}
\tif strings.EqualFold(it.Kind, "JoyCon") {
\t\treturn isKnownJoyConCode(it.Code)
\t}
\tif !strings.EqualFold(it.Kind, "Mouse") {
\t\treturn false
\t}''',
    "JoyCon holdable long press",
)

longpress = replace_once(
    longpress,
    '長押し判定の最後の入力には、サイド1・サイド2・修飾キー以外のキーボードキーを指定してください',
    '長押し判定の最後の入力には、Joy-Conボタン・サイド1・サイド2・修飾キー以外のキーボードキーを指定してください',
    "JoyCon long press validation message",
)

long_path.write_text(longpress, encoding="utf-8")
