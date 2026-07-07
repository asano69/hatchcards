# `connections` コレクションと token 暗号化の設計

このドキュメントは、Gitea/GitHub リポジトリをミラーするための `connections`
コレクションについて、なぜ CRUD の一部だけを専用 API 経由にしているのか、
あとから見て判断理由がわかるように残すものです。

## 前提: なぜ全部を汎用 Record API に任せられないか

PocketBase の Record API（`pb.collection("connections").create(...)` など）
をフロントエンドから直接叩けば、コード量は最小で済みます。しかし
`connections` コレクションには「平文のまま DB に保存してはいけない」
フィールド（token）があります。汎用 Record API は受け取った値をそのまま
保存するだけなので、暗号化のような**書き込み時に必ず通したい変換処理**を
挟む場所がありません。

`internal/db/db.go` の `reviews` → `cards` 同期に使っている
`OnRecordCreate` フックも、発想は同じです。「レビューが保存されたら
副作用としてカードの performance も更新する」という不変条件を、呼び出し側
（ハンドラ）の実装ミスに関係なく保証したいので、DB 層のフックに寄せて
います。token の暗号化も同じ理由で「呼び出し側の実装に依存させない場所」
に置く必要があります。

フックと専用 API のどちらでも実現できますが、今回は専用 API を選びました。
理由は次の通りです。

- `OnRecordCreate`/`OnRecordUpdate` フックで暗号化しようとすると、
  「クライアントが平文 token をそのまま `token_ciphertext` に送ってきた
  場合」と「暗号化済みの値を再度暗号化されると困る更新の場合」を
  フック内で区別する必要があり、かえって複雑になります。
- 専用 API なら、リクエストの型（`connectionRequest.Token` は常に平文、
  レスポンスの型は常に `core.Record` で `token_ciphertext` を含む）が
  最初から固定されるため、区別のロジックが不要です。

## 採用した設計: 操作ごとに窓口を分ける

`connections` に対する 5 つの操作を、暗号化が関わるかどうかで 2 グループに
分けました。

| 操作 | 経路 | 理由 |
|---|---|---|
| 一覧 (List) | 汎用 Record API | 平文 token を持っていないので素通しで安全 |
| 詳細 (View) | 汎用 Record API | 同上 |
| 削除 (Delete) | 汎用 Record API | token を書き込まない操作なので暗号化は無関係 |
| 作成 (Create) | 専用 API (`POST /api/connections`) | 平文 token を受け取り、暗号化してから保存する必要がある |
| 更新 (Update) | 専用 API (`PATCH /api/connections/{id}`) | 同上。ただし token が空なら再暗号化せず既存値を保持する |

ポイントは「暗号化が絡む操作だけ」専用 API を通す、という線引きです。
一覧・削除まで専用 API に持っていくと単なる薄いラッパーが増えるだけで、
見通しがかえって悪くなります（simplicity-first の方針に反する）。

### 一覧・削除が汎用 API のままで安全な理由

`migrations/*_collections_snapshot.go` で `token_ciphertext` フィールドに
`"hidden": true` を設定してあります。これにより、この項目は PocketBase の
Record API のレスポンスに一切含まれません。つまり一覧・詳細取得を汎用
API に任せても、暗号化済みの値がクライアントに漏れることはありません
（`_superusers` コレクションの `password` フィールドと同じ扱いです）。

### 作成・更新のデータフロー

```
[Connections.jsx]
  平文 token を含むフォーム
        │ POST /api/connections
        │ PATCH /api/connections/{id}
        ▼
[connections_api.go] RegisterConnectionsAPI
  リクエストボディをパース (connectionRequest)
        │ ConnectionInput に変換
        ▼
[db/connections.go] CreateConnection / UpdateConnection
  cryptoutil.Encrypt(token) → token_ciphertext
        │ record.Set(...) → db.app.Save(record)
        ▼
[PocketBase] "connections" テーブルに保存
  token_ciphertext だけが書き込まれる。平文 token はどこにも残らない
```

更新時、`in.Token == ""` の場合は `token_ciphertext` を触りません。
これにより「トークンを変更せずに他のフィールドだけ編集する」という
一般的な編集フローで、ユーザーが毎回トークンを再入力する必要がなくなり
ます（`Connections.jsx` の `token` 入力欄が空でも保存できるのはこのため）。

### 復号のタイミング

復号 (`DecryptConnectionToken`) はミラー処理（go-git の `PlainClone`/
`Fetch` に渡す直前）でのみ呼び出します。復号結果はメモリ上にのみ存在し、
使用後は `cryptoutil.Zero` で明示的にゼロクリアします。DB や JSON
レスポンスに平文が書き出される経路は存在しません。

```
[mirror 処理] (未実装、今後のステップ)
  DecryptConnectionToken(id)
        │ plaintext []byte
        ▼
  remote_url に username:token を埋め込み go-git へ渡す
        │
        ▼
  cryptoutil.Zero(plaintext)  // 使用後は必ずゼロクリア
```

## マスター鍵について

- 環境変数 `HATCHCARDS_MASTER_KEY`（base64 エンコードされた 32 バイト）
  から取得します。
- `cryptoutil.loadMasterKey` はプロセス起動時に一度だけ読み込むのでは
  なく、`Encrypt`/`Decrypt` の呼び出しごとに毎回読み直します。鍵を
  パッケージ変数としてメモリに保持し続けないようにするためです。
- マイグレーションはコード管理していますが（`Automigrate: false`、
  `internal/migrations`）、`connections` コレクション自体のスキーマ編集
  （`hidden` フラグやユニークインデックスの追加）は PocketBase の
  ダッシュボードから行う運用です。

## この設計であえてやらないこと（スコープ外）

- go-git を使った実際の clone/fetch 処理。
- ミラー処理の cron 実行（PocketBase 組み込みの cron を使う想定だが未着手）。
- `last_synced_at` / `last_error` の更新ロジック（ミラー処理本体と一緒に
  実装する）。
- デプロイキー方式（DB を消すたびに再登録が必要になり運用上不便なため
  不採用。詳細は要件定義時のやり取り参照）。

## 関連ファイル

- `internal/cryptoutil/cryptoutil.go` — AES-256-GCM による暗号化/復号
- `internal/db/connections.go` — `connections` コレクションへの読み書き
- `internal/cmd/serve/connections_api.go` — 作成/更新専用の HTTP ハンドラ
- `frontend/src/routes/Connections.jsx` — 登録・編集・削除 UI
- `migrations/*_collections_snapshot.go` — `connections` コレクションの
  スキーマ定義（`token_ciphertext` の `hidden` 設定はここで確認できる）