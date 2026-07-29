# MouseButtonMapper 引き継ぎ正本

この文書は、リポジトリURLだけを渡された別スレッド／開発者が、現状を誤認せず作業を再開するための正本です。

## 1. Repositoryと状態

- Repository: `https://github.com/GoodLight999/MouseButtonMapper`
- 安定版: `main` / `8.3.0`
- 開発branch: `feature/joycon-l`
- Draft PR: `#5`
- 現在のRC: `8.4.0-rc3`
- branch起点: `3126d531cff93c519bdf3634847d553033787940`
- 最終head、全緑CI run、artifact ID、SHA-256: PR #5本文を正本とする

PRは実機確認完了までDraft・未マージを維持する。過去チャット添付、古いRC、`trash`／archiveを正本にしない。

## 2. 製品方針

MouseButtonMapperの中核は、安定したマウス／キーボード割り当てです。ゲームコントローラー対応は任意の実験機能であり、中核機能を巻き込んで失敗させてはいけません。

### 全体コントローラーゲート

`Config.Controller.Enabled`が全入力経路の唯一の全体ゲートです。

- `Config.Version`: 10
- 新規設定: OFF
- Version 9以前からの移行: OFF
- OFF時:
  - Joy-Con／互換Raw HID workerを起動しない
  - XInput workerを起動しない
  - 遅れて届くコントローラーイベントを無視する
  - コントローラー詳細UIとHold設定を隠す
  - コントローラー入力を含むルールをactive rulesから外す
  - 保存済みrule／profile／Joy-Con設定は削除しない
  - コントローラー由来の長押し・Hold・DOWN状態を解放する
  - マウス／キーボードのDOWN・長押し・ルールは維持する
- ON時:
  - Joy-Con／互換HIDとXInput subsystemを開始する
  - 保存済みコントローラールールを再びactive rulesへ含める

BetterJoy／JoyToKeyへ外注する場合はOFFのまま使う。外部ツールが生成するキーボード／マウス入力は通常経路で処理できる。

## 3. 実装済み機能

### 中核

- Windows低レベルマウス／キーボードフック
- action queue経由の`SendInput`
- 入出力記録、短押し、長押し実行／キャンセル
- プロファイルと前面アプリ自動切替
- 多重起動拒否、トレイ、緊急停止、フックwatchdog
- 旧設定互換と安全な設定保存

### 任意コントローラー入力

- 純正Joy-Con（L）: Nintendo HID `057e:2006`
- Switch互換Raw HID:
  - Generic Desktop Game Pad／Joystickだけを候補化
  - 未知VID/PIDは自動接続せずStable IDを手動選択
  - `0x21`、`0x30`、`0x3f`、7バイト／先頭0付き8バイトinput-only報告
  - 書込み非対応時のread-only fallback
  - 未対応report長と先頭hexの診断
- XInput:
  - `xinput1_4`→`xinput1_3`→`xinput9_1_0`の順で動的ロード
  - P1～P4
  - A/B/X/Y、D-pad、LB/RB、Start/Back、stick press、LT/RT、左右stick方向
  - trigger／stickヒステリシス、切断時の合成UP
- 共通ルール:
  - コントローラー＋マウス／キーボード複合入力
  - Tap、長押し、Hold
  - キー、同時キー、マウスボタン、ホイール出力
  - 切断／OFF／reload／緊急停止／終了時の安全解放

## 4. 重要なデバッグ結果

- rc1が互換Joy-Conを認識しなかった主因は、列挙とOpenの双方で純正VID/PIDだけを許可していたこと。
- Steamだけが入力を取れる互換品は、Windows XInput／通常DirectInputではなくSwitch系Raw HIDをSteam/SDLが直接解釈している可能性が高い。
- rc2でRaw HID候補とXInputを追加した。
- rc3でコントローラー機能をdefault OFFの全体ゲート配下へ移動した。
- rc3監査で、XInput長押しのkey生成／validationがJoy-Conだけに限定されていた不具合を修正した。
- コントローラーOFF処理がマウス／キーボード長押しまで全消去しないことを回帰試験で固定した。

## 5. 並行処理の不変条件

