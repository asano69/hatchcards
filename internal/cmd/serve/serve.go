// Package serve implements the "serve" command, which runs a single HTTP server
// that hosts the index page and all drill sessions defined in the config file.
package serve

import (
	"encoding/json"
	"fmt"

	"io"
	"net/http"
	"net/url"

	"github.com/asano69/hashcards/internal/assets"

	"github.com/asano69/hashcards/internal/cmd/drill/handlers"

	"github.com/asano69/hashcards/internal/collection"
	"github.com/asano69/hashcards/internal/config"
	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/fsrs"

	"github.com/asano69/hashcards/internal/types"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// sessionInfo is the JSON representation returned by /api/sessions.
type sessionInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	DrillURL string `json:"drill_url"`
	// RetriPct is the average retrievability of the deck's cards, as a
	// percentage in [0, 100]. New (unreviewed) cards contribute 0%.
	// Orphan cards are excluded because only collection cards are considered.
	RetriPct float64 `json:"retri_pct"`
}

// Run opens the database and collection once, registers all drill routes, then
// starts listening. The database and collection are shared across all sessions.
func Run(app *pocketbase.PocketBase, cfg *config.Config, out io.Writer) error {
	database, err := db.New(app)
	if err != nil {
		return err
	}

	col, err := collection.Load(cfg.Data.Root, database)
	if err != nil {
		return err
	}
	fsrsCfg := fsrs.FSRSConfig{
		TargetRecall: cfg.FSRS.TargetRecall,
		MinInterval:  cfg.FSRS.MinInterval,
		MaxInterval:  cfg.FSRS.MaxInterval,
	}

	router := http.NewServeMux()

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

	manager := handlers.NewManager()
	for _, sc := range cfg.Sessions {
		var cardLimit, newCardLimit *int
		if sc.CardLimit > 0 {
			cl := sc.CardLimit
			cardLimit = &cl
		}
		if sc.NewCardLimit > 0 {
			ncl := sc.NewCardLimit
			newCardLimit = &ncl
		}
		var deckFilter *string
		if sc.FromDeck != "" {
			df := sc.FromDeck
			deckFilter = &df
		}
		burySiblings := sc.BurySiblings == nil || *sc.BurySiblings

		if err := manager.AddSession(
			sc.Path, col, database, sc.AnswerControls,
			cardLimit, newCardLimit, deckFilter, burySiblings, fsrsCfg,
		); err != nil {
			return fmt.Errorf("setup session %q: %w", sc.Name, err)
		}
	}
	handlers.RegisterAPI(router, manager)

	router.HandleFunc("GET /drill", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFileFS(w, r, assets.FS, "index.html")
	})

	// GET / serves the same static shell as /drill; index.js's Solid Router
	// decides which screen to render based on the current path.
	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFileFS(w, r, assets.FS, "index.html")
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/static/{path...}", apis.Static(assets.FS, false))
		e.Router.Any("/{path...}", apis.WrapStdHandler(router))
		return e.Next()
	})
	fmt.Fprintf(out, "Listening on http://%s\n", addr)
	return apis.Serve(app, apis.ServeConfig{
		HttpAddr:        addr,
		ShowStartBanner: false,
	})
}

// computeAvgRetrieval returns the average retrievability of all cards in the
// collection that belong to the given deck (or all decks when deckFilter is
// empty), expressed as a percentage in [0, 100].
//
// New (unreviewed) cards contribute 0% to the average.
// Orphan cards are naturally excluded because only col.Cards is iterated.
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

// buildSessionList converts config sessions to the JSON-serialisable form,
// computing average retrievability for each session's deck.
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
