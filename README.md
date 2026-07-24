# MouseButtonMapper

Windows向けの多ボタンマウス割り当てツールです。マウスボタン・ホイール・修飾キーの組み合わせを、任意のキー操作へ変換します。

## 現在の安定版

- バージョン: **8.3.0**
- 対象: Windows x64
- 配布形態: ポータブルZIP（インストーラーなし）
- 設定保存先: `%LOCALAPPDATA%\MouseButtonMapper\config.json`
- 自動起動: スタートアップフォルダーへショートカットを置く方式

## 主な機能

- 多ボタンマウス、ホイール、キーボード修飾キーを組み合わせた割り当て
- 入力と実行内容の記録（すべての同時押しを離すと自動確定）
- 短押し／長押しの分岐、長押し時の別操作、長押しによる短押しキャンセル
- 複数プロファイル
- 前面アプリ・ゲームに応じたプロファイル自動切替
- プロセス名、ウィンドウタイトル、実行ファイルパスによる条件指定
- 低レベルフック専用スレッド、出力ワーカー、自動再フック監視
- 多重起動拒否
- 緊急停止: `Ctrl + Alt + Shift + F12`

## ダウンロード

`main`へのpushとPull RequestでGitHub ActionsがWindows x64版をビルドし、Actionsの実行結果に `MouseButtonMapper-windows-x64` という成果物を保存します。

タグ `v*` をpushすると、同じポータブルZIPとSHA-256チェックサムをGitHub Releasesへ登録します。

## ローカルビルド

Go 1.23以上が必要です。

```powershell
$env:CGO_ENABLED = "0"
go test ./...
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

別スレッド・別開発者への引き継ぎは、最初に以下を読めば開始できます。

1. [`docs/HANDOFF.md`](docs/HANDOFF.md) — 完全引き継ぎ資料
2. [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — 構造と重要な不変条件
3. [`docs/REGRESSION_CHECKLIST.md`](docs/REGRESSION_CHECKLIST.md) — 破壊防止チェック
4. [`CONTRIBUTING.md`](CONTRIBUTING.md) — 変更手順

## Tauriについて

安定版へのTauri導入は現時点では見送っています。理由と将来の安全な移行条件は [`docs/TAURI_ASSESSMENT.md`](docs/TAURI_ASSESSMENT.md) に記載しています。

## ライセンス

公開リポジトリですが、現時点ではオープンソースライセンスを付与していません。詳細は [`LICENSE`](LICENSE) を参照してください。
