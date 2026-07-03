# api/sessionのふるまい

`GET /api/sessions`エンドポイントは、利用可能なドリルセッションのリストと各デッキの平均想起確率を返します。 [1](#1-0) 

## 概要

このエンドポイントは以下の動作をします：

1. **コレクションの再読み込み**: リクエストごとにディスクからコレクションを再読み込みし、起動後に追加・削除されたデッキ/カードを反映します [2](#1-1) 
2. **セッションリストの構築**: `config.toml`の`[[session]]`設定をJSON形式に変換します [3](#1-2) 
3. **平均想起確率の計算**: 各セッションのデッキについて、カードの平均想起確率を計算します [4](#1-3) 

## 返されるJSON構造

各セッションは以下の情報を含みます：

```json
{
  "name": "セッション表示名",
  "path": "デッキパス",
  "drill_url": "/drill?deck=...",
  "retri_pct": 85.5
}
```

- `name`: `config.toml`の`title`またはデッキ名 [5](#1-4) 
- `path`: デッキ名から変換されたURLパスセグメント [6](#1-5) 
- `drill_url`: ドリルページへのURL [7](#1-6) 
- `retri_pct`: 平均想起確率（0-100のパーセンテージ） [8](#1-7) 

## RetriPctの計算方法

`computeAvgRetrieval`関数は以下のロジックで計算します [4](#1-3) ：

1. デッキフィルタが指定されている場合、そのデッキのカードのみを対象にします
2. 各カードについて：
   - 新しいカード（未レビュー）は0%として扱います
   - レビュー済みカードの場合、最後のレビューからの経過日数と安定性（stability）からFSRSの想起確率を計算します
3. 全カードの平均をパーセンテージで返します

## フロントエンドでの使用

`frontend/src/routes/Sessions.jsx`がこのAPIを呼び出し、セッションリストを表示します [9](#1-8) 。

## Notes

- このエンドポイントは`api/session`（単数形）ではなく`api/sessions`（複数形）です
- コレクションの再読み込みはリクエストごとに行われるため、サーバーを再起動しなくてもデッキの変更が反映されます [10](#1-9) 

Wiki pages you might want to explore:
- [Glossary (asano69/hashcards)](/wiki/asano69/hashcards#10)
- [serve Command (asano69/hashcards)](/wiki/asano69/hashcards#2.1)

### Citations

**File:** internal/cmd/serve/serve.go (L60-73)
```go
	// GET /api/sessions reloads the collection from disk on every request so
	// that decks/cards added or removed since startup are reflected in the
	// session list and the retrievability percentages shown on the index
	// page. This mirrors the reload that the old server-rendered "/" route
	// used to perform before rendering index.html.
	router.HandleFunc("GET /api/sessions", func(w http.ResponseWriter, r *http.Request) {
		freshCol, err := collection.Load(cfg.Data.Root, database)
		if err != nil {
			freshCol = col
		}
		sessionJSON, _ := json.Marshal(buildSessionList(cfg, freshCol, database))
		w.Header().Set("Content-Type", "application/json")
		w.Write(sessionJSON)
	})
```

**File:** internal/cmd/serve/serve.go (L153-184)
```go
func computeAvgRetrieval(col *collection.Collection, database *db.Database, deckFilter string) float64 {
	today := types.Today()

	cards := col.Cards
	if deckFilter != "" {
		var filtered []types.Card
		for _, c := range cards {
			if c.DeckName() == deckFilter {
				filtered = append(filtered, c)
			}
		}
		cards = filtered
	}

	if len(cards) == 0 {
		return 0
	}

	var total float64
	for _, card := range cards {
		perf, err := database.GetCardPerformance(card.Hash())
		if err != nil || perf.IsNew() {
			// New cards contribute 0; total is unchanged.
			continue
		}
		rp := perf.Reviewed()
		elapsed := today.Time().Sub(rp.LastReviewedAt.Date().Time()).Hours() / 24
		total += fsrs.Retrievability(elapsed, rp.Stability)
	}

	return total / float64(len(cards)) * 100
}
```

**File:** internal/cmd/serve/serve.go (L188-203)
```go
func buildSessionList(cfg *config.Config, col *collection.Collection, database *db.Database) []sessionInfo {
	list := make([]sessionInfo, 0, len(cfg.Sessions))
	for _, sc := range cfg.Sessions {
		drillURL := "/drill"
		if sc.Path != "" {
			drillURL = "/drill?deck=" + url.QueryEscape(sc.Path)
		}
		list = append(list, sessionInfo{
			Name:     sc.Name,
			Path:     sc.Path,
			DrillURL: drillURL,
			RetriPct: computeAvgRetrieval(col, database, sc.FromDeck),
		})
	}
	return list
}
```

**File:** frontend/src/routes/Sessions.jsx (L6-12)
```javascript
async function fetchSessions() {
  const res = await fetch("/api/sessions");
  if (!res.ok) {
    throw new Error(`GET /api/sessions failed: ${res.status}`);
  }
  return res.json();
}
```
