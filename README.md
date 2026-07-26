# MouseButtonMapper

Windows向けの多ボタンマウス／Joy-Con（L）割り当てツールです。マウス、キーボード、Joy-Con（L）の入力や組み合わせを、キー・マウスボタン・ホイール操作へ変換します。

## 現在の配布候補

- バージョン: **8.4.0-rc1**
- 状態: Joy-Con（L）実機検証用Release Candidate
- 対象: Windows x64
- 配布形態: 単体EXE、ポータブルZIP、ソースZIP、SHA-256一覧
- 設定保存先: `%LOCALAPPDATA%\MouseButtonMapper\config.json`
- 自動起動: スタートアップフォルダーへショートカットを置く方式

`main`の安定版は8.3.0です。8.4.0-rc1はDraft PR #5上の候補であり、Joy-Con実機の接続・切断・長時間試験が終わるまで安定版へマージしません。

## 主な機能

- 多ボタンマウス、ホイール、キーボード、Joy-Con（L）の割り当て
- Joy-Con（L）の上下左右、L、ZL、SL、SR、－、キャプチャー、スティック押込み
- 左スティックの4方向／8方向、デッドゾーン、ヒステリシス、反転、キャリブレーション
- Joy-Con＋右手マウス、Joy-Con＋キーボードの複合入力
- キー、複数キー同時押し、マウスボタン、ホイールへの変換
- 短押し／長押しの分岐、長押し時の別操作、長押しによる短押しキャンセル
- Joy-Conボタンを押している間だけキーを保持するHoldモード
- 入力と実行内容の記録（すべての同時押しを離すと自動確定）
- 複数プロファイルと前面アプリ／ゲーム連動の自動切替
- 低レベルフック専用スレッド、出力ワーカー、自動再フック監視
- 多重起動拒否
- 緊急停止: `Ctrl + Alt + Shift + F12`

## Joy-Con（L）の使い方

1. WindowsのBluetooth設定でJoy-Con（L）を通常ペアリングします。
2. MouseButtonMapperを起動し、設定画面のJoy-Con欄を開きます。
3. Joy-Conを有効にし、`Joy-Conを接続・再検索`を押します。
4. `Joy-Con入力を記録`または通常の入力記録からボタン／スティック方向を割り当てます。
5. スティックを使う場合はキャリブレーションを実施し、デッドゾーンと4方向／8方向を保存します。

本アプリはBetterJoy、ViGEmBus、HidHideを必須としません。ゲームが物理Joy-Con入力を直接受け取る場合は二重入力になり得るため、実ゲームで確認してください。

## ダウンロード

Pull Requestと作業ブランチのCIは、次を一つのActions成果物へ保存します。

- `MouseButtonMapper-v8.4.0-rc1.exe`
- `MouseButtonMapper-v8.4.0-rc1-windows-x64.zip`
- `MouseButtonMapper-v8.4.0-rc1-source.zip`
- `MouseButtonMapper-v8.4.0-rc1-SHA256SUMS.txt`

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
