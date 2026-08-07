# Architecture

## 中核データフロー

```text
physical mouse / keyboard
        │
        ▼
WH_MOUSE_LL / WH_KEYBOARD_LL dedicated thread
        │  short state/suppression decision
        ├────────► CallNextHookEx
        ▼
actionCh
        ▼
outputWorker
        ▼
output_windows.go
        ▼
SendInput(extraInfoMarker)
```

フックcallbackを短く保つことが最優先です。`output_windows.go`はcontroller subsystemから独立した中核で、KeyとMouse（Left/Right/Middle/X1/X2/WheelUp/WheelDown）を同じrule outputとして処理します。

## Runtime workers

- Hook thread: low-level hooksとmessage loop
- Output worker: queued rule output
- Config writer: JSON保存
- Hook watchdog: hook再登録
- Foreground monitor: WinEventHook＋fallback polling
- HTTP server: `127.0.0.1`のrandom port
- Long-press timers: callbackは結果をqueueへ渡すだけ
- Joy-Con worker: HID enumerate/open/read/reconnect
- XInput worker: dynamic DLL loadとP1～P4 polling

## 全体コントローラーゲート

```text
Config.Controller.Enabled
        │
        ├─ false ─► controller workers absent
        │           controller events ignored
        │           detailed UI hidden
        │           controller rules preserved but inactive
        │
        └─ true ──► JoyConWorker + XInputWorker
                    shared controller rule adapter
```

ゲートはworker開始、event入口、rule rebuild、UIの四層で適用します。一層だけに依存してはいけません。

OFF切替順序:

1. config gateをfalseにする
2. active rulesからcontroller ruleを除外する
3. controller DOWN／pending／long-press／Hold outputを解放する
4. workersを停止する
5. statusをdisabledへ戻す

これにより停止途中の遅延callbackは入口で破棄されます。mouse/key stateは変更しません。

## Controller data flow

```text
Nintendo / compatible Raw HID          XInput
              │                          │
              ▼                          ▼
        JoyConWorker                XInputWorker
              │                          │
              └──────────┬───────────────┘
                         ▼
          handleControllerInputEvent
                         │
                         ▼
shared pressed set / recording / combinations /
Tap / long press / Hold / output queue
```

保存形式は`Item{Kind, Code}`です。

- Joy-Con: `JoyCon:L`, `JoyCon:StickUp`
- XInput: `XInput:P1:LB`, `XInput:P2:A`

runtime state keyにはKindを含め、source間のcollisionを防ぎます。

## Raw HID safety

- BetterJoyの3rd-party登録方式を参考に、Windowsが公開する全HID interfaceを列挙する
- 純正Joy-Conは既知VID/PIDでauto detect可能
- unknown HIDは自動Openせず、UIで正確なpath fingerprintを明示登録したinterfaceだけを左Joy-Con互換として扱う
- 登録はVID/PID/serial/fingerprintを保持し、exact pathを優先してserialを再接続fallbackにする
- 明示登録済みdeviceだけR/W OpenとNintendo report-mode commandを試し、拒否時はread-onlyへ縮退する
- unsupported reportを別形式として推測せず、lengthとbounded hexを診断する
- closeとwriteをmutexで直列化し、blocking readは`CancelIoEx`で解除する

## XInput safety

- DLLは利用可能なものをruntime loadする
- packet numberの変化だけを差分処理する
- trigger／stickにpress/releaseの別thresholdを使う
- direction changeはrelease-before-press
- disconnect/API errorで全DOWNへsynthetic UP

## Rule and long-press state

- controller feature OFFはstored ruleを変更せず、active rule rebuildでfilterする
- mouse/key long-pressとcontroller long-pressは別key namespaceを使う
- Joy-ConとXInputの両方をcontroller long-press triggerとしてvalidateする
- timerとUPが競合しても一度だけcompleteする
- Hold outputはreference countし、physical keyを勝手にreleaseしない

## Profile state

- Base profile: 通常時profile
- Editor profile: GUIで編集中
- Effective profile: runtime適用中

三者を同じ変数へ統合しない。controller setting/calibration saveもprofile IDへbindする。

## Config compatibility

`Config.Version`は11です。`Controller.Visible`と`Controller.Enabled`の欠落はfalseとして扱います。Visible=falseでは専用UIを生成せず、Enabled=falseではworkerを起動しません。新fieldは原則optionalとし、旧profile/rule/auto-switchを保存時に失わないようにします。

## Security boundary

HTTP serverはlocalhostだけへbindします。外部interfaceへ公開しません。自己注入eventは`extraInfoMarker`で除外します。
