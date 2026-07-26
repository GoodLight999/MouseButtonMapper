# Release procedure

## Release Candidate

1. `feature/joycon-l`のDraft PRと`docs/HANDOFF.md`が最新であることを確認する。
2. `version.go`、`CHANGELOG.md`、README、portable説明、CI／Release workflowの版番号を同じRC番号へ揃える。
3. CIがPull Requestの一時merge refではなく、PR head SHAを明示checkoutしていることを確認する。
4. Linuxのgofmt、公開監査、test、raceと、Windowsのtest、vet、build、self-testを通す。
5. 次の4成果物を同じcommitから生成する。
   - 単体EXE
   - Windows x64 portable ZIP
   - source ZIP
   - SHA-256一覧
6. SHA-256一覧を独立再計算し、portable ZIP内部EXEが単体EXEと同一であることを確認する。
7. source ZIP先頭のcommit表記がPR head SHAと一致することを確認する。
8. RCはDraft PRまたはGitHub Prereleaseとして配布し、安定版と明確に区別する。
9. `docs/REGRESSION_CHECKLIST.md`に沿ってWindows実機試験を行う。

## Stable release

1. Joy-Con（L）のBluetooth接続、全入力、切断・再接続、スリープ復帰、ゲーム同時操作、1時間入力、8時間常駐を完了する。
2. 未確認項目または重大な既知不具合が残っていないことを確認する。
3. RC表記を外して`8.4.0`へ更新し、最終CIを通す。
4. PR #5をReadyにし、レビュー後に`main`へマージする。
5. `main`の最終commitへ`v8.4.0`タグを作成してpushする。
6. `release.yml`がEXE、portable ZIP、source ZIP、SHA-256一覧を生成する。
7. GitHub Releasesへ4成果物が登録されたことを確認し、ハッシュを再検算する。
8. Releaseから取得したportable ZIPを新規フォルダーへ展開し、最終スモークテストを行う。

コード署名を導入するまでは、署名済みであるかのような表現をしないでください。
