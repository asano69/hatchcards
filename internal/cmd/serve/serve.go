// Package serve implements the "serve" command, which runs a single HTTP server
// that hosts the index page and all drill sessions defined in the config file.
package serve

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	// RetriPct is the average retrievability of the deck's cards, as a
	// percentage in [0, 100]. New (unreviewed) cards contribute 0%.
	// Orphan cards are excluded because only collection cards are considered.
	RetriPct float64 `json:"retri_pct"`
}

// newCardData holds template data for the /new card registration page.
type newCardData struct {
	Mode         string // "single" or "bulk"
	Decks        []string
	SelectedDeck string
	Question     string
	Answer       string
	BulkContent  string
	Success      bool
	SavedFile    string
	Error        string
}

// Run opens the database and collection once, registers all drill routes, then
// starts listening. The database and collection are shared across all sessions.
func Run(cfg *config.Config, staticDir string, out io.Writer) error {
	database, err := db.Open(cfg.Data.DB)
	if err != nil {
		return err
	}
	defer database.Close()

	col, err := collection.Load(cfg.Data.Root, database)
	if err != nil {
		return err
	}

	fsrsCfg := fsrs.FSRSConfig{
		TargetRecall: cfg.FSRS.TargetRecall,
		MinInterval:  cfg.FSRS.MinInterval,
		MaxInterval:  cfg.FSRS.MaxInterval,
	}

	router := mux.NewRouter()

	// Serve static assets under /static/.
	staticFS := http.FileServer(http.Dir(staticDir))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFS))

	// Serve KaTeX assets under /katex/ so that the URLs embedded in
	// katex.min.css (which reference /katex/fonts/...) resolve correctly.
	katexFS := http.FileServer(http.Dir(filepath.Join(staticDir, "katex")))
	router.PathPrefix("/katex/").Handler(http.StripPrefix("/katex/", katexFS))

	// Serve PWA and root-level files directly from the static directory.
	for _, f := range []string{"manifest.json", "sw.js", "favicon.svg"} {
		f := f
		router.HandleFunc("/"+f, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(staticDir, f))
		})
	}

	// Redirect /favicon.ico to the SVG icon so that browsers (e.g. Firefox)
	// that unconditionally request favicon.ico do not receive a 404.
	router.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/favicon.svg", http.StatusMovedPermanently)
	})

	// /api/sessions returns the session list as JSON for backward compatibility.
	router.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		sessionJSON, _ := json.Marshal(buildSessionList(cfg, col, database))
		w.Header().Set("Content-Type", "application/json")
		w.Write(sessionJSON)
	}).Methods(http.MethodGet)

	// /new — card registration form.
	router.HandleFunc("/new", newCardGetHandler(cfg, staticDir)).Methods(http.MethodGet)
	router.HandleFunc("/new", newCardPostHandler(cfg, staticDir, database)).Methods(http.MethodPost)

	// /delete — pass db and collectionRoot so each request reloads the collection.
	dh := &deleteHandler{
		col:            col,
		staticDir:      staticDir,
		db:             database,
		collectionRoot: cfg.Data.Root,
	}
	router.HandleFunc("/delete", dh.handleGet).Methods(http.MethodGet)
	router.HandleFunc("/delete", dh.handlePost).Methods(http.MethodPost)

	// Register drill routes with more specific (longer) paths first to avoid
	// the all-decks PathPrefix("/drill") intercepting named-deck requests.
	sessions := make([]config.SessionConfig, len(cfg.Sessions))
	copy(sessions, cfg.Sessions)
	sort.Slice(sessions, func(i, j int) bool {
		return len(sessions[i].Path) > len(sessions[j].Path)
	})
	for _, sc := range sessions {
		sc := sc
		// Use "/drill" (no trailing slash) for the all-decks session so that
		// the BasePath in templates becomes "/drill/" without a double slash.
		var mountPath string
		if sc.Path == "" {
			mountPath = "/drill"
		} else {
			mountPath = "/drill/" + sc.Path
		}
		if err := registerDrillRoute(router, sc, col, database, fsrsCfg, mountPath, staticDir); err != nil {
			return fmt.Errorf("setup session %q: %w", sc.Name, err)
		}
	}

	// Index page — server-rendered with Go templates, no client-side JS fetch needed.
	// The collection is reloaded on every request so retrievability and card counts
	// reflect cards added or deleted since the server started.
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		freshCol, err := collection.Load(cfg.Data.Root, database)
		if err != nil {
			freshCol = col
		}
		renderIndex(w, staticDir, buildSessionList(cfg, freshCol, database))
	}).Methods(http.MethodGet)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	fmt.Fprintf(out, "Listening on http://%s:%d\n", cfg.Server.Host, cfg.Server.Port)
	return http.Serve(ln, router)
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
func renderIndex(w http.ResponseWriter, staticDir string, sessions []sessionInfo) {
	tmpl, err := template.ParseFiles(
		filepath.Join(staticDir, "templates", "base.html"),
		filepath.Join(staticDir, "templates", "index.html"),
	)
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

	// Log session startup information.
	deckLabel := "all decks"
	if sc.FromDeck != "" {
		deckLabel = fmt.Sprintf("deck=%q", sc.FromDeck)
	}
	fmt.Printf("[session] name=%q path=%q %s cards=%d\n",
		sc.Name, mountPath, deckLabel, len(due))

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

// deckNames returns a deduplicated list of deck names from the config sessions.
// Sessions without a FromDeck (all-decks sessions) are skipped.
func deckNames(cfg *config.Config) []string {
	seen := make(map[string]struct{})
	var names []string
	for _, s := range cfg.Sessions {
		if s.FromDeck == "" {
			continue
		}
		if _, ok := seen[s.FromDeck]; !ok {
			seen[s.FromDeck] = struct{}{}
			names = append(names, s.FromDeck)
		}
	}
	return names
}

// newCardGetHandler renders the blank card registration form.
func newCardGetHandler(cfg *config.Config, staticDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decks := deckNames(cfg)
		selected := ""
		if len(decks) > 0 {
			selected = decks[0]
		}
		renderNewCard(w, staticDir, newCardData{
			Mode:         "single",
			Decks:        decks,
			SelectedDeck: selected,
		})
	}
}

