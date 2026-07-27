# MouseButtonMapper 完全引き継ぎ資料

この文書は、別スレッドのLLMまたは新しい開発者がリポジトリURLだけを受け取り、安全に作業を再開するための正本です。作業開始時と終了時に必ず確認し、実装・検証・実機確認の境界を曖昧にしないでください。

## 1. 正本とブランチ

- Repository: `https://github.com/GoodLight999/MouseButtonMapper`
- 安定版の唯一の正本: `main`
- `main`基準版: `8.3.0`
- Joy-Con作業ブランチ: `feature/joycon-l`
- Draft PR: `#5`
- RC版: `8.4.0-rc1`
- Joy-Conブランチの起点: `3126d531cff93c519bdf3634847d553033787940`
- Joy-Con実装の初回全緑基準: `74679f72791b09137fda107d8422851c535ec7b1`
- 初回exact-head RC検証: `858f6348c26a32a509da5fc9b52961e4e19cb685`
- 再検索gate回帰試験追加後の全緑基準: `94cc8c80bc9a27601a5e1400d13ec95b6ecaad3d`
- 現在の製品実装commit: `8b4ec43e55daa319715ece7050d2c538a116e67c`

ブランチ先端は文書更新commitを含むため、配布時の最終head SHAはPR #5本文に記録します。過去スレッドのZIP、ローカル派生版、`trash`／archived資料を正本として参照してはいけません。`main`と`feature/joycon-l`だけを使用します。

## 2. 現在の状態

### 完了しているもの

- Joy-Con（L）のWindows Bluetooth HID列挙、Open、`ReadFile`、`WriteFile`、切断検知、手動再検索、再接続
- Nintendo VID `0x057e` / Joy-Con（L）PID `0x2006`による識別
- 入力報告`0x30`、サブコマンド応答`0x21`、限定的な`0x3f`フォールバック
- 上下左右、L、ZL、SL、SR、－、キャプチャー、スティック押込み
- 左スティック12bit解析、正規化、キャリブレーション、デッドゾーン、ヒステリシス、4方向／8方向、軸反転
- Joy-Con＋マウス／キーボードの複合ルール
- Joy-Con短押し、長押し実行、長押しキャンセル、Hold出力
- キー、複数キー同時押し、マウスボタン、ホイール出力
- 入力記録、接続状態、診断、スティック設定、キャリブレーションのWeb UI/API
- プロファイル別optional Joy-Con設定と旧JSON互換
- 切断・設定再読込・フック再登録・緊急停止・終了時の状態解放
- `CancelIoEx`によるブロッキングHID読取解除
- 物理キーとJoy-Con出力の競合保護
- ルール編集だけではJoy-Conを再接続しない再構築経路
- キーボード終端の複合ルールで前置入力を正しく消費

### 追加済みの安全性修正

- 連続する`RequestRescan()`をatomic pending gateで一回の接続試行へ合流
- 列挙・Open・report-mode設定中に来た再検索を、その試行で処理済みとして消化
- 接続完了後の新しい再検索は再度受け付けることを回帰試験で固定
- スティック方向切替時、旧方向のUPを新方向のDOWNより先に送信
- `0x3f`の物理ボタンとスティックhatを別トークンとして解析
- subsystem停止時に`joyConWorker`参照を残さない
- HID `WriteFile`とhandle closeを同じmutexで直列化
- キャリブレーション結果を開始時のプロファイルIDへ保存し、途中のEditor切替へ流出させない
- キャリブレーション対象プロファイルが削除された場合、別プロファイルへ誤保存せずエラーにする
- 同名プロファイルでも安定したEditor indexでJoy-Con UI更新を判定
- 保存ボタンを`Joy-Con接続・スティック設定を保存`とし、対象を明示

### 自動検証履歴

`94cc8c80…`では通常CIの次の全項目が成功しています。

- `gofmt`
- 公開リポジトリ監査
- Linux `go test ./...`
- Linux `go test -race ./...`
- Windows `go test ./...`
- Windows `go vet -unsafeptr=false ./...`
- Windows GUIビルドとRC成果物生成

`8b4ec43e…`の監査修正は、適用workflow内でWindows `go test ./...`、`go vet -unsafeptr=false ./...`、`git diff --check`に成功してからcommitされています。ただし、配布には文書更新を含む最終branch headの通常CIが全緑であることを改めて要求します。

