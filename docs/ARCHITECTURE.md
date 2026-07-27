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
- Joy-Con／互換HID worker: HID列挙、候補判定、接続、入力報告読取、切断検知、再接続
- XInput worker: DLLを動的ロードし、最大4スロットの状態差分をポーリング
- Controller event adapter: Joy-Con／XInputをOS非依存イベントへ正規化し、共通押下状態へ短く受け渡す

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

## ゲームコントローラー入力層

### データフロー

```text
Windows Bluetooth/USB HID        XInput
        │                           │
        ▼                           ▼
HID列挙・接続ワーカー       XInputポーリングワーカー
        │  ReadFile / 切断検知 / 再列挙
        ▼
Joy-Con／Switch互換report parser（OS非依存）       XInput state tracker
        │  ボタン差分・12bitスティック・バッテリー       │  ボタン・trigger・stick差分
        └──────────────────────┬──────────────────────┘
                               ▼
Controller state normalizer（OS非依存）
        │  重複DOWN除去 / 欠落UP補正 / ヒステリシス
        ▼
InputEvent{Token, Down, Source, Time}
        │
        ├──────────────► GUI診断・入力記録
        │
        ▼
共通押下状態・ルール評価
        │
        ▼
既存actionCh → outputWorker → SendInput
```

HID I/O、再接続待機、キャリブレーション集計、GUI更新を低レベルフックコールバックへ入れてはいけません。

### デバイス識別

- 純正Nintendo vendor ID: `0x057e`、Joy-Con（L）product ID: `0x2006`。
- HID capsのUsage Page `0x01`かつUsage `0x04`（Joystick）／`0x05`（Game Pad）だけを互換候補にする。
- 主要識別子はVID/PID、製品名、デバイスパスfingerprint、シリアル文字列、report長。
- 純正または明示的なJoy-Con（L）名だけを自動選択し、未知VID/PIDはStable IDの手動選択を要求する。
- 手動選択された未知機器は`ForcedCompatible`としてread-onlyから開始し、無差別なNintendoサブコマンド送信を避ける。

### HID報告

- `0x30`: 標準フル入力報告。ボタン、バッテリー、左スティックを取得する主経路。
- `0x21`: サブコマンド応答。標準入力状態を含むため、入力差分にも利用できる。
- `0x3f`: Joy-Con簡易報告。物理ボタンとstick hatを別に解析する。
- 7バイトinput-only: SDLのSwitch HID実装と同じbuttons/hat/8bit stick配置を互換品のread-only経路で解析する。
- 純正／書込み対応機器だけサブコマンド`0x03`で入力報告モード`0x30`を要求する。
- 未対応reportは長さと先頭hexを診断へ残し、黙って誤解釈しない。
- 初版ではIMUデータを解析・公開しない。

### 共通入力トークン

```text
mouse:X1
mouse:WheelUp
key:17
joycon:left:ZL
joycon:left:StickUp
xinput:P1:LB
xinput:P1:LStickUp
joycon:left:StickUpFull   （将来拡張）
```

- トークン比較はKindとCodeの正規化後に行う。
- `Item{Kind:"JoyCon", Code:"ZL"}`と`Item{Kind:"XInput", Code:"P1:LB"}`を設定JSONの互換境界とする。
- Joy-Con／XInput＋マウス／キーボードの複合入力は、デバイス別の別ルールエンジンではなく同じ押下集合で評価する。
- 既存のマウス・キーボード規則を一括変換せず、アダプターを介して段階的に共通化する。

### 押下状態の不変条件

- 同一トークンの重複DOWNは新しい押下として扱わない。
- UPは一度だけ状態を解除する。
- HID／XInput切断時は、そのデバイス由来でDOWN中の全トークンへ合成UPを発生させる。
- プロファイル切替は既存仕様どおり全入力解放まで保留する。
- 緊急停止、設定再読込、終了開始時は全コントローラーの保留長押しと押下状態を破棄する。
- 再接続後の最初の報告を基準状態として扱い、切断前の押下状態を引き継がない。

### スティック正規化

生値は12bitのX/Yとして解析し、キャリブレーション値`Min`, `Center`, `Max`から軸ごとに正規化する。

```text
raw < center:  (raw - center) / (center - min)
raw > center:  (raw - center) / (max - center)
```

- 出力は`[-1, 1]`へclampする。
- X/Y反転は正規化後に適用する。
- 中心ドリフト補正は保存済み中心値を更新する明示的キャリブレーションで行う。
- 方向DOWNしきい値とUPしきい値を分離し、`press > release`を必須とする。
- 8方向ではX/Yを独立判定して斜め二方向を同時にDOWNできる。
- 4方向では絶対値の大きい軸だけを採用し、軸優勢が入れ替わる境界で不要な連打を起こさない。

### 設定配置

Joy-Con設定は各`Profile`へoptionalで追加する。

```text
Profile
 ├─ Rules
 └─ JoyCon
     ├─ Enabled
     ├─ PreferredDevice
     ├─ Stick
     │   ├─ DeadZone / ReleaseZone
     │   ├─ DirectionMode
     │   ├─ InvertX / InvertY
     │   └─ Calibration
     └─ Reconnect
```

Effective profileが変わった時点でJoy-Con設定も同じプロファイルから取得する。自動切替機構を複製してはいけません。

### 障害分離

- Joy-Con／XInput未接続は正常状態であり、アプリ起動失敗にしない。
- HID DLL/API、列挙、Open、Read、Writeの失敗は診断へ記録し、既存入力コアは継続する。
- HIDワーカー停止を待つためのキャンセル経路を持ち、ブロッキングReadはデバイスハンドルcloseで解除する。
- 再検索要求はチャネルで合流させ、複数GUI操作で列挙ゴルーチンを多重起動しない。
