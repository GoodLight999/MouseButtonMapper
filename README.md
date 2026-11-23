# MouseButtonMapper

Windows向けのGUI-first入力リマッパーです。AutoHotkeyの全機能を追うのではなく、マウス／キーボードの割り当てを少ない操作で確実に作れることを中核にします。プラグイン形式で対応マウスのオンボード設定（**Ray Pawn+ / RM-3360LR 等の SinoWealth 系マウス**）にも対応し、必要な場合だけJoy-Con（L）、Switch互換Raw HID、XInputコントローラーを実験機能として追加できます。

## Ray Pawn+ (RM-3360LR) の公式ソフトが入手できず困っている方へ

ゲーミングマウス **Ray Pawn+（RM-3360LR / SINOWEALTH Wired Gaming Mouse / VID 258A / PID 1007 / FW GM07 / PMW3360）** は、公式クライアントの配布サーバーが閉鎖されてダウンロードできなくなっています。

MouseButtonMapper には実機 HID 解析に基づく **Ray Pawn+ / SinoWealth 専用プラグイン** を同梱しており、**公式ソフトなしで** 以下を安全に設定できます。

- 全 9 ボタンのオンボード設定書き換え（右サイドの `PageUp/PageDown` 変更・`DPI±` のカスタムボタン化）
- Windows 11 の隠しショートカット（`Win + F16` シャットダウン等）と衝突しない **`F13 / F18 / F19 / F20` ワンクリック最適化**
- DPI 設定（100〜12,000 DPI・不要スロット削除）・ポーリングレート（125〜1000 Hz）・LED 単色 / 消灯 / 色変更
- 起動中アプリ・ゲームに応じた **プロファイル自動切替 ＋ マウス本体の LED 色・DPI 連動**
- 初回読み込み時の **自動バックアップ** と **ワンクリック工場出荷復元**

詳しい手順・対応確認・USB HID プロトコル仕様は [**Ray Pawn+ (RM-3360LR) 非公式設定ガイド (`docs/RAY_PAWN_PLUS_GUIDE.md`)**](docs/RAY_PAWN_PLUS_GUIDE.md) をご覧ください。

## 現在の配布候補

- バージョン: **8.5.0-rc1**（`8.4.0-rc5` ベース）
- 対象: Windows x64
- 安定版: `main`の8.3.0
- 開発版: プラグイン機能（Ray Pawn+ / SinoWealth 対応）・GUI 刷新・Hold モード・論理サイドボタン認識を含む RC
- 設定保存先: `%LOCALAPPDATA%\MouseButtonMapper\config.json`

## 重要: コントローラー機能は初期状態で非表示・OFF

ゲームコントローラー対応は実験機能です。新規設定および旧設定からの移行では、専用UI自体を表示せず、workerも起動しません。再表示する場合だけ「動作状態と管理」にある小さな`実験機能を表示`を押します。

表示後は`機能を停止して設定画面から隠す`で、入力処理を停止し、専用設定を再び画面から消せます。

OFFの間は次の動作になります。

- Joy-Con／互換HIDの列挙・接続を行わない
- XInput DLLのポーリングを行わない
- コントローラー由来の入力イベントを無視する
- コントローラー専用セクション、詳細設定、Hold設定をDOMから除去する
- 保存済みコントローラールールは削除せず、実行対象からだけ外す
- マウス／キーボード機能、長押し、プロファイル、自動切替は通常どおり動作する

BetterJoyやJoyToKeyへコントローラー処理を任せる場合は、全体スイッチをOFFのまま使用してください。それらからキーボード／マウス入力を出せば、MouseButtonMapperは通常入力として扱えます。

## 主な機能