### 成果物

同じ最終headから次の4点を生成します。

- `MouseButtonMapper-v8.4.0-rc1.exe`
- `MouseButtonMapper-v8.4.0-rc1-windows-x64.zip`
- `MouseButtonMapper-v8.4.0-rc1-source.zip`
- `MouseButtonMapper-v8.4.0-rc1-SHA256SUMS.txt`

CIはPull Requestの一時merge refではなく、`github.event.pull_request.head.sha`を明示checkoutします。配布前に必ず次を確認します。

- SHA-256一覧と独立再計算が一致
- portable ZIP内部EXEと単体EXEがバイト一致
- source ZIPのcommit commentが最終PR head SHAと一致
- PR #5本文へ最終head、CI run、4成果物のSHA-256を記録

### 未完了・マージ禁止理由

実物のJoy-Con（L）を使うWindows実機試験は未完了です。以下を確認するまでPR #5をReady、Merge、安定版Releaseにしてはいけません。

- Bluetoothペアリングと入力報告モード`0x30`への切替
- 全物理ボタンと実際のスティック軸・方向の照合
- バッテリー表示
- 電源断、Bluetooth切断、切断・再接続20回
- スリープ復帰
- Joy-Con＋右手マウス／キーボード複合操作
- 実ゲームでの二重入力有無
- 高負荷ゲームと高イベント量マウスの共存
- 1時間連続入力
- 8時間常駐

RCは実機試験用として引き渡せますが、安定版とは表現しません。

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

- `WH_MOUSE_LL`と`WH_KEYBOARD_LL`は専用OSスレッドに置く。
- フックコールバック内でSleep、HID I/O、ログファイル書込、設定保存、長時間ロック待ち、`SendInput`を行わない。
- 出力は`actionCh`へ渡し、`outputWorker`が`SendInput`を実行する。
- 自己注入イベントは`extraInfoMarker`で除外する。
- Joy-Con HID I/Oは専用`JoyConWorker`で行う。
- 状態通知は接続・エラー・入力変化時を除き最大20Hz。
- HID切断時はDOWN中の全Joy-Conトークンへ合成UPを出す。
- 終了時は`CancelIoEx`、handle close、worker待機でブロッキング読取を解除する。
- `WriteReport`とhandle closeは`writeMu`で直列化する。
- Joy-Con未接続やHID失敗は正常な劣化状態であり、既存マウス／キーボード機能を止めない。

## 5. 長押しとHold

- 長押し設定: `LongPressEnabled`, `LongPressMs`, `LongPressAction`, `LongPressOutput`。
- `LongPressAction`: `Execute`または`Cancel`。
- タイマーはフック外で動作し、UPとの競合でも一度だけ確定する。
- 長押し中に入力が長い複合ルールの前置入力となった場合、単独長押しを中止する。
- HoldはJoy-Con単入力からキー出力に限定し、押下中だけキーを保持する。
- Hold参照カウントにより複数Joy-Conルールで同じキーを共有できる。
- Joy-Con側のUPで、同じキーの物理押下を勝手に解放しない。

## 6. Joy-Con（L）実装

### デバイスと報告

- VID: `0x057e`
- PID: `0x2006`
- 接続後、サブコマンド`0x03`で報告モード`0x30`を要求する。
- `0x30`と`0x21`を主入力としてボタン、左スティック、バッテリーを解析する。
- `0x3f`はOS互換の限定フォールバック。bytes 1–2の物理ボタンとbyte 3のスティックhatを分離して解析する。
- `0x3f`をアナログ軸や実機保証の主経路にしない。
- ジャイロ、加速度、振動、LED、仮想Xboxコントローラー化は8.4.0へ混ぜない。
- BetterJoy、ViGEmBus、HidHideを必須依存にしない。

### 共通入力

- 設定互換境界は`Item{Kind, Code}`。
- Joy-Conは`Item{Kind:"JoyCon", Code:"ZL"}`等で保存する。
- 実行時は`InputToken`とDOWN/UPイベントへ正規化する。
- Joy-Con、マウス、キーボードは同じ押下集合とルール優先順位で評価する。
- キーボードを最後の入力とする複合ルールでも、前置Joy-Con／マウス入力を消費済みにする。
- 同一報告内の方向変更はrelease群をpress群より先にemitする。

### スティック

