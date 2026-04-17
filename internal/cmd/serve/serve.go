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
	"github.com/asano69/hashcards/internal/fsrs"
	"github.com/asano69/hashcards/internal/rng"
	"github.com/asano69/hashcards/internal/types"
)

// sessionInfo is the JSON representation returned by /api/sessions.
type sessionInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	DrillURL string `json:"drill_url"`
}

// Run opens the database and collection once, registers all drill routes, then
// starts listening. The database and collection are shared across all sessions,
// which avoids the "Card already exists" error that arose when each session
// opened its own connection and ran syncDB independently.
func Run(cfg *config.Config, staticDir string, out io.Writer) error {
	database, err := db.Open(cfg.Server.DB)
	if err != nil {
		return err
	}
	defer database.Close()

	col, err := collection.Load(cfg.Server.Root, database)
	if err != nil {
		return err
	}

	fsrsCfg := fsrs.FSRSConfig{
		TargetRecall: cfg.Server.TargetRecall,
		MinInterval:  cfg.Server.MinInterval,
		MaxInterval:  cfg.Server.MaxInterval,
	}

	router := mux.NewRouter()

	// Serve static assets under /static/.
	staticFS := http.FileServer(http.Dir(staticDir))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFS))

	// Serve PWA and root-level files directly from the static directory.
	for _, f := range []string{"manifest.json", "sw.js", "favicon.svg", "favicon.ico"} {
		f := f
		router.HandleFunc("/"+f, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(staticDir, f))
		})
	}

	// /api/sessions returns the session list as JSON.
	sessionInfos := buildSessionList(cfg)
	sessionJSON, _ := json.Marshal(sessionInfos)
	router.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(sessionJSON)
	}).Methods(http.MethodGet)

	// Register a drill route for each configured session.
	for _, sc := range cfg.Sessions {
		sc := sc
		mountPath := "/drill/" + sc.Path
		if err := registerDrillRoute(router, sc, col, database, fsrsCfg, mountPath, staticDir); err != nil {
			return fmt.Errorf("setup session %q: %w", sc.Name, err)
		}
	}

	// Index page.
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
	}).Methods(http.MethodGet)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	fmt.Fprintf(out, "Listening on http://%s:%d\n", cfg.Server.Host, cfg.Server.Port)
	return http.Serve(ln, router)
}

// buildSessionList converts config sessions to the JSON-serialisable form,
// including the full drill URL for each session.
func buildSessionList(cfg *config.Config) []sessionInfo {
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
		})
	}
	return list
}

// registerDrillRoute sets up a drill handler for one session using the
// pre-loaded collection and shared database connection.
func registerDrillRoute(
	router *mux.Router,
	sc config.SessionConfig,
	col *collection.Collection,
	database *db.Database,
	fsrsCfg fsrs.FSRSConfig,
	mountPath string,
	staticDir string,
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

	r := rng.FromSeed(uint64(time.Now().UnixNano()))
	sess := drillstate.New(due, r, fsrsCfg)
	htmlCache := drillcache.Build(due, mountPath)

	var mu sync.Mutex
	// done is buffered so the non-blocking send in handlers never panics.
	done := make(chan struct{}, 1)

	sub := router.PathPrefix(mountPath).Subrouter()
	handlers.Register(
		sub, &mu, sess, htmlCache, database,
		done, staticDir, col.Root, sc.AnswerControls, mountPath,
		cardLimit, newCardLimit, deckFilter, burySiblings, fsrsCfg,
	)
	return nil
}
