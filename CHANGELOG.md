# Changelog

## 8.4.0-rc5

- 製品の主軸をマウス／キーボードのGUI-firstリマッパーとして明文化し、コントローラー対応を実験機能へ固定
- 短押し／長押しの実行内容へ左／右／中クリック、サイド1／2、ホイール上下を直接追加できるGUIを追加
- 出力記録がキーボードしか受け付けず、ホイールとサイドボタンを記録できなかった不具合を修正
- Win32フォールバック編集画面でもマウス出力の保存・テストを可能にし、Web UIとの差異を解消
- 汎用マウス／キーボード出力処理をJoy-Con名義の実装から`output_windows.go`へ分離し、controller非依存の中核として整理
- ホイール／サイドボタン出力記録とWeb UIクイック操作の回帰テストを追加
- 通常のルール画面を常に「マウス・キーボードの割り当て」とし、controller UI表示時にも主従を逆転させないよう修正

## 8.4.0-rc4

- BetterJoyの実コードを確認し、互換品検出を`全HID列挙 + 明示的な3rd-party登録`方式へ変更
- Generic Desktop Gamepad/Joystick Usage filterを撤去し、vendor-specific collectionの互換Joy-Conも候補表示
- 自由入力の「互換HID識別子」を廃止し、現在列挙された正確なHID interface fingerprintからだけ登録可能に変更
- VID/PID/serial/fingerprint/productをプロファイルへ保存し、exact interface優先・serial fallbackで再接続
- metadataを開けないHID interfaceも、path fingerprintとpath由来VID/PID、inspect error付きで一覧へ残す
- 明示登録した互換品は書込み可能ならNintendo report-mode初期化を試し、Open/Write拒否時だけread-onlyへfallback
- `Controller.Visible`を追加し、専用controller sectionとHold UIを設定画面から完全に除去可能に変更
- `機能を停止して設定画面から隠す`でVisible/Enabledを同時にOFFにし、workerと保持出力を安全停止
- 非表示時は一般管理欄の小さな`実験機能を表示`だけを残し、controller名称・詳細を通常画面から排除
- 全HID候補の誤選択警告、Steam完全終了手順、製品名・VID/PID・serial・Usage・report長表示を追加
- Config.Versionを11へ更新

## 8.4.0-rc3

- Joy-Con／Switch互換Raw HID／XInputを一括して停止・非表示にする全体オプトイン設定を追加
- 新規設定とVersion 9以前からの移行ではコントローラー機能を既定OFFに変更
- OFF時はHID列挙、XInput polling、コントローラーevent処理、controller rule実行を停止
- OFFでも保存済みcontroller rule、profile別Joy-Con設定、calibration、Stable IDを削除せず保持
- OFF時もmouse/key rule、long press、profile switch、自動切替が継続するようruntime stateを分離
- controller Hold／long-press中にOFFへ切り替えた場合の安全解放を追加
- 詳細controller UIとHold設定をON時だけ表示し、再有効化用のglobal toggleだけを常時表示
- BetterJoy／JoyToKeyへ入力処理を外注する構成をUI・README・引き継ぎ資料へ明記
- XInput入力が長押しkey生成とvalidationから漏れていた不具合を修正
- controller OFFがmouse long-pressを巻き添えで消さない回帰試験を追加
- Config.Versionを10へ更新し、RC成果物を8.4.0-rc3へ統一

## 8.4.0-rc2

- 純正Joy-Con専用だったHID列挙を、Generic DesktopのGame Pad／Joystickコレクション候補まで拡張
- VID/PIDが異なる互換品を自動誤選択せず、検出候補のStable IDを明示選択できる互換モードを追加
- HID Usage Page、Usage、入力／出力report長、製品名、VID/PIDを診断表示へ追加
- Nintendo系`0x30`／`0x21`／`0x3f`に加え、SDLのSwitchドライバーで使われる7バイトinput-only形式を解析
- 書込み非対応または短いinput reportの互換品は、Nintendoサブコマンドを送らないread-only経路へ安全に降格
- 未対応Raw HID reportは長さと先頭hexをエラーへ残し、機器固有マッピングを追加可能に変更
- XInputを最大4台まで動的ロード・監視し、A/B/X/Y、LB/RB、LT/RT、十字キー、左右スティック方向、押込み、Start/Backへ対応
- Joy-ConとXInputを共通コントローラー入力状態へ統合し、短押し、長押し、Hold、入力記録、マウス／キーボード複合ルールを共用
- コントローラー切断時に全DOWNを合成UPし、Hold出力や長押し状態を残留させない処理を追加
- XInputとJoy-Conの状態キーを入力種別込みで分離し、同名コードの衝突を防止
- 同一report内の方向切替は全UPを全DOWNより先にemitする回帰試験を追加
- ゲームコントローラーUIへXInput接続状態、最後の入力、互換HID候補datalistを追加
- Linux unit/race/vetとWindowsクロスコンパイルを通過。純正／互換Joy-ConとXInputの実機試験は継続中