- 生値は12bit X/Y。
- `Min`, `Center`, `Max`から`[-1,1]`へ正規化する。
- `DeadZone > ReleaseZone`を必須とし、境界揺れを防ぐ。
- 8方向はX/Yを独立判定し、斜め二方向を同時DOWNできる。
- 4方向は優勢軸を選び、対角付近に切替余裕を持たせる。
- キャリブレーションは開始時のprofile IDへ拘束する。
- 半倒し／全倒し二段階は実機安定後の将来拡張。

## 7. 設定とGUI

- `Config.Version`は9。
- Joy-Con設定は各`Profile`のoptionalフィールド。
- Effective profile変更時にJoy-Con設定も同時に切り替える。
- ルールの保存・追加・削除・並べ替えだけではJoy-Conを再接続しない。
- Joy-Con設定またはEffective profileが変わる場合だけ再検索する。
- 同名プロファイルを名前だけで識別しない。Editor indexまたはprofile IDを使う。
- GUI文言は対象を明示する。
  - `Joy-Conを接続・再検索`
  - `Joy-Con入力を記録`
  - `選択中の割り当てへJoy-Con入力を設定`
  - `Joy-Con接続・スティック設定を保存`
  - `キャリブレーションを開始`
  - `選択中の割り当てを保存`

## 8. 主要ファイル

- `main.go`: Windows本体、フック、設定、HTTP API、トレイ、ルール評価
- `longpress_windows.go`: 長押し統合
- `joycon_logic.go`: OS非依存報告解析、スティック、状態差分
- `joycon_device.go`: デバイス／設定／状態型
- `joycon_hid_windows.go`: SetupAPI、HidD、CreateFile、ReadFile、WriteFile
- `joycon_worker.go`: 列挙、接続、読取、切断、再検索gate、再接続
- `joycon_app_windows.go`: App、ルール、記録、長押し統合
- `joycon_output_windows.go`: Tap/Hold、キー／マウス／ホイール出力
- `joycon_web_windows.go`: APIとEditor profile状態
- `joycon_ui_windows.go`: Joy-Con UI JavaScript
- `joycon_lifecycle_windows_test.go`: 停止、キャリブレーション保存先、同名profile回帰試験
- `docs/REGRESSION_CHECKLIST.md`: 実機試験の正本

## 9. 作業手順

1. `main`と`feature/joycon-l`の先端を確認する。
2. 変更前に再現試験または回帰試験を追加する。
3. 入力コアとGUIを同時に大改造しない。
4. `gofmt`、公開監査、Linux test/race、Windows test/vet/build/packageを実行する。
5. 成果物を独立検算する。
6. Windows実機チェックリストを実施する。
7. 未確認事項をPRと本資料へ残す。
8. 実機確認前にPRをマージ、安定版タグ付けしない。

## 10. 禁止事項

- 過去の破綻版・添付ZIP・archived/trash資料を正本にしない。
- フックコールバックをSleepさせない。
- HID I/Oをフックへ入れない。
- タイマーとUPの双方から長押しを二重実行しない。
- 物理入力をJoy-Con出力のUPで破壊しない。
- 設定再読込やフック再登録を跨いでHoldキーを残留させない。
- ルール編集のたびにJoy-Conを切断・再接続しない。
- 再検索ボタン連打を複数の接続試行へ展開しない。
- キャリブレーションを別プロファイルへ誤保存しない。
- 実機未検証の状態を「完全修正」「安定版」と断定しない。
- PR #5をユーザーの実機確認前にマージしない。

## 11. 次の作業

1. 文書更新を含む最終branch headの通常CIを全緑にする。
2. 同じ最終headからEXE、portable ZIP、source ZIP、SHA-256一覧を取得する。
3. ハッシュ、ZIP内部EXE、source ZIP commit commentを独立検算する。
4. PR #5本文へ最終head、CI run、成果物SHAを記録する。
5. ユーザーへRC一式を渡す。
6. `docs/REGRESSION_CHECKLIST.md`に沿ってJoy-Con（L）実機結果を記録する。
7. 実機不具合を同じブランチで修正し、必要なら`rc2`へ更新する。
8. 全必須項目完了後だけ`8.4.0`へ版更新し、PRをReadyにしてマージする。
9. `main`上の最終commitへ`v8.4.0`タグを付け、GitHub Releaseを発行する。