- 多ボタンマウス、ホイール、キーボードの割り当て（認識済みマウスの `左サイド1/2`・`右サイド1/2`・`DPI+ボタン`・`DPI-ボタン` 表示対応）
- 出力先としてキー、左／右／中クリック、サイド1／2、ホイール上下を同じルールで使用
- 出力の物理記録と、GUIのクイック追加ボタンによるマウス操作登録
- 短押し／長押し分岐、長押し時の別操作、短押しキャンセル、**押し続け連動（Hold）モード**
- 入力／実行内容の記録と、全入力UP時の自動確定
- 複数プロファイル、前面アプリ連動の自動切替（起動中アプリピッカー・管理者権限 UIPI 警告付き）
- プラグイン機構による対応マウス本体設定（**Ray Pawn+ / RM-3360LR / SinoWealth 258A:1007** のオンボードボタン・DPI・ポーリングレート・LED 色・プロファイル連動）
- 低レベルフック専用スレッド、出力ワーカー、フック監視
- 多重起動拒否
- 緊急停止: `Ctrl + Alt + Shift + F12`
- 任意機能として純正Joy-Con（L）、手動選択したSwitch互換Raw HID、XInput P1～P4
- コントローラー＋マウス／キーボード複合入力、短押し、長押し、Hold

## UX方針

通常画面はマウス／キーボードの割り当てへ集中し、コントローラー詳細は必要な人だけが展開するprogressive disclosureとします。入力／実行内容は「記録 → 必要なら直接編集 → テスト → 保存」の順で完結し、実行可能なマウス操作は文字列を暗記せずGUIから追加できることを不変条件にします。

UI改修時は次の設計資料を参照します。

- https://impeccable.style/
- https://www.tasteskill.dev/
- https://github.com/emilkowalski/skills

方針として、カードの入れ子や装飾過多を避け、情報階層、即時フィードバック、必要時だけ詳細を見せることを優先します。

## コントローラーを試す場合

1. 「動作状態と管理」の`実験機能を表示`を押し、コントローラー入力をONにします。
2. 純正Joy-Conは通常ペアリング後、`HID一覧を更新・再接続`を押します。
3. 互換品はSteamを完全終了し、Windowsが公開する全HIDインターフェース一覧から対象を1つ選び、左Joy-Con互換として保存します。
4. 一覧はキーボードやマウス等も含み得るため、対象の製品名・VID/PID・serialを確認して明示選択してください。未知HIDは自動接続しません。
5. XInput機器はP1～P4として自動検出されます。
6. `ゲームコントローラー入力を記録`から割り当てます。

互換品登録はBetterJoyの3rd-party controller方式を参考に、VID/PID/serialと正確なHID interface fingerprintを保存します。書込み可能ならNintendo初期化を試し、拒否された場合だけread-onlyへ縮退します。未対応reportは長さと先頭hexを診断へ残します。コントローラー入力はゲーム側から隠さないため、二重入力が発生する場合は全体スイッチをOFFにし、BetterJoy／JoyToKey／HidHide等の外部構成を検討してください。

## CI成果物

Draft PRと作業ブランチのCIは、同一headから次を生成します。

- `MouseButtonMapper-v8.4.0-rc5.exe`
- `MouseButtonMapper-v8.4.0-rc5-windows-x64.zip`
- `MouseButtonMapper-v8.4.0-rc5-source.zip`
- `MouseButtonMapper-v8.4.0-rc5-SHA256SUMS.txt`

## ローカル検証

Go 1.23以上が必要です。

```powershell
$env:CGO_ENABLED = "0"
go test ./...
go vet -unsafeptr=false ./...
go build -trimpath -ldflags="-s -w -H=windowsgui" -o MouseButtonMapper.exe .
```

## 開発前に読む資料

1. [`docs/HANDOFF.md`](docs/HANDOFF.md)
2. [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
3. [`docs/REGRESSION_CHECKLIST.md`](docs/REGRESSION_CHECKLIST.md)
4. [`docs/RELEASE.md`](docs/RELEASE.md)
5. [`CONTRIBUTING.md`](CONTRIBUTING.md)

## ライセンス

現時点ではオープンソースライセンスを付与していません。詳細は[`LICENSE`](LICENSE)を参照してください。
