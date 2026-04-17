// Package serve implements the "serve" command, which runs a single HTTP server
// that hosts the index page and all drill sessions defined in the config file.
package serve

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"

	drillcache "github.com/asano69/hashcards/internal/cmd/drill/cache"
	"github.com/asano69/hashcards/internal/cmd/drill/handlers"
	drillstate "github.com/asano69/hashcards/internal/cmd/drill/state"
	"github.com/asano69/hashcards/internal/collection"
	"github.com/asano69/hashcards/internal/config"
	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/rng"
	"github.com/asano69/hashcards/internal/types"
)

// sessionInfo is the JSON representation returned by /api/sessions.
type sessionInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Run starts the HTTP server defined by cfg and blocks until it exits.
func Run(cfg *config.Config, staticDir string, out io.Writer) error {
	router := mux.NewRouter()

	// Serve all static assets (CSS, JS, KaTeX, templates) under /static/.
	staticFS := http.FileServer(http.Dir(staticDir))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFS))

	// /api/sessions returns the list of configured sessions as JSON.
	sessions := buildSessionList(cfg)
	sessionJSON, _ := json.Marshal(sessions)
	router.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(sessionJSON)
	}).Methods(http.MethodGet)

	// Register a drill route for each session in the config.
	for _, sc := range cfg.Sessions {
		sc := sc // capture loop variable
		mountPath := "/drill/" + sc.Path
		if err := registerDrillRoute(router, sc, mountPath, staticDir); err != nil {
			return fmt.Errorf("setup session %q: %w", sc.Name, err)
		}
	}

	// Serve the index page at /.
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
	}).Methods(http.MethodGet)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	fmt.Fprintf(out, "Listening on http://localhost:%d\n", cfg.Server.Port)
	return http.Serve(ln, router)
}

// buildSessionList converts config sessions to the JSON-serialisable form.
func buildSessionList(cfg *config.Config) []sessionInfo {
	list := make([]sessionInfo, 0, len(cfg.Sessions))
	for _, sc := range cfg.Sessions {
		list = append(list, sessionInfo{Name: sc.Name, Path: sc.Path})
	}
	return list
}

// registerDrillRoute opens the database, loads the collection, builds the
// initial session state, and attaches drill handlers to the router at
// /drill/<path>/.
func registerDrillRoute(router *mux.Router, sc config.SessionConfig, mountPath, staticDir string) error {
	database, err := db.Open(sc.DB)
	if err != nil {
		return err
	}

	col, err := collection.Load(sc.Root, database)
	if err != nil {
		database.Close()
		return err
	}

	due, err := col.DueToday(types.Today())
	if err != nil {
		database.Close()
		return err
	}

	// Resolve optional filter values.
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

	r := rng.FromSeed(uint64(time.Now().UnixNano()))
	sess := drillstate.New(due, r)
	htmlCache := drillcache.Build(due, mountPath)

	var mu sync.Mutex
	// done is buffered so the non-blocking send in handlers never panics.
	// The serve command does not read from this channel; the Shutdown action
	// is not exposed in the serve UI.
	done := make(chan struct{}, 1)

	sub := router.PathPrefix(mountPath).Subrouter()
	handlers.Register(sub, &mu, sess, htmlCache, database,
		done, staticDir, sc.Root, sc.AnswerControls, mountPath,
		cardLimit, newCardLimit, deckFilter, burySiblings)

	return nil
}
