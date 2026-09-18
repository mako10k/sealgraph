# Issue #17 R4-c1 要件候補 — ファイル中の部分文字列を直接指定する Trace 登録

- 状態: Proposed（所有者の初回レビュー待ち）
- 日付: 2026-09-18
- 上位入力: 所有者の「ファイル中の部分文字列を直接指定して登録できるようにしたい。検査に引っかからなかったらエラー」と、指定文字列を Seal 内容にも設定し、複数一致時は先頭の一致を採用するという同日の回答。
- 基底: Accepted Issue #17 R1・R2・R3。既存の要件本文と Seal は変更しない。

## 目的と範囲

利用者が元ファイルの byte 位置と JSON recipe を手で書かなくても、一つの元ファイルに含まれる非空の部分文字列から、一つの External run を持つ小さな Seal を作れるようにする。この操作は既存 Candidate 一件の内容と OriginMap を同時に設定する。root、Cause、draft、attachments、REF の選択は既存 Candidate の明示的な操作に従い、Seal の公開は引き続き別の `seal REF` で行う。

入力文字列は正規化・大小文字変換をせず、指定された UTF-8 bytes のまま扱う。元ファイル全文を安定して読み、入力 bytes が連続して一致する開始位置を byte 0 から探索する。一件もなければエラーとし、Candidate を更新しない。複数あれば byte offset が最小の一致を `source_start` として記録する。この位置は保存時に成立した一件と後続探索のヒントであり、唯一の由来位置を意味しない。

成功時は入力文字列の bytes を Candidate content 全体とし、一つの External run がその全体を覆う。元ファイル全文は、run 外を含め SourceSnapshot として immutable に保存する。全文保存の事前告知、入力パスの安全性、Candidate の競合検出、失敗時の境界は既存 `trace set` と同等にする。現在ファイルとのローカル binding は自動作成しない。既存の recipe 指定、多 source・Untraced run、Cause、STALE、比較・位置一覧の意味は維持する。

## 受入条件

1. 元ファイルに指定 bytes が一件あると、Candidate content と単一 External run の copy 一致が成立し、Seal 後に元ファイル全文を復元できる。
2. 複数一致では最小 byte offset を記録する。重複する一致を含む全位置の導出は、従来の `trace occurrences` の責務とする。
3. 不一致、空文字列、元ファイル読取失敗、既存 Candidate 不在、競合では理由付きで失敗し、Candidate と REF HEAD は変わらない。
4. 成功しても local source binding と Seal は暗黙作成しない。既存の recipe 経路は同じ入力で同じ OriginMap を作れる。

## 未決の下流設計

公開コマンド・flag 名、文字列の安全な入力方法、成功出力 schema と help/completion、既存 `trace set` への組込みか別操作かは後継 CLI 判断で定める。新しい保存 schema・repository format は要求しない。R4-c1 の採用は ADR、実装、実 repository の移行、push、release、merge を許可しない。
