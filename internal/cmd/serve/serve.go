// Package serve implements the "serve" command, which runs a single HTTP server
// that hosts the index page and all drill sessions defined in the config file.
package serve

import (
	"fmt"

	"io/fs"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

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

	"github.com/sirupsen/logrus"
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

// deckSession describes one drill route derived from the collection: either
// the combined "All Decks" session (Deck == "") or a single deck's session.
type deckSession struct {
	Name string
	Path string
	Deck string
}

// nonAlphanumRe matches runs of characters that are not lowercase letters or digits.
var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)

// deckToPath converts a deck name to a clean URL path segment.
// Empty string maps to "" (the root drill route /drill/).
func deckToPath(deck string) string {
	if deck == "" {
		return ""
	}
	s := strings.ToLower(deck)
	s = nonAlphanumRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// deckSessions returns one session for "All Decks" plus one session per
// distinct deck name found in the collection (i.e. one per deck JSON file),
// sorted by deck name for a stable, deterministic order.
func deckSessions(col *collection.Collection) []deckSession {
	names := make(map[string]struct{})
	for _, c := range col.Cards {
		names[c.DeckName()] = struct{}{}
	}
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)

	sessions := []deckSession{{Name: "All Decks", Path: "", Deck: ""}}
	for _, name := range sorted {
		sessions = append(sessions, deckSession{Name: name, Path: deckToPath(name), Deck: name})
	}
	return sessions
}

// registerNewSessions registers a drill session for every deck in decks that
// does not already have one in manager, so an already-running deck's
// in-progress session is never reset. It returns the names of the decks that
// were newly registered.
func registerNewSessions(
	manager *handlers.Manager,
	decks []deckSession,
	col *collection.Collection,
	database *db.Database,
	fsrsCfg fsrs.FSRSConfig,
) ([]string, error) {
	var added []string
	for _, ds := range decks {
		if manager.Has(ds.Path) {
			continue
		}
		var deckFilter *string
		if ds.Deck != "" {
			df := ds.Deck
			deckFilter = &df
		}
		if err := manager.AddSession(
			ds.Path, col, database, "full",
			nil, nil, deckFilter, true, fsrsCfg,
		); err != nil {
			return nil, fmt.Errorf("setup session %q: %w", ds.Name, err)
		}
		added = append(added, ds.Name)
	}
	return added, nil
}

// Run opens the database and collection once, registers all drill routes, then
// starts listening. The database and collection are shared across all sessions.
func Run(app *pocketbase.PocketBase, cfg *config.Config) error {
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

	// Sessions are no longer hand-configured: one is created per deck found
	// in the collection, plus a combined "All Decks" session. All auto
	// sessions share the same defaults (no card limits, full answer
	// controls, sibling burial on).
	manager := handlers.NewManager()
	if _, err := registerNewSessions(manager, deckSessions(col), col, database, fsrsCfg); err != nil {
		return err
	}

	// assetsFS exposes just the "assets/" subdirectory that Vite's default
	// (unprefixed) base writes hashed JS/CSS bundles into, so they're served
	// at the conventional /assets/... URL instead of /static/assets/....
	assetsFS, err := fs.Sub(assets.FS, "assets")
	if err != nil {
		return fmt.Errorf("sub assets fs: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// GET /api/sessions reloads the collection from disk on every request
		// so decks/cards added or removed since startup are reflected.
		e.Router.GET("/api/sessions", func(re *core.RequestEvent) error {
			freshCol, err := collection.Load(cfg.Data.Root, database)
			if err != nil {
				logrus.WithError(err).Warn("collection load failed, using stale collection")
				freshCol = col
			}
			return re.JSON(http.StatusOK, buildSessionList(freshCol, database))
		}).Bind(apis.RequireSuperuserAuth())

		handlers.RegisterAPI(e.Router, manager, cfg.Data.Root)
		RegisterConnectionsAPI(e.Router, database, cfg.Data.Root, cfg.Data.HooksDir)
		RegisterMirrorAPI(e.Router, database, cfg.Data.Root, cfg.Data.HooksDir)
		RegisterHooksAPI(e.Router, cfg.Data.HooksDir)
		// POST /api/rescan re-scans the deck directory and registers a
		// session for any deck added since the server started, without
		// requiring a restart. Existing sessions are left untouched.
		e.Router.POST("/api/rescan", func(re *core.RequestEvent) error {
			freshCol, err := collection.Load(cfg.Data.Root, database)
			if err != nil {
				return re.BadRequestError("reload collection failed", err)
			}
			added, err := registerNewSessions(manager, deckSessions(freshCol), freshCol, database, fsrsCfg)
			if err != nil {
				return re.BadRequestError("register new sessions failed", err)
			}
			return re.JSON(http.StatusOK, map[string]any{"added": added})
		}).Bind(apis.RequireSuperuserAuth())

		e.Router.GET("/assets/{path...}", apis.Static(assetsFS, false))

		// Solid Router decides which screen to render client-side, so both
		// /drill and / serve the same static shell. This shell is left
		// unauthenticated on purpose: it's an empty HTML/JS bundle with no
		// data in it. Every route that actually returns collection data is
		// guarded above with RequireSuperuserAuth, so an unauthenticated
		// visitor only ever sees the login screen the SPA renders client-side.
		serveShell := func(re *core.RequestEvent) error {
			re.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeFileFS(re.Response, re.Request, assets.FS, "index.html")
			return nil
		}
		e.Router.GET("/drill", serveShell)
		e.Router.GET("/", serveShell)

		// Vite's public/ directory (favicon.svg etc.) is copied to the root
		// of the build output, so it's served directly rather than under
		// /assets/.
		e.Router.GET("/favicon.svg", func(re *core.RequestEvent) error {
			re.Response.Header().Set("Content-Type", "image/svg+xml")
			http.ServeFileFS(re.Response, re.Request, assets.FS, "favicon.svg")
			return nil
		})

		return e.Next()
	})

	logrus.WithField("addr", addr).Info("listening")
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

// buildSessionList converts the collection's decks to the JSON-serialisable
// form, computing average retrievability for each session's deck.
func buildSessionList(col *collection.Collection, database *db.Database) []sessionInfo {
	decks := deckSessions(col)
	list := make([]sessionInfo, 0, len(decks))
	for _, ds := range decks {
		drillURL := "/drill"
		if ds.Path != "" {
			drillURL = "/drill?deck=" + url.QueryEscape(ds.Path)
		}
		list = append(list, sessionInfo{
			Name:     ds.Name,
			Path:     ds.Path,
			DrillURL: drillURL,
			RetriPct: computeAvgRetrieval(col, database, ds.Deck),
		})
	}
	return list
}
