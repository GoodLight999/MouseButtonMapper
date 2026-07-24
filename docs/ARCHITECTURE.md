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
- Long-press timers: `time.AfterFunc`でしきい値を監視し、結果だけを既存`actionCh`へ渡す

## プロファイル状態

- Base profile: 通常時に使うプロファイル
- Editor profile: GUIで編集しているプロファイル
- Effective profile: 現在入力変換へ適用中のプロファイル

これらを単一変数へ統合してはいけません。

## 公開境界

HTTPサーバーは`127.0.0.1`のランダムポートへbindします。外部インターフェースへbindしてはいけません。

## 設定互換性

`Config.Version`は現在9です。新規フィールドは原則optionalとして追加し、古いJSONを読み込めるようにします。削除や名称変更が必要な場合は明示的な移行処理を入れます。


## 長押し状態機械

- 長押し判定は、ルールの最後の入力を押した時点で開始します。
- フックコールバック内ではタイマー待機を行いません。`time.AfterFunc`のコールバックも`SendInput`を直接呼ばず、既存の出力キューへ渡します。
- しきい値より前に離した場合は短押し出力、しきい値以後は長押し出力またはキャンセルを一度だけ確定します。
- タイマー発火とUPイベントが競合しても、トークンと状態フラグにより二重実行しません。
- 終了開始時は`shuttingDown`を立て、`shutdownCh`で出力ワーカーを止めます。`actionCh`自体は閉じず、遅れて戻ったタイマーが閉じたチャネルへ送信するpanicを防ぎます。
- 単押し長押し中に、その入力がより長い組み合わせルールの前置入力として使われた場合、単押し側を中止します。
- 長押し判定の対象はX1/X2または修飾キー以外のキーボードキーです。左・右・中クリックは通常操作を壊さないため対象外です。
