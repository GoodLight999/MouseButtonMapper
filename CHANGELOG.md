# Changelog

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