## 8.4.0-rc1

- Nintendo Switch Joy-Con（L）をWindows Bluetooth HIDから直接列挙・接続する入力ワーカーを追加
- 主要入力報告`0x30`とサブコマンド応答`0x21`から、ボタン、左スティック、バッテリー状態を解析
- 上下左右、L、ZL、SL、SR、－、キャプチャー、スティック押込みを割り当て可能に変更
- 左スティックへキャリブレーション、デッドゾーン、押下／解放ヒステリシス、4方向／8方向、X/Y反転を追加
- Joy-Con＋マウス、Joy-Con＋キーボードの複合ルールを既存ルールエンジンへ統合
- Joy-Con入力の短押し／長押し／キャンセルとHold出力を追加
- キー、複数キー同時押し、マウスボタン、ホイール出力へ対応
- Hold出力へ参照カウントを導入し、物理キーボードと同じキーを共有しても物理押下を勝手に解放しないよう修正
- 切断、緊急停止、設定再読込、フック再登録、終了時にJoy-Con由来の押下・長押し・Hold出力を解放
- HID読取停止へ`CancelIoEx`を追加し、終了・再接続時のブロッキング読取を解除
- 自動再接続、手動再検索、再接続無効時の待機、状態通知の最大20Hz制限を追加
- ルールだけを編集した際にJoy-Conを不要に切断・再接続しないよう再構築経路を分離
- キーボードを最後の入力とする複合ルールでも、Joy-Con／マウス前置入力を消費済みとして扱うよう修正
- Joy-Con設定、接続状態、入力記録、スティック設定、キャリブレーションのWeb UI/APIを追加
- 旧`config.json`との後方互換を維持したプロファイル別Joy-Con設定を追加
- Linux単体試験・race detector、Windows単体試験・vet・GUIビルド・portable packagingを通過
- 実機Bluetooth、切断・再接続、スリープ復帰、ゲーム同時操作、1時間入力、8時間常駐はRC段階で未確認
- `0x3f`簡易報告は限定フォールバックのままとし、初版の主経路には使用しない

## 8.3.0

- サイドボタンまたは修飾キー以外のキーボードキーに長押し判定を追加
- 短押し時と長押し時へ別々の実行内容を設定可能
- 長押し時に短押しの実行だけをキャンセルするモードを追加
- 長押し判定時間を100〜5000msで設定可能
- 長押しタイマーをフックコールバック外で処理し、押下解除境界でも二重発火しない状態機械を追加
- 長押し時の実行内容にも記録・テスト操作を追加
- Windows CIで長押し状態機械の実行テストを追加
- 終了処理と長押しタイマーが競合しても、閉じた出力チャネルへ送信しないよう停止手順を強化

## 8.2.0

- 前面アプリ連動プロファイル切替を修正
- `SetWinEventHook(EVENT_SYSTEM_FOREGROUND)` と `GetForegroundWindow` の定期確認を併用
- 自動切替の保存UIを対象別の明確な操作へ分離
- プロセス名の `.exe` 有無を同一視
- 自動切替の判定理由・現在適用中プロファイルを診断表示
- v8.0.0の専用フックスレッド、出力ワーカー、フック監視を維持

## Repository stabilization

- 埋め込みHTMLと既定設定を `web_assets.go` へ無変更で分離
- CI用 `--self-test` を追加
- 公開リポジトリ監査、Windowsビルド、ポータブルZIP生成をGitHub Actions化
- 完全引き継ぎ資料と回帰試験チェックリストを追加
