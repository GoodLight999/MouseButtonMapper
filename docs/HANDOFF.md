# MouseButtonMapper 完全引き継ぎ資料

この文書は、別スレッドのLLMまたは新しい開発者がリポジトリURLだけを受け取り、安全に作業を再開するための正本です。作業開始時と終了時に必ず更新します。

## 1. 正本とブランチ

- Repository: `https://github.com/GoodLight999/MouseButtonMapper`
- 安定版の唯一の正本: `main`
- `main`基準版: `8.3.0`
- Joy-Con作業ブランチ: `feature/joycon-l`
- Draft PR: `#5`
- RC版: `8.4.0-rc1`
- Joy-Conブランチの起点: `3126d531cff93c519bdf3634847d553033787940`
- 自動試験が全緑になった実装基準commit: `74679f72791b09137fda107d8422851c535ec7b1`

過去スレッドのZIP、ローカル派生版、`trash`／archived資料を正本として参照してはいけません。`main`と明示された作業ブランチだけを使用します。

## 2. 現在の状態

### 完了しているもの

- Joy-Con（L）Windows HID列挙、Open、`ReadFile`、`WriteFile`、切断検知、再検索、再接続
- Nintendo VID `0x057e` / Joy-Con（L）PID `0x2006`による識別
- 入力報告`0x30`、サブコマンド応答`0x21`、限定的な`0x3f`フォールバック
- 上下左右、L、ZL、SL、SR、－、キャプチャー、スティック押込み
- 左スティックの12bit解析、正規化、キャリブレーション、デッドゾーン、ヒステリシス、4方向／8方向、反転
- Joy-Con＋マウス／キーボードの複合ルール
- Joy-Con短押し、長押し、キャンセル、Hold出力
- キー、複数キー同時押し、マウスボタン、ホイール出力
- 入力記録、接続状態、診断、スティック設定、キャリブレーションのWeb UI/API
- プロファイル別optional Joy-Con設定と旧JSON互換
- 切断・再読込・フック再登録・緊急停止・終了時の状態解放
- `CancelIoEx`によるブロッキングHID読取解除
- 物理キーとJoy-Con出力の競合保護
- ルール編集だけではJoy-Conを再接続しない再構築経路
- キーボード終端の複合ルールで前置入力を正しく消費

### 自動検証済み

基準commit `74679f7…`で次が成功しています。

- `gofmt`
- 公開リポジトリ監査
- `go test ./...`
- `go test -race ./...`
- Windows `go test ./...`
- Windows `go vet -unsafeptr=false ./...`
- Windows GUI EXEビルド
- portable ZIP生成

### 未完了・マージ禁止理由

実物のJoy-Con（L）を使うWindows実機試験は未完了です。以下を確認するまでPR #5をReady、Merge、安定版Releaseにしてはいけません。

- Bluetoothペアリングと入力報告モード`0x30`への切替
- 全物理ボタンとスティック方向の実報告照合
- バッテリー表示
- 切断・再接続20回、電源断、Bluetooth切断
- スリープ復帰
- Joy-Con＋右手マウス／キーボード複合操作
- 実ゲームでの二重入力有無
- 高負荷ゲーム、Redragon等の高イベント量マウス
- 1時間連続入力
- 8時間常駐

## 3. プロジェクトの不変条件

優先順位は、常駐安定性、入力を壊さないこと、設定互換性、GUIの明確さ、機能追加の順です。

- EXE単体で任意フォルダーから起動できること。
- 自動起動はスタートアップショートカット方式を維持すること。
- 多重起動を拒否し、二度目の起動は既存GUIを表示すること。
- 設定保存先は`%LOCALAPPDATA%\MouseButtonMapper\config.json`。
- 旧設定を読み込み、保存しても既存ルール、プロファイル、自動切替条件を失わないこと。
- 入力記録／出力記録は、全同時押しを離した時点で自動確定すること。
- チェック欄は本物の操作可能なチェックボックスであること。
- 保存ボタンは対象を文言で明示し、単独の「保存」「適用」を増やさないこと。
- Base profile、Editor profile、Effective profileを混同しないこと。
- 右クリック、ドラッグ、通常キー入力を割り当て外で失わせないこと。
- 緊急停止`Ctrl+Alt+Shift+F12`を常に残すこと。

## 4. 入力・並行処理の安全境界

- `WH_MOUSE_LL`と`WH_KEYBOARD_LL`は専用OSスレッドに置きます。
- フックコールバック内でSleep、HID I/O、ログファイル書込、設定保存、長時間ロック待ち、`SendInput`を行いません。
- 出力は`actionCh`へ渡し、`outputWorker`が`SendInput`を実行します。
- 自己注入イベントは`extraInfoMarker`で除外します。
- Joy-Con HID I/Oは専用`JoyConWorker`で行います。
- 状態通知は接続・エラー・入力変化時を除き最大20Hzです。
- HID切断時はDOWN中の全Joy-Conトークンへ合成UPを出します。
- 終了時は`CancelIoEx`、ハンドルclose、ワーカー待機の順でブロッキング読取を解除します。
- Joy-Con未接続やHID失敗は正常な劣化状態であり、既存マウス／キーボード機能を止めません。

