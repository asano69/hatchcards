# Design

- ラップトップで編集したマークダウンファイルは、Gitにプッシュすることになる。
- サーバは、あらかじめ設定したリポジトリから定期的にミラーする。
- go-gitでのミラー処理をPocketBaseのバイナリに直接組み込み、PocketBase内蔵のcronで動かす
- 強制的にミラーをトリガーするためのAPIをつくる。


## 秘密管理

- DBに保存する秘密情報を暗号化 → 使用時にメモリ上でのみ復号。
- crypto/aes + crypto/cipher(GCM) だけで十分に実用的。依存の脆弱性リスクやAPI変更の心配もない。
- DBにはEncryptの戻り値をそのまま（byteaやbase64で）保存。使う直前にDecryptでメモリに展開し、gitのSSH鍵やトークンとして利用。使い終わったらfor i := range plaintext { plaintext[i] = 0 }でゼロクリアする。
- マスターキー（32バイト）自体は環境変数から取得する。
