# Contributing

## 絶対条件

- `main`へ直接変更を入れない。
- 変更はブランチとPull Requestで行う。
- ユーザーの既存`config.json`を削除・上書き・互換性破壊しない。
- 入力フックのコールバック内へ重い処理、ファイルI/O、ネットワークI/O、待機処理を追加しない。
- GUIだけを修正したつもりで入力コアを変更しない。逆も同様。
- 実機確認前に「確実に動く」と断定しない。

## 必須検査

```bash
gofmt -w *.go
go test ./...
go test -race ./...
GOOS=windows GOARCH=amd64 go vet -unsafeptr=false ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w -H=windowsgui' -o MouseButtonMapper.exe .
```

Windowsではさらに:

```powershell
.\MouseButtonMapper.exe --self-test
```

実機回帰項目は [`docs/REGRESSION_CHECKLIST.md`](docs/REGRESSION_CHECKLIST.md) をすべて確認してください。

## Pull Request

PR本文へ以下を記載します。

- 変更の目的
- 入力コアへ触れたか
- 設定スキーマへ触れたか
- 実施した自動テスト
- 実施したWindows実機試験
- 未確認事項
