# Architecture

## 実行時データフロー

```text
物理マウス/キーボード
        │
        ▼
低レベルフック専用スレッド
        │  状態照合・抑制判定
        ├──────────────► CallNextHookEx
        │
        ▼
actionCh（短い非同期受け渡し）
        │
        ▼
outputWorker
        │
        ▼
SendInput（dwExtraInfoマーカー付き）
```

フックコールバックを軽く保つことが安定性の中心です。

## 並行処理

- Hook thread: Windowsメッセージループとフック登録
- Output worker: 変換後入力の送信
- Config writer: JSONの安全な保存
- Hook watchdog: 入力進行とフック活動を比較し再登録
- Foreground monitor: WinEventHookと定期確認
- HTTP server: localhost GUI API

## プロファイル状態

- Base profile: 通常時に使うプロファイル
- Editor profile: GUIで編集しているプロファイル
- Effective profile: 現在入力変換へ適用中のプロファイル

これらを単一変数へ統合してはいけません。

## 公開境界

HTTPサーバーは`127.0.0.1`のランダムポートへbindします。外部インターフェースへbindしてはいけません。

## 設定互換性

`Config.Version`は現在8です。新規フィールドは原則optionalとして追加し、古いJSONを読み込めるようにします。削除や名称変更が必要な場合は明示的な移行処理を入れます。