- `WH_MOUSE_LL`／`WH_KEYBOARD_LL` callbackでSleep、HID I/O、設定保存、ファイルlog、`SendInput`を行わない。
- 出力は`actionCh`へ渡し、output workerが実行する。
- Joy-Con I/Oは`JoyConWorker`、XInputは`XInputWorker`で行う。
- OFFへ切り替える前にconfig gateを下げ、遅延callbackを入口で捨てる。
- HID `WriteFile`とhandle closeは同じmutexで直列化する。
- 切断／停止時はDOWN中トークンへ合成UPを出し、Hold keyを残さない。
- 物理的に押されているキーを合成UPで解放しない。
- コントローラー失敗は正常な縮退状態であり、マウス／キーボード機能を止めない。

## 6. 設定互換性

```text
Config.Version = 10
Config.Controller.Enabled = false  // default
Profile.JoyCon                    // optional, preserved while global OFF
Rule.Input Kind                   // Mouse / Key / JoyCon / XInput
```

全体OFFは設定削除ではなくruntime filteringで実現する。ON/OFFのたびにrule、profile、calibration、Stable IDを消してはいけない。

## 7. 主要ファイル

- `main.go`: App、Config、rule rebuild、HTTP API、hooks、profiles
- `controller_feature_windows.go`: 全体ゲート、worker同期、API
- `controller_input_windows.go`: Joy-Con／XInput共通状態、record、Tap、長押し、Hold
- `longpress_windows.go`: 長押し状態機械。Joy-ConとXInputの両方を扱う
- `joycon_hid_windows.go`: Windows SetupAPI／HID I/O
- `joycon_worker.go`: 列挙、接続、read、reconnect、rescan gate
- `joycon_logic.go`: report parserとstick処理
- `joycon_app_windows.go`: Joy-Con subsystem接続
- `xinput_logic.go`: XInput差分と閾値
- `xinput_windows.go`: DLL load、P1～P4 polling
- `joycon_ui_windows.go`: 全体toggleと詳細UI
- `joycon_web_windows.go`: controller/Joy-Con API state
- `controller_feature_windows_test.go`: default OFF、worker停止、rule保存、late event、長押し分離
- `docs/REGRESSION_CHECKLIST.md`: 実機検証正本

## 8. 必須自動検証

```text
gofmt
python scripts/check_public_repo.py
go test ./...
go test -race ./...
go vet -unsafeptr=false ./...
Windows go test ./...
Windows go vet -unsafeptr=false ./...
Windows GUI build/package
```

UI JavaScriptは抽出して`node --check`も行う。最終配布は同一branch headから生成し、次を独立確認する。

- Actions artifact digest
- SHA-256 manifest
- standalone EXEとportable内部EXEのバイト一致
- source ZIP commentと最終head SHAの一致
- PE32+ Windows GUI x86-64

## 9. 実機ゲート

### 全体OFF

- 起動直後に詳細controller UIが隠れている
- HID列挙／XInput監視が走らない
- mouse/key rule、long press、profile switchが動く
- BetterJoy／JoyToKeyのkeyboard/mouse出力を通常入力として扱える
- ON→Hold中→OFFでkeyが残留しない
- OFF→ONで保存済みcontroller ruleが復帰する

### ON時

- 純正Joy-Conの接続、全button、stick、battery、sleep/resume
- 互換Raw HIDのcandidate選択、report診断、全input
- XInput P1～P4のbuttons/triggers/sticks
- disconnect/reconnect 20回
- controller＋right-hand mouse/keyboard
- game側との二重入力
- 1時間連続input、8時間常駐

完了までは安定版と呼ばず、PRをReady／Mergeしない。

## 10. 禁止事項

- controller featureを暗黙にONへ移行しない。
- OFF時に保存済みcontroller設定を削除しない。
- OFF時にmouse/key stateやlong pressを巻き添えで消さない。
- 未知HIDへ無差別にNintendo subcommandを送らない。
- keyboard/mouse HIDをcontroller candidateにしない。
- hook callbackへHID/XInput処理を入れない。
- 実機未検証を「完全対応」「安定版」と断定しない。
- PR #5を実機確認前にmergeしない。

## 11. 次の担当者が最初に行うこと

1. PR #5本文で最終head／CI／artifactを確認する。
2. `Controller.Enabled=false`のruntime pathを先に回帰確認する。
3. `docs/REGRESSION_CHECKLIST.md`の全体ゲートから実機試験を始める。
4. 互換品が未対応なら、診断に出るVID/PID、Usage、report length、先頭hexを基にparser testを先に追加する。
5. 全hardware gate完了後だけ8.4.0 stable化を検討する。
