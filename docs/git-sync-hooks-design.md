# Git同期後フック機能 — 設計まとめ（引き継ぎ用）

このドキュメントは、`connections`（gitミラー同期）機能に「同期後にスクリプトを
実行してMarkdownからJSONデッキを自動生成する」フックを追加するための設計案を
まとめたものです。マイグレーション自体は着手者が行う前提で、実装方針とその
根拠、既存挙動への影響範囲を記録します。

## 目的

- フラッシュカードの元データがJSONではなくMarkdownノートとして管理されている
  リポジトリでも、git同期後に変換スクリプトを走らせてJSONデッキを自動生成
  できるようにする。
- これにより、原理的に任意のリポジトリからフラッシュカードを取り込めるように
  なる。

## 前提となる制約（simplicity-first）

- 既存の「JSONのみのリポジトリはクローンするだけで自動認識される」という
  シンプルな挙動は絶対に壊さない。
- hookは完全にopt-in。設定しないconnectionは今までと1バイトも挙動が変わら
  ない。
- セキュリティ上、**任意のシェル文字列を実行させる設計は採用しない**。
  実行できるスクリプトは運用者があらかじめ配置した read-only なディレクトリ
  の中身に限定し、connectionが保持するのは「そのディレクトリ内のファイル名」
  という検証済み識別子のみとする。

## 全体アーキテクチャ

```
[git remote]
     │ mirror.Sync (clone/pull) ※既存コード、無変更
     ▼
local_path (作業ツリー。Markdownソースが入っている)
     │ hookが設定されていれば実行
     ▼
hook script (運用者が事前配置した実行ファイル1本、read-onlyディレクトリ内)
     │ $HATCHCARDS_SOURCE_DIR を読み、$HATCHCARDS_OUTPUT_DIR にJSONを書く
     ▼
DATA_ROOT/.generated/<local_path>/*.json
     │ walkDecks が再帰的に拾う（collection.go、無変更）
     ▼
既存のカード同期パイプライン（syncDB等）
```

ポイントは、生成物を**クローンした作業ツリーの外**（`DATA_ROOT/.generated/`
配下）に出すこと。作業ツリーの中に生成JSONを混ぜると、次回`pull`時に
「untracked working tree files would be overwritten」のような衝突を起こす
リスクがあるため。

## 変更・追加が必要な箇所

### 1. config（`internal/config/config.go`）

```go
type DataConfig struct {
	Root     string
	HooksDir string // "" or unset dir => no hooks available
}
```

環境変数 `HOOKS_DIR`（デフォルト `./hooks`）を1つ追加するだけ。

### 2. 新規パッケージ `internal/hook`

```go
// Package hook resolves and runs pre-installed post-sync scripts. Scripts
// are never uploaded or authored through the API; they are placed on disk
// by the operator ahead of time, so the only thing a connection stores is
// a name that gets resolved against this fixed, read-only directory.
package hook

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/asano69/hatchcards/internal/errs"
)

// nameRe rejects anything but a bare identifier, so a name can never
// escape hooksDir via "../" or an absolute path.
var nameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// List returns available hook names, or an empty list if hooksDir doesn't
// exist (e.g. the operator hasn't configured any hooks). This keeps
// installations without a hooks directory working exactly as before.
func List(hooksDir string) ([]string, error) {
	entries, err := os.ReadDir(hooksDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errs.Newf("read hooks dir: %v", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !nameRe.MatchString(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
			continue
		}
		names = append(names, e.Name())
	}
	return names, nil
}

// Resolve validates name and returns the absolute path of the hook script.
// It re-checks existence/executability at call time (not just at save
// time) so hooksDir contents can change without restarting the server.
// Returns ("", nil) when name is empty, meaning "no hook configured".
func Resolve(hooksDir, name string) (string, error) {
	if name == "" {
		return "", nil
	}
	if !nameRe.MatchString(name) {
		return "", errs.Newf("invalid hook name: %q", name)
	}
	path := filepath.Join(hooksDir, name)
	info, err := os.Stat(path)
	if err != nil {
		return "", errs.Newf("hook not found: %s", name)
	}
	if !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return "", errs.Newf("hook is not executable: %s", name)
	}
	return path, nil
}

// Run executes the resolved script directly (no shell), passing the
// source and output directories as environment variables. sourceDir is
// the connection's git working tree; outputDir is where generated JSON
// decks should be written.
func Run(ctx context.Context, scriptPath, sourceDir, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return errs.Newf("create hook output dir: %v", err)
	}
	cmd := exec.CommandContext(ctx, scriptPath)
	cmd.Dir = sourceDir
	cmd.Env = append(os.Environ(),
		"HATCHCARDS_SOURCE_DIR="+sourceDir,
		"HATCHCARDS_OUTPUT_DIR="+outputDir,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errs.Newf("hook %q failed: %v\n%s", filepath.Base(scriptPath), err, out)
	}
	return nil
}
```

