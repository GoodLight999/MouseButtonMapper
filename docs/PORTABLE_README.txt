MouseButtonMapper v8.4.0-rc2 portable edition

この版はJoy-Con（L）／Switch互換Raw HID／XInput実機検証用のRelease Candidateです。
安定版8.3.0を置き換える前に、別フォルダーで動作確認してください。

1. MouseButtonMapper.exeを任意のフォルダーへ置いて起動します。
2. 自動起動する場合はAdd_Startup_Shortcut.cmdを一度実行します。
3. 自動起動を外す場合はRemove_Startup_Shortcut.cmdを実行します。
4. 操作不能時はCtrl+Alt+Shift+F12、またはEmergency_Stop_MouseButtonMapper.cmdを使います。

設定は次へ保存され、配布フォルダーを削除しても残ります。
%LOCALAPPDATA%\MouseButtonMapper\config.json

旧インストール方式だけを消し、設定を残す場合:
Remove_Legacy_Install_Keep_Config.cmd

Joy-Con（L）:
- Windows Bluetooth設定でJoy-Con（L）を通常ペアリングします。
- 設定画面でJoy-Conを有効にし、Joy-Conを接続・再検索を押します。
- ゲームコントローラー入力を記録し、必要ならスティックをキャリブレーションします。
- 自動認識されない互換品は「優先するJoy-Con／互換HID識別子」の候補を選んで保存します。
- 互換Raw HIDを初めて確認するときは、デバイス占有を避けるためSteamを完全終了してください。
- XInputコントローラーは最大4台を自動検出します。
- BetterJoy、ViGEmBus、HidHideは必須ではありません。
- ゲーム側も物理コントローラーを読む場合は二重入力になり得ます。

長押し設定は、割り当て編集欄の「この割り当てで長押し判定を使う」から設定します。
