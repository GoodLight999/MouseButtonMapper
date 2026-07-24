# Tauri導入評価

## 結論

Tauriは将来のGUIホスト候補ですが、v8.2.0安定化ブランチには導入しません。

## 理由

- Tauri on WindowsもWebView2を使用するため、WebView2依存そのものは消えません。
- 現在のGo入力コアを維持するにはTauriからGo EXEをsidecarとして起動する構成が現実的です。
- sidecar化すると単一プロセスではなくなり、多重起動、終了順序、クラッシュ復旧、スタートアップ、ログ、ポート受け渡しの設計が追加で必要です。
- Tauriの標準配布はMSI/NSISを中心としており、現在の「インストール不要の単純なEXE/ZIP」という要件と緊張します。
- GUI基盤変更と入力コア変更を同時に行うと、過去に発生した操作不能・空白GUIを再発させる危険が高いです。

## 将来試す場合の条件

- `experiment/tauri-host`のような独立ブランチで行う
- Goコアを変更しないsidecar方式から開始する
- 現行Edge app版と同じHTTP APIを使う
- 回帰チェックリストをすべて通す
- Tauri版が失敗してもGo単体版へ即時に戻せる配布を維持する
- 安定版を置換する前に長時間実機試験を行う

## 参照

- Tauri sidecar: https://v2.tauri.app/develop/sidecar/
- Tauri WebView2: https://v2.tauri.app/reference/webview-versions/
- Windows installer/WebView2 options: https://v2.tauri.app/distribute/windows-installer/
