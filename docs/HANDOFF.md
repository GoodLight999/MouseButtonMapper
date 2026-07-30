# MouseButtonMapper 引き継ぎ正本

この文書は、リポジトリURLだけを渡された別スレッド／開発者が、現状を誤認せず作業を再開するための正本です。

## 1. Repositoryと状態

- Repository: `https://github.com/GoodLight999/MouseButtonMapper`
- 安定版: `main` / `8.3.0`
- 開発branch: `feature/joycon-l`
- Draft PR: `#5`
- 現在のRC: `8.4.0-rc4`
- branch起点: `3126d531cff93c519bdf3634847d553033787940`
- 最終head、全緑CI run、artifact ID、SHA-256: PR #5本文を正本とする

PRは実機確認完了までDraft・未マージを維持する。過去チャット添付、古いRC、`trash`／archiveを正本にしない。

## 2. 製品方針

MouseButtonMapperの中核は、安定したマウス／キーボード割り当てです。ゲームコントローラー対応は任意の実験機能であり、中核機能を巻き込んで失敗させてはいけません。

### 全体コントローラーゲート

`Config.Controller.Enabled`が全入力経路の実行ゲート、`Config.Controller.Visible`が専用UIの表示ゲートです。

- `Config.Version`: 11
- 新規設定: `Visible=false / Enabled=false`
- Version 10以前からの移行: Enabledだった場合だけVisibleを維持。それ以外は非表示・OFF
- OFF時:
  - Joy-Con／互換Raw HID workerを起動しない
  - XInput workerを起動しない
  - 遅れて届くコントローラーイベントを無視する
  - コントローラー専用sectionとHold設定をDOMから削除する
  - コントローラー入力を含むルールをactive rulesから外す
  - 保存済みrule／profile／Joy-Con設定は削除しない
  - コントローラー由来の長押し・Hold・DOWN状態を解放する
  - マウス／キーボードのDOWN・長押し・ルールは維持する
- ON時:
  - Joy-Con／互換HIDとXInput subsystemを開始する
  - 保存済みコントローラールールを再びactive rulesへ含める

UIの`機能を停止して設定画面から隠す`はVisibleとEnabledを同時にfalseへ変更する。隠した後は一般管理欄の小さな`実験機能を表示`だけを残す。BetterJoy／JoyToKeyへ外注する場合は非表示・OFFのまま使う。外部ツールが生成するキーボード／マウス入力は通常経路で処理できる。

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
- Switch互換Raw HID（rc4でBetterJoy方式へ修正）:
  - SetupAPIでWindowsが公開する全HID interfaceを列挙する。Usageによる事前除外はしない
  - unknown HIDは自動接続せず、UIで明示選択した正確なpath fingerprintだけを左Joy-Con互換として登録
  - 登録にはVID/PID/serial/fingerprint/productを保存し、再接続時はfingerprint優先・serial fallbackで照合
  - 選択済みinterfaceだけをOpenする。書込み可能ならNintendo `0x03` report-mode commandを試し、Open/Write拒否時だけread-onlyへ縮退
  - `0x21`、`0x30`、`0x3f`、7バイト／先頭0付き8バイトinput-only報告
  - metadata取得失敗interfaceもpath fingerprint、path由来VID/PID、inspect error付きで候補へ残す
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
- rc3でコントローラー機能をdefault OFFへ移動したが、専用の大きな有効化sectionが常時残り、「設定画面から隠す」という要求を満たしていなかった。
- rc4でVisibleとEnabledを分離し、専用UIをDOMから完全除去できるよう修正した。
- rc4でBetterJoyの`hid_enumerate(0,0)`＋manual 3rd-party registrationを確認し、Usage-filterと自由入力Stable IDを廃止した。
- rc3監査で、XInput長押しのkey生成／validationがJoy-Conだけに限定されていた不具合を修正した。
- コントローラーOFF処理がマウス／キーボード長押しまで全消去しないことを回帰試験で固定した。

## 5. 先行実装として確認したBetterJoyコード

rc4では一般論ではなく、`Davidobot/BetterJoy`の次の実コードを確認した。

- `BetterJoyForCemu/Program.cs`: `hid_enumerate(0x0, 0x0)`で全HIDを列挙し、custom controllerをVID/PID/serialで照合してexact pathを`hid_open_path`する
- `BetterJoyForCemu/3rdPartyControllers.cs`: 全HID一覧からユーザーが対象を選び、Pro／Left Joycon／Right Joyconのtypeを明示指定して保存する
- `BetterJoyForCemu/Joycon.cs`: Nintendo output report長49、Attach時のreport-mode初期化とthird-party controller分岐

MouseButtonMapperはBetterJoyのコードをコピーしていない。互換品の入口設計として、全HID列挙・明示登録・exact path Openという方式を再実装した。現段階ではLeft Joy-Con互換だけを登録対象とする。

## 6. 並行処理の不変条件

- `WH_MOUSE_LL`／`WH_KEYBOARD_LL` callbackでSleep、HID I/O、設定保存、ファイルlog、`SendInput`を行わない。
- 出力は`actionCh`へ渡し、output workerが実行する。
- Joy-Con I/Oは`JoyConWorker`、XInputは`XInputWorker`で行う。
- OFFへ切り替える前にconfig gateを下げ、遅延callbackを入口で捨てる。
- HID `WriteFile`とhandle closeは同じmutexで直列化する。
- 切断／停止時はDOWN中トークンへ合成UPを出し、Hold keyを残さない。
- 物理的に押されているキーを合成UPで解放しない。
- コントローラー失敗は正常な縮退状態であり、マウス／キーボード機能を止めない。

## 7. 設定互換性

```text
Config.Version = 11
Config.Controller.Visible = false  // default
Config.Controller.Enabled = false  // default
Profile.JoyCon                    // optional, preserved while global OFF
Rule.Input Kind                   // Mouse / Key / JoyCon / XInput
```

全体OFFは設定削除ではなくruntime filteringで実現する。ON/OFFのたびにrule、profile、calibration、manual HID registrationを消してはいけない。

## 8. 主要ファイル

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
- `joycon_ui_windows.go`: true-hide UI、generic restore control、全HID manual registration list
- `joycon_web_windows.go`: controller/Joy-Con API state
- `controller_feature_windows_test.go`: default OFF、worker停止、rule保存、late event、長押し分離
- `docs/REGRESSION_CHECKLIST.md`: 実機検証正本

## 9. 必須自動検証

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

## 10. 実機ゲート

### 全体OFF

- 起動直後にcontroller専用sectionが存在せず、一般管理欄の小さな実験機能restoreだけが見える
- HID列挙／XInput監視が走らない
- mouse/key rule、long press、profile switchが動く
- BetterJoy／JoyToKeyのkeyboard/mouse出力を通常入力として扱える
- ON→Hold中→OFFでkeyが残留しない
- OFF→ONで保存済みcontroller ruleが復帰する

### ON時

- 純正Joy-Conの接続、全button、stick、battery、sleep/resume
- 全HID interface一覧、対象のmanual registration、report診断、全input
- XInput P1～P4のbuttons/triggers/sticks
- disconnect/reconnect 20回
- controller＋right-hand mouse/keyboard
- game側との二重入力
- 1時間連続input、8時間常駐

完了までは安定版と呼ばず、PRをReady／Mergeしない。

## 11. 禁止事項

- controller featureを暗黙にONへ移行しない。
- OFF時に保存済みcontroller設定を削除しない。
- OFF時にmouse/key stateやlong pressを巻き添えで消さない。
- 未知HIDを自動選択・自動Openしない。Nintendo subcommandはユーザーが明示登録したinterfaceにだけ送る。
- 全HIDを候補表示するため、UIに誤選択警告を必ず維持する。
- hook callbackへHID/XInput処理を入れない。
- 実機未検証を「完全対応」「安定版」と断定しない。
- PR #5を実機確認前にmergeしない。

## 12. 次の担当者が最初に行うこと

1. PR #5本文で最終head／CI／artifactを確認する。
2. `Controller.Enabled=false`のruntime pathを先に回帰確認する。
3. `docs/REGRESSION_CHECKLIST.md`の全体ゲートから実機試験を始める。
4. 互換品が未対応なら、登録されたVID/PID/serial/fingerprint、report length、先頭hexを基にparser testを先に追加する。
5. 全hardware gate完了後だけ8.4.0 stable化を検討する。