// newCardPostHandler handles form submission and appends the card to the
// appropriate uploads/<deck>.md file under the collection root.
// After saving, the collection is reloaded to sync new cards into the database,
// which ensures the progress bar and delete page reflect the new card immediately.
func newCardPostHandler(cfg *config.Config, staticDir string, database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decks := deckNames(cfg)
		mode := r.FormValue("mode")
		if mode == "" {
			mode = "single"
		}
		deck := strings.TrimSpace(r.FormValue("deck"))

		data := newCardData{
			Mode:         mode,
			Decks:        decks,
			SelectedDeck: deck,
		}

		if deck == "" {
			data.Error = "Deck is required."
			renderNewCard(w, staticDir, data)
			return
		}

		uploadsDir := filepath.Join(cfg.Data.Root, "uploads")
		if err := os.MkdirAll(uploadsDir, 0755); err != nil {
			data.Error = fmt.Sprintf("Could not create uploads directory: %v", err)
			renderNewCard(w, staticDir, data)
			return
		}

		// Sanitize the deck name to produce a safe file name.
		safeName := strings.Map(func(r rune) rune {
			if strings.ContainsRune(`/\:*?"<>|`, r) {
				return '_'
			}
			return r
		}, deck)
		mdPath := filepath.Join(uploadsDir, safeName+".md")

		var entry string
		if mode == "bulk" {
			bulk := strings.TrimSpace(r.FormValue("bulk_content"))
			if bulk == "" {
				data.BulkContent = bulk
				data.Error = "Content is required."
				renderNewCard(w, staticDir, data)
				return
			}
			entry = bulk + "\n\n---\n\n"
		} else {
			question := strings.TrimSpace(r.FormValue("question"))
			answer := strings.TrimSpace(r.FormValue("answer"))
			data.Question = question
			data.Answer = answer
			if question == "" || answer == "" {
				data.Error = "Question and answer are required."
				renderNewCard(w, staticDir, data)
				return
			}
			entry = fmt.Sprintf("Q: %s\nA: %s\n\n---\n\n", question, answer)
		}

		f, err := os.OpenFile(mdPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			data.Error = fmt.Sprintf("Could not open file: %v", err)
			renderNewCard(w, staticDir, data)
			return
		}
		defer f.Close()

		if _, err := f.WriteString(entry); err != nil {
			data.Error = fmt.Sprintf("Could not write card: %v", err)
			renderNewCard(w, staticDir, data)
			return
		}
		f.Close()

		// Reload the collection to insert the new card into the database.
		// This ensures the progress bar and delete page reflect the new card
		// without requiring a server restart or a completed drill session.
		if _, err := collection.Load(cfg.Data.Root, database); err != nil {
			fmt.Printf("[new card] warning: reload collection: %v\n", err)
		}

		// On success, reset the form but keep the same deck and mode selected.
		renderNewCard(w, staticDir, newCardData{
			Mode:         mode,
			Decks:        decks,
			SelectedDeck: deck,
			Success:      true,
			SavedFile:    mdPath,
		})
	}
}

// renderNewCard renders the new.html template with the given data.
func renderNewCard(w http.ResponseWriter, staticDir string, data newCardData) {
	tmpl, err := template.ParseFiles(
		filepath.Join(staticDir, "templates", "base.html"),
		filepath.Join(staticDir, "templates", "new.html"),
	)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		fmt.Printf("new card render error: %v\n", err)
	}
}