**なぜ`sh -c`を使わないか**: `exec.CommandContext(ctx, scriptPath)` はargvが
固定1要素のみで、シェル解釈が一切挟まらない。connection側にどんな文字列を
入れられても injection の余地がない。引数ではなく環境変数で
source/outputを渡すのも同じ理由（シェルの引用符展開に依存しない）。

### 3. `connections` スキーマへのフィールド追加

```go
type ConnectionInput struct {
	Name      string
	RemoteURL string
	Username  string
	Token     string
	Enabled   bool
	HookName  string // "" means no post-sync hook
}
```

保存時（`CreateConnection`/`UpdateConnection`）に一度 `hook.Resolve` を呼び、
存在しない・実行不可な名前ならエラーを返す（早期フィードバック）。

マイグレーションは着手者側で追加（`hook_name` フィールド、デフォルト空文字）。
既存レコードは自動的に「hookなし」として扱われるため、追加による既存データ
への影響はない。

### 4. `internal/cmd/serve/mirror_api.go` への配線

`mirror.Sync` 成功直後、hookが設定されている場合のみ実行:

```go
if syncErr == nil && mc.HookName != "" {
	scriptPath, err := hook.Resolve(cfg.HooksDir, mc.HookName)
	if err == nil {
		outputDir := filepath.Join(dataRoot, ".generated", mc.LocalPath)
		syncErr = hook.Run(ctx, scriptPath, localPath, outputDir)
	} else {
		syncErr = err
	}
}
```

hookの成功/失敗は既存の `RecordSyncResult` にそのまま渡す。新しいフィールド
は増やさず、既存の `last_error` 表示をそのまま流用できる。

### 5. API: `GET /api/hooks`

`hook.List(cfg.Data.HooksDir)` の結果をそのまま返すだけの薄いエンドポイント。
フロントエンドはこれで `<select>` の選択肢を構築する（自由入力欄にはしない
— 二重ロックの意味もある）。

### 6. フロントエンド（`Connections.jsx`）

hook名の入力を自由テキストではなく `GET /api/hooks` から取得した名前の
`<select>` にする。空選択肢 = hookなし。

### 7. デプロイ（Docker/compose）

```
hooks/                 <- ホスト側で用意、read-onlyでマウント
  question_to_json     <- 実行可能ファイル（言語不問）
```

`compose.yaml` に `./hooks:/hatchcards/hooks:ro` を追加。既存の
`scripts/question_to_json.py` のような「input_dir, output_dirを取る」契約の
スクリプトはそのまま流用できる（環境変数経由で渡す点だけ合わせる）。

## hookを使わない場合、挙動は変わるか

**結論: 変わらない。** ただし実装時に以下2点を守る必要がある。

1. **`hook.List` はhooksDir不在時にエラーを返さず空リストを返す**こと
   （上記実装済み）。これがないと、hooksディレクトリを用意していない既存
   環境で `/api/hooks` を叩いた瞬間にエラーになってしまう。
2. **`connections` テーブルの `hook_name` はデフォルト空文字**であること。
   これにより既存レコードは自動的に「hookなし」経路を通る。

この2点さえ守れば、`mirror.Sync`（clone/pull）、`collection.Load`
（walkDecksのJSON探索）、DBスキーマの根幹（cards/sessions/reviews）は
すべて無変更のまま維持される。

## 未決定・要確認事項

- **hookの単位**: 現在の案はconnectionごとに個別設定。運用イメージとして
  「リポジトリによって形式がバラバラ → connectionごと」で良いか、
  それとも「変換ロジックを1箇所に集約したい → グローバル1本」の方が
  合っているか、要確認。
- **タイムアウト**: `hook.Run` に `context.WithTimeout` を挟む上限値
  （デフォルト案: 5分、環境変数で上書き可能にするか）。
- **`.generated/` ディレクトリの掃除**: connectionが削除された場合、対応
  する `.generated/<local_path>/` を残すか消すか（今のところ未実装、
  現状は残置のまま運用者が判断する想定）。

## 関連ファイル（実装時に触る想定）

- `internal/config/config.go` — `HooksDir` 追加
- `internal/hook/hook.go` — 新規パッケージ（List / Resolve / Run）
- `internal/db/connections.go` — `ConnectionInput.HookName` 追加、
  保存時の `hook.Resolve` 検証
- `internal/cmd/serve/mirror_api.go` — 同期成功後のhook実行配線
- `internal/cmd/serve/connections_api.go` — `hook_name` の受け渡し
- 新規: `GET /api/hooks` エンドポイント
- `frontend/src/routes/Connections.jsx` — hook名を`<select>`化
- `migrations/` — `connections` コレクションへの `hook_name` フィールド
  追加（着手者側で作成）
- `compose.yaml` / Dockerfile — `hooks/` の read-only マウント追加