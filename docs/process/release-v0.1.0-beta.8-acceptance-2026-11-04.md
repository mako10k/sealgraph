# beta.8 candidate acceptance and publication authority

2026-11-04、ユーザーは「候補一式と検証記録を受け入れます。v0.1.0-beta.8として公開し、公開後のタグ・配布物の読み戻し確認までお願いします。」と指示した。

Accepted candidate manifest SHA-256: 501e4c92ec5283ef2a8633a5bf226f1062120027933beb162f0cae7a194f4ea5。
Release tag v0.1.0-beta.8は固定source f75b43ef3a38ca1eb8f73a7220251db8a487a2b4を指す。候補記録を持つ後続branch HEADへ付けない。Linux amd64 tar.gz（2205801 bytes、SHA256 7ef439d354081a6b327b57defcff3664cf475be8c8c53a4de5704c26cf4a214c）とchecksum file（108 bytes、SHA256 dc5383731034a21d20e22f9b79222bac2d0d402d41c1c7e085ef312dd4fc9ead）の2 assets。公開ノートdocs/releases/v0.1.0-beta.8.mdは候補ノートの公開状態表記のみ更新、SHA256 b1965525d84952fe4f168b9677b8f6bb8908afc023c17fa07e73368f062d3121。

Rootが唯一の実行者。予定remote mutationsはtag push1、Release object create1、asset upload2、draftからprereleaseへのpublish1。prerelease=true、latest=false。draftでassets/body/tagを確認してから公開し、公開後にもmetadata/download bytesを読戻す。既存tag/Releaseがあれば上書きしない。失敗/曖昧結果は読取りで状態を確定してから判断し、force/clobber/盲目的再試行をしない。main統合/インストール/実repo Seal更新は今回対象外。

独立payloadレビュー `/root/beta8_publication_payload_review` は、source tree、2 assetsのsize/hash/checksum、safe archive inventory、binary version、README/LICENSE source一致、ノートの差分を読取り照合し、confirmed blockerなし・実行推奨と報告した。限定のcredential-pattern確認も異常なし（網羅的な秘密検査ではない）。独立レビューは所有者受入の代替ではなく、remote state確認は実行者の後続作業。元candidate manifestのprepared_unpublished/pendingは当時のsnapshotとして改変せず保持する。
