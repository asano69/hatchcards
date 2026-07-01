// Package serve implements the "serve" command, which runs a single HTTP server
// that hosts the index page and all drill sessions defined in the config file.
package serve

import (
	"encoding/json"
	"fmt"

	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/asano69/hashcards/internal/assets"
	drillcache "github.com/asano69/hashcards/internal/cmd/drill/cache"
	"github.com/asano69/hashcards/internal/cmd/drill/handlers"
	drillstate "github.com/asano69/hashcards/internal/cmd/drill/state"
	"github.com/asano69/hashcards/internal/collection"
	"github.com/asano69/hashcards/internal/config"
	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/fsrs"
	"github.com/asano69/hashcards/internal/rng"
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

	router.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/favicon.svg", http.StatusMovedPermanently)
	})

	// /api/sessions returns the session list as JSON for backward compatibility.
	router.HandleFunc("GET /api/sessions", func(w http.ResponseWriter, r *http.Request) {
		sessionJSON, _ := json.Marshal(buildSessionList(cfg, col, database))
		w.Header().Set("Content-Type", "application/json")
		w.Write(sessionJSON)
	})

	// Register drill routes with more specific (longer) paths first to avoid
	// the all-decks PathPrefix("/drill") intercepting named-deck requests.
	sessions := make([]config.SessionConfig, len(cfg.Sessions))
	copy(sessions, cfg.Sessions)
	sort.Slice(sessions, func(i, j int) bool {
		return len(sessions[i].Path) > len(sessions[j].Path)
	})
	for _, sc := range sessions {
		sc := sc
		var mountPath string
		if sc.Path == "" {
			mountPath = "/drill"
		} else {
			mountPath = "/drill/" + sc.Path
		}
		if err := registerDrillRoute(router, sc, col, database, fsrsCfg, mountPath); err != nil {
			return fmt.Errorf("setup session %q: %w", sc.Name, err)
		}
	}

	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		freshCol, err := collection.Load(cfg.Data.Root, database)
		if err != nil {
			freshCol = col
		}
		renderIndex(w, buildSessionList(cfg, freshCol, database))
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Serve embedded static assets under /static/ and /katex/ using
		// PocketBase's own static-file handler, instead of a hand-rolled
		// http.StripPrefix + http.FileServer wrapper.
		e.Router.GET("/static/{path...}", apis.Static(assets.FS, false))
		e.Router.GET("/katex/{path...}", apis.Static(assets.Sub("katex"), false))

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
		drillURL := "/drill/"
		if sc.Path != "" {
			drillURL = "/drill/" + sc.Path + "/"
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

// renderIndex renders the index page using Go templates with the session list
// injected server-side, eliminating the need for a client-side fetch.
func renderIndex(w http.ResponseWriter, sessions []sessionInfo) {
	tmpl, err := assets.ParseTemplate("index.html")
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct{ Sessions []sessionInfo }{sessions}
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		fmt.Printf("index render error: %v\n", err)
	}
}

// registerDrillRoute sets up a drill handler for one session.
func registerDrillRoute(
	router *http.ServeMux,
	sc config.SessionConfig,
	col *collection.Collection,
	database *db.Database,
	fsrsCfg fsrs.FSRSConfig,
	mountPath string,
) error {
	due, err := col.DueToday(types.Today())
	if err != nil {
		return err
	}

	var cardLimit *int
	if sc.CardLimit > 0 {
		cl := sc.CardLimit
		cardLimit = &cl
	}
	var newCardLimit *int
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

	due = handlers.FilterDue(due, cardLimit, newCardLimit, deckFilter)
	if burySiblings {
		due = handlers.BurySiblings(due)
	}

	// Log session startup information.
	deckLabel := "all decks"
	if sc.FromDeck != "" {
		deckLabel = fmt.Sprintf("deck=%q", sc.FromDeck)
	}
	fmt.Printf("[session] name=%q path=%q %s cards=%d\n",
		sc.Name, mountPath, deckLabel, len(due))

	// Shuffle before filtering so that new card selection is random rather
	// than always picking the same cards in hash order.
	r := rng.FromSeed(uint64(time.Now().UnixNano()))
	due = rng.Shuffle(due, r)
	due = handlers.FilterDue(due, cardLimit, newCardLimit, deckFilter)
	if burySiblings {
		due = handlers.BurySiblings(due)
	}

	sess := drillstate.New(due, r, fsrsCfg)
	htmlCache := drillcache.Build(due, mountPath)

	var mu sync.Mutex
	// done is buffered so the non-blocking send in handlers never panics.
	done := make(chan struct{}, 1)

	handlers.Register(
		router, &mu, sess, htmlCache, database,
		done, col.Root, sc.AnswerControls, mountPath,
		cardLimit, newCardLimit, deckFilter, burySiblings, fsrsCfg,
	)
	return nil
}
