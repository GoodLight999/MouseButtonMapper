# Release procedure

1. `main`のCIが成功していることを確認する。
2. `version.go`、`CHANGELOG.md`、既定設定の`SavedBy`を更新する。
3. `vX.Y.Z`タグを作成してpushする。
4. `release.yml`がWindows x64ポータブルZIPとSHA-256を生成する。
5. GitHub Releasesへ成果物が登録されたことを確認する。
6. Windows実機で配布ZIPを新規フォルダーへ展開して回帰試験する。

コード署名を導入するまでは、署名済みであるかのような表現をしないでください。