## 5. 長押しとHold

- 長押し設定: `LongPressEnabled`, `LongPressMs`, `LongPressAction`, `LongPressOutput`。
- `LongPressAction`: `Execute`または`Cancel`。
- タイマーはフック外で動作し、UPとの競合でも一度だけ確定します。
- 長押し中に入力が長い複合ルールの前置入力となった場合、単独長押しを中止します。
- HoldはJoy-Con単入力からキー出力に限定し、押下中だけキーを保持します。
- Hold参照カウントにより複数Joy-Conルールで同じキーを共有できます。
- Joy-Con側のUPで、同じキーの物理押下を勝手に解放してはいけません。

## 6. Joy-Con（L）実装

### デバイスと報告

- VID: `0x057e`
- PID: `0x2006`
- 接続後、サブコマンド`0x03`で報告モード`0x30`を要求します。
- `0x30`と`0x21`を主入力としてボタン、左スティック、バッテリーを解析します。
- `0x3f`はOS互換の限定フォールバックです。初版のアナログ要件や実機保証の主経路にしてはいけません。
- ジャイロ、加速度、振動、LED、仮想Xboxコントローラー化は8.4.0へ混ぜません。
- BetterJoy、ViGEmBus、HidHideを必須依存にしません。

### 共通入力

- 設定互換境界は`Item{Kind, Code}`です。
- Joy-Conは`Item{Kind:"JoyCon", Code:"ZL"}`等で保存します。
- 実行時は`InputToken`とDOWN/UPイベントへ正規化します。
- Joy-Con、マウス、キーボードは同じ押下集合とルール優先順位で評価します。
- キーボードを最後の入力とする複合ルールでも、前置Joy-Con／マウス入力を消費済みにします。

### スティック

- 生値は12bit X/Yです。
- `Min`, `Center`, `Max`から`[-1,1]`へ正規化します。
- `DeadZone > ReleaseZone`を必須とし、境界揺れを防ぎます。
- 8方向はX/Yを独立判定し、斜め二方向を同時DOWNできます。
- 4方向は優勢軸を選び、対角付近に切替余裕を持たせます。
- 半倒し／全倒し二段階は、実機安定後の将来拡張です。

## 7. 設定とGUI

- `Config.Version`は9。
- Joy-Con設定は各`Profile`のoptionalフィールドです。
- Effective profileの変更時にJoy-Con設定も同時に切り替えます。
- ルールの保存・追加・削除・並べ替えだけではJoy-Conを再接続しません。
- Joy-Con設定またはEffective profileが変わる場合だけ再検索します。
- GUI文言は対象を明示します。
  - `Joy-Conを接続・再検索`
  - `Joy-Con入力を記録`
  - `選択中の割り当てへJoy-Con入力を設定`
  - `スティック設定を保存`
  - `キャリブレーションを開始`
  - `選択中の割り当てを保存`

## 8. 主要ファイル

- `main.go`: Windows本体、フック、設定、HTTP API、トレイ、ルール評価
- `longpress_windows.go`: 長押し統合
- `joycon_logic.go`: OS非依存報告解析、スティック、状態差分
- `joycon_device.go`: デバイス／設定／状態型
- `joycon_hid_windows.go`: SetupAPI、HidD、CreateFile、ReadFile、WriteFile
- `joycon_worker.go`: 列挙、接続、読取、切断、再接続
- `joycon_app_windows.go`: App、ルール、記録、長押し統合
- `joycon_output_windows.go`: Tap/Hold、キー／マウス／ホイール出力
- `joycon_web_windows.go`: API
- `joycon_ui_windows.go`: Joy-Con UI JavaScript
- `docs/REGRESSION_CHECKLIST.md`: 実機試験の正本

## 9. 作業手順

1. `main`と対象作業ブランチの先端を確認します。
2. 変更前に回帰試験を追加します。
3. 入力コアとGUIを同時に大改造しません。
4. `gofmt`, Linux test/race, Windows test/vet/build/packageを実行します。
5. Windows実機チェックリストを実施します。
6. 未確認事項をPRと本資料へ残します。
7. 実機確認前にPRをマージ、安定版タグ付けしません。

## 10. 禁止事項

- 過去の破綻版・添付ZIP・archived/trash資料を正本にしない。
- フックコールバックをSleepさせない。
- HID I/Oをフックへ入れない。
- タイマーとUPの双方から長押しを二重実行しない。
- 物理入力をJoy-Con出力のUPで破壊しない。
- 設定再読込やフック再登録を跨いでHoldキーを残留させない。
- ルール編集のたびにJoy-Conを切断・再接続しない。
- 実機未検証の状態を「完全修正」「安定版」と断定しない。
- PR #5をユーザーの実機確認前にマージしない。

## 11. 次の作業

1. v8.4.0-rc1成果物をWindows CIで生成する。
2. ユーザーへEXE、portable ZIP、source ZIP、SHA-256を渡す。
3. `docs/REGRESSION_CHECKLIST.md`に沿って実機結果を記録する。
4. 実機不具合を同じブランチで修正し、RC番号を更新する。
5. 全必須項目完了後だけ`8.4.0`へ版更新し、PRをReadyにしてマージする。
