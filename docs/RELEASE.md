# Release procedure

## RC

1. `main`が安定版8.3.0のままであることを確認する。
2. PR #5がDraft・未マージであることを確認する。
3. `version.go`、README、CHANGELOG、portable説明、CI／Release workflowを同じRC番号へ揃える。
4. 最終branch headで次を通す。
   - gofmt
   - public repository audit
   - Linux test/race
   - Windows test/vet
   - Windows GUI build
5. 同じheadから次を生成する。
   - `MouseButtonMapper-v8.4.0-rc3.exe`
   - `MouseButtonMapper-v8.4.0-rc3-windows-x64.zip`
   - `MouseButtonMapper-v8.4.0-rc3-source.zip`
   - `MouseButtonMapper-v8.4.0-rc3-SHA256SUMS.txt`
6. 独立検算する。
   - artifact digest
   - manifest SHA-256
   - standalone EXE == portable内部EXE
   - source ZIP comment == final head
   - EXEがPE32+ Windows GUI x86-64
7. PR本文へfinal head、CI run、artifact ID、SHA-256、未確認実機項目を記録する。
8. RCは実機試験用と明記し、stable tagを付けない。

## rc3必須回帰

- fresh/old configでcontroller featureがdefault OFF
- OFF時にJoy-Con／XInput workerが起動しない
- OFF時にlate controller eventを無視する
- OFF時にcontroller ruleをactive rulesから外すが保存データは維持する
- OFF時もmouse/key ruleとlong-pressが動く
- controller Hold中のOFFでowned keyをreleaseする
- ONへ戻すと保存済みruleと詳細UIが復帰する
- XInput long-press key／validationが動く

## Stable release

`docs/REGRESSION_CHECKLIST.md`の必須実機項目が終わり、重大な既知不具合がない場合だけ:

1. RC表記を外して8.4.0へ更新する。
2. 最終CIと成果物検算をやり直す。
3. PR #5をReadyにし、review後にmainへmergeする。
4. main上のcommitへ`v8.4.0`tagを付け、GitHub Releaseを発行する。
