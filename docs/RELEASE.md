# Release procedure

## Release Candidate

1. `feature/joycon-l`のDraft PR #5と`docs/HANDOFF.md`が最新であることを確認する。
2. `main`が安定版`8.3.0`のままであり、RC作業が`feature/joycon-l`だけに存在することを確認する。
3. `version.go`、`CHANGELOG.md`、README、portable説明、CI／Release workflowの版番号を同じRC番号へ揃える。
4. CIがPull Requestの一時merge refではなく、`github.event.pull_request.head.sha || github.sha`を明示checkoutしていることを確認する。
5. 次の自動検証を最終branch headで通す。
   - `gofmt`
   - 公開リポジトリ監査
   - Linux `go test ./...`
   - Linux `go test -race ./...`
   - Windows `go test ./...`
   - Windows `go vet -unsafeptr=false ./...`
   - Windows GUIビルド
6. 再検索gate、方向release-before-press、`0x3f`物理ボタン／hat分離、停止時worker解放、キャリブレーション保存先の回帰試験が含まれていることを確認する。
7. 次の4成果物を同じ最終headから生成する。
   - `MouseButtonMapper-v8.4.0-rc1.exe`
   - `MouseButtonMapper-v8.4.0-rc1-windows-x64.zip`
   - `MouseButtonMapper-v8.4.0-rc1-source.zip`
   - `MouseButtonMapper-v8.4.0-rc1-SHA256SUMS.txt`
8. Actions artifactを新規フォルダーへ取得し、SHA-256一覧を独立再計算する。
9. portable ZIP内部の`MouseButtonMapper.exe`と単体EXEがバイト単位で同一であることを確認する。
10. source ZIPのZIP commentが最終PR head SHAと完全一致することを確認する。
11. 単体EXEがPE32+ Windows GUI x86-64であること、portable ZIPが正常に展開できることを確認する。
12. PR #5本文へ次を記録する。
    - 最終head SHA
    - 全緑CI run ID
    - 4成果物のSHA-256
    - 実機未検証項目
13. RCは実機試験用であり、安定版ではないことを明記する。PRはDraft・未マージのまま維持する。
14. `docs/REGRESSION_CHECKLIST.md`に沿ってWindows実機試験を行う。

## Hardware gate

次の実機項目が終わるまで、PRをReadyまたはMergeにしない。

- Bluetoothペアリングと`0x30`report-mode negotiation
- 全物理ボタン、スティック軸、スティック方向、バッテリー表示
- 電源断、Bluetooth切断、切断・再接続20回
- スリープ／復帰
- Joy-Con＋右手マウス／キーボード複合操作
- 実ゲームでの二重入力
- 高負荷ゲームと高イベント量マウスの共存
- 1時間連続入力
- 8時間常駐

## Stable release

1. Hardware gateをすべて完了する。
2. 未確認項目または重大な既知不具合が残っていないことを確認する。
3. RC表記を外して`8.4.0`へ更新し、最終CIを通す。
4. PR #5をReadyにし、レビュー後に`main`へマージする。
5. `main`の最終commitへ`v8.4.0`タグを作成してpushする。
6. `release.yml`がEXE、portable ZIP、source ZIP、SHA-256一覧を生成する。
7. GitHub Releasesへ4成果物が登録されたことを確認し、ハッシュを再検算する。
8. Releaseから取得したportable ZIPを新規フォルダーへ展開し、最終スモークテストを行う。

コード署名を導入するまでは、署名済みであるかのような表現をしないでください。
