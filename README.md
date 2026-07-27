# MouseButtonMapper

Windows向けの多ボタンマウス／ゲームコントローラー割り当てツールです。マウス、キーボード、Joy-Con（L）、Switch互換Raw HID、XInputコントローラーの入力や組み合わせを、キー・マウスボタン・ホイール操作へ変換します。

## 現在の配布候補

- バージョン: **8.4.0-rc2**
- 状態: Joy-Con（L）・互換Raw HID・XInput実機検証用Release Candidate
- 対象: Windows x64
- 配布形態: 単体EXE、ポータブルZIP、ソースZIP、SHA-256一覧
- 設定保存先: `%LOCALAPPDATA%\MouseButtonMapper\config.json`
- 自動起動: スタートアップフォルダーへショートカットを置く方式

`main`の安定版は8.3.0です。8.4.0-rc2はDraft PR #5上の候補であり、純正Joy-Con、互換品、XInput機器の接続・切断・長時間試験が終わるまで安定版へマージしません。

## 主な機能

- 多ボタンマウス、ホイール、キーボード、Joy-Con（L）、Switch互換Raw HID、XInputの割り当て
- Joy-Con（L）の上下左右、L、ZL、SL、SR、－、キャプチャー、スティック押込み
- 左スティックの4方向／8方向、デッドゾーン、ヒステリシス、反転、キャリブレーション
- Joy-Con／XInput＋右手マウス、コントローラー＋キーボードの複合入力
- キー、複数キー同時押し、マウスボタン、ホイールへの変換
- 短押し／長押しの分岐、長押し時の別操作、長押しによる短押しキャンセル
- Joy-Con／XInputボタンを押している間だけキーを保持するHoldモード
- 入力と実行内容の記録（すべての同時押しを離すと自動確定）
- 複数プロファイルと前面アプリ／ゲーム連動の自動切替
- 低レベルフック専用スレッド、出力ワーカー、自動再フック監視
- 多重起動拒否
- 緊急停止: `Ctrl + Alt + Shift + F12`

## ゲームコントローラーの使い方

### 純正Joy-Con（L）

1. WindowsのBluetooth設定でJoy-Con（L）を通常ペアリングします。
2. MouseButtonMapperを起動し、設定画面のゲームコントローラー欄を開きます。
3. Joy-Conを有効にし、`Joy-Conを接続・再検索`を押します。
4. `ゲームコントローラー入力を記録`からボタン／スティック方向を割り当てます。
5. スティックを使う場合はキャリブレーションを実施します。

### Switch互換品

互換品はWindows標準ゲームパッドやJoyToKeyでは入力が見えず、Steam InputだけがRaw HIDを直接解釈できる場合があります。rc2はHID Usage Page/Usageからゲームパッド候補だけを列挙し、`0x30`、`0x3f`、SDL互換のinput-only形式を読み取ります。

自動判定されない場合は、設定画面の`優先するJoy-Con／互換HID識別子`から検出候補を選んで保存します。未対応reportの場合は、エラー欄へreport長と先頭hexが表示されるため、機器固有マッピングの追加に利用できます。Steamがデバイスを占有する場合があるため、初回確認時はSteamを完全終了してください。

### XInput

Xbox系、XInputモードの互換パッド、片手ゲームパッドを最大4台まで自動監視します。A/B/X/Y、LB/RB、LT/RT、十字キー、左右スティック方向、スティック押込み、Start/Backを入力記録できます。

本アプリはBetterJoy、ViGEmBus、HidHideを必須としません。ゲームが物理コントローラー入力を直接受け取る場合は二重入力になり得るため、実ゲームで確認してください。

## ダウンロード

Pull Requestと作業ブランチのCIは、次を一つのActions成果物へ保存します。

- `MouseButtonMapper-v8.4.0-rc2.exe`
- `MouseButtonMapper-v8.4.0-rc2-windows-x64.zip`
- `MouseButtonMapper-v8.4.0-rc2-source.zip`
- `MouseButtonMapper-v8.4.0-rc2-SHA256SUMS.txt`

タグ`v*`をpushすると、Release workflowが同じ形式の成果物をGitHub Releasesへ登録します。

## ローカルビルド

Go 1.23以上が必要です。

```powershell
$env:CGO_ENABLED = "0"
go test ./...
go vet -unsafeptr=false ./...
go build -trimpath -ldflags="-s -w -H=windowsgui" -o MouseButtonMapper.exe .
.\MouseButtonMapper.exe --self-test
```

LinuxまたはmacOSからWindows版をクロスコンパイルする場合:

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags='-s -w -H=windowsgui' \
  -o MouseButtonMapper.exe .
```

## 開発前に読む資料

1. [`docs/HANDOFF.md`](docs/HANDOFF.md) — 正本・現在地・禁止事項
2. [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — 構造と不変条件
3. [`docs/REGRESSION_CHECKLIST.md`](docs/REGRESSION_CHECKLIST.md) — Windows実機確認
4. [`CONTRIBUTING.md`](CONTRIBUTING.md) — 変更手順

## Tauriについて

安定版へのTauri導入は見送っています。理由と将来の安全な移行条件は[`docs/TAURI_ASSESSMENT.md`](docs/TAURI_ASSESSMENT.md)に記載しています。

## ライセンス

公開リポジトリですが、現時点ではオープンソースライセンスを付与していません。詳細は[`LICENSE`](LICENSE)を参照してください。
