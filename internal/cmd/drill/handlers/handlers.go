// Package handlers implements the JSON drill API.
//
// The drill session state (queue, completed reviews, revealed flag) lives
// server-side in a state.State per configured deck, same as before. Only the
// transport changed: instead of full-page form POSTs returning rendered
// HTML, the client calls a small JSON API and re-renders itself.
//
// Endpoints:
//
//	GET  /api/drill/state?deck=<path>   — current session state as JSON
//	POST /api/drill/action?deck=<path>  — apply an action, returns new state
//	GET  /drill/file/{path...}          — media files (deck-independent)
package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	drillcache "github.com/asano69/hashcards/internal/cmd/drill/cache"
	"github.com/asano69/hashcards/internal/cmd/drill/state"
	"github.com/asano69/hashcards/internal/collection"
	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/fsrs"
	"github.com/asano69/hashcards/internal/rng"
	"github.com/asano69/hashcards/internal/types"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/sirupsen/logrus"
)

// fileMountBase is the fixed URL prefix used to rewrite <img>/<audio> src
// attributes during rendering. It no longer varies per deck since file
// serving is now a single shared route.
const fileMountBase = "/drill"

// shortHash returns the first 7 characters of a card hash hex string.
func shortHash(h types.CardHash) string {
	s := h.Hex()
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

// FilterDue applies card-limit, new-card-limit, and deck-filter to a due list.
func FilterDue(due []collection.DueCard, cardLimit *int, newCardLimit *int, deckFilter *string) []collection.DueCard {
	if deckFilter != nil {
		filtered := due[:0]
		for _, dc := range due {
			if dc.Card.DeckName() == *deckFilter {
				filtered = append(filtered, dc)
			}
		}
		due = filtered
	}
	if cardLimit != nil && len(due) > *cardLimit {
		due = due[:*cardLimit]
	}
	if newCardLimit != nil {
		limit := *newCardLimit
		var result []collection.DueCard
		newCount := 0
		for _, dc := range due {
			if dc.Performance.IsNew() {
				if newCount >= limit {
					continue
				}
				newCount++
			}
			result = append(result, dc)
		}
		due = result
	}
	return due
}

// BurySiblings removes all but the first cloze card sharing the same family hash.
func BurySiblings(due []collection.DueCard) []collection.DueCard {
	seen := make(map[types.CardHash]struct{})
	result := make([]collection.DueCard, 0, len(due))
	for _, dc := range due {
		fh := dc.Card.FamilyHash()
		if fh != nil {
			if _, ok := seen[*fh]; ok {
				continue
			}
			seen[*fh] = struct{}{}
		}
		result = append(result, dc)
	}
	return result
}

// -------------------------------------------------------------------------
// Manager — one handler per configured deck, keyed by URL path segment.
// -------------------------------------------------------------------------

// Manager dispatches drill state/action requests to the right per-deck
// handler. "" is the key for the "all decks" session.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*handler
}

func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*handler)}
}

// AddSession creates and registers a handler for one configured deck.
func (m *Manager) AddSession(
	deckKey string,
	col *collection.Collection,
	database *db.Database,
	answerControls string,
	cardLimit *int,
	newCardLimit *int,
	deckFilter *string,
	burySiblings bool,
	fsrsCfg fsrs.FSRSConfig,
) error {
	due, err := col.DueToday(types.Today())
	if err != nil {
		return err
	}
	due = FilterDue(due, cardLimit, newCardLimit, deckFilter)
	if burySiblings {
		due = BurySiblings(due)
	}

	// Shuffle before filtering so new-card selection is random rather than
	// always picking the same cards in hash order.
	r := rng.FromSeed(uint64(time.Now().UnixNano()))
	due = rng.Shuffle(due, r)
	due = FilterDue(due, cardLimit, newCardLimit, deckFilter)
	if burySiblings {
		due = BurySiblings(due)
	}

	h := &handler{
		mu:             &sync.Mutex{},
		sess:           state.New(due, r, fsrsCfg),
		cache:          drillcache.Build(due, col.Root, fileMountBase),
		db:             database,
		col:            col,
		macros:         loadMacros(filepath.Join(col.Root, "macros.tex")),
		answerControls: answerControls,
		cardLimit:      cardLimit,
		newCardLimit:   newCardLimit,
		deckFilter:     deckFilter,
		burySiblingsOn: burySiblings,
		fsrsCfg:        fsrsCfg,
	}

	m.mu.Lock()
	m.sessions[deckKey] = h
	m.mu.Unlock()

	return nil
}

func (m *Manager) get(deckKey string) (*handler, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.sessions[deckKey]
	return h, ok
}

// Has reports whether a session is already registered for deckKey. It lets
// callers (e.g. the rescan handler) add sessions for newly discovered decks
// without disturbing decks that are already registered.
func (m *Manager) Has(deckKey string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.sessions[deckKey]
	return ok
}

// RegisterAPI attaches the drill API and the shared media route directly to
// PocketBase's router, so responses go through the same JSON/error helpers
// as the rest of the app instead of a separate net/http.ServeMux.
//
// Every route here requires a valid superuser session, except the media
// route below: media is referenced via plain <img>/<audio> src attributes
// in server-rendered card HTML, so the browser has no way to attach an
// Authorization header to those requests. Since this route only serves
// files already referenced from cards (no secrets), it is left unauthenticated.
func RegisterAPI(r *router.Router[*core.RequestEvent], m *Manager, collectionRoot string) {
	r.GET("/api/drill/state", m.handleState).Bind(apis.RequireSuperuserAuth())
	r.POST("/api/drill/action", m.handleAction).Bind(apis.RequireSuperuserAuth())
	// apis.Static already handles Content-Type detection, path traversal
	// guards, and file existence checks, so there is no need to hand-roll
	// a MIME table or an os.Stat/http.ServeFile handler here.
	r.GET("/drill/file/{path...}", apis.Static(os.DirFS(collectionRoot), false))
}

func (m *Manager) handleState(e *core.RequestEvent) error {
	h, ok := m.get(e.Request.URL.Query().Get("deck"))
	if !ok {
		return e.NotFoundError("unknown deck", nil)
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	// If no reviews have started yet, reload due cards from disk so that
	// cards added or deleted since the session was created are reflected.
	if len(h.sess.Done) == 0 && !h.sess.Revealed && !h.sess.IsFinished() {
		h.resetSession()
	}
	return e.JSON(http.StatusOK, h.stateResponse())
}

func (m *Manager) handleAction(e *core.RequestEvent) error {
	h, ok := m.get(e.Request.URL.Query().Get("deck"))
	if !ok {
		return e.NotFoundError("unknown deck", nil)
	}

	var body struct {
		Action string `json:"action"`
	}
	if err := e.BindBody(&body); err != nil {
		return e.BadRequestError("invalid request body", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.applyAction(body.Action)
	return e.JSON(http.StatusOK, h.stateResponse())
}

// -------------------------------------------------------------------------
// Per-deck handler
// -------------------------------------------------------------------------

type handler struct {
	mu             *sync.Mutex
	sess           *state.State
	cache          *drillcache.Cache
	db             *db.Database
	col            *collection.Collection
	sessionSaved   bool
	endedAt        time.Time
	macros         [][2]string
	answerControls string
	cardLimit      *int
	newCardLimit   *int
	deckFilter     *string
	burySiblingsOn bool
	fsrsCfg        fsrs.FSRSConfig
}

// applyAction mutates session state in response to a client action. It
// replaces the old postRoot switch-case; the only behavioural difference is
// that no HTTP redirect happens here — the client decides what to do next
// based on the returned status.
func (h *handler) applyAction(action string) {
	switch action {
	case "Reveal":
		if !h.sess.IsFinished() {
			h.sess.Reveal()
		}

	case "Undo":
		h.sess.Undo()

	case "End":
		h.finishSession()

	case "Reset":
		// Discard the current session without saving and reload fresh due
		// cards. The client is responsible for navigating back to "/".
		h.resetSession()

	case "ReturnHome":
		h.finishSession()
		h.resetSession()

	case "Forgot", "Hard", "Good", "Easy":
		if !h.sess.IsFinished() && h.sess.Revealed {
			grade := actionToGrade(action)

			var beforeStability, beforeDifficulty float64
			var beforeInterval int64
			if dc, ok := h.sess.Current(); ok {
				if rp := dc.Performance.Reviewed(); rp != nil {
					beforeStability = rp.Stability
					beforeDifficulty = rp.Difficulty
					beforeInterval = rp.IntervalDays
				}
			}

			newPerf, ok := h.sess.Grade(grade)
			if ok {
				last := h.sess.Done[len(h.sess.Done)-1]
				logrus.WithFields(logrus.Fields{
					"hash":                 shortHash(last.Card.Hash()),
					"grade":                action,
					"stability_before":     beforeStability,
					"stability_after":      newPerf.Stability,
					"difficulty_before":    beforeDifficulty,
					"difficulty_after":     newPerf.Difficulty,
					"interval_before_days": beforeInterval,
					"interval_after_days":  newPerf.IntervalDays,
					"due":                  newPerf.DueDate.String(),
				}).Info("card graded")
			}
			if h.sess.Remaining() == 0 {
				h.finishSession()
			}
		}
	}
}

func (h *handler) finishSession() {
	if h.sessionSaved || h.sess.Total() == 0 {
		return
	}
	h.sess.Finish()
	h.sessionSaved = true
	h.endedAt = time.Now()

	// Card performance (stability, difficulty, due date, ...) is no longer
	// updated here: the "reviews" OnRecordCreate hook in internal/db keeps
	// each card's cached performance in sync as every review row is
	// inserted, inside the same transaction as SaveSession below.
	if err := h.db.SaveSession(
		h.sess.StartedAt,
		types.NewTimestamp(h.endedAt),
		h.sess.ToReviewRecords(),
	); err != nil {
		logrus.WithError(err).Warn("save session failed")
		return
	}
	logrus.WithField("reviews", len(h.sess.Done)).Info("session saved")
}

// resetSession loads fresh due cards and replaces the current session state.
// Nothing from the current session is written to the database.
func (h *handler) resetSession() {
	col, err := collection.Load(h.col.Root, h.db)
	if err != nil {
		logrus.WithError(err).Warn("reset session: load collection failed")
		return
	}
	h.col = col

	due, err := col.DueToday(types.Today())
	if err != nil {
		logrus.WithError(err).Warn("reset session: due today failed")
		return
	}
	due = FilterDue(due, h.cardLimit, h.newCardLimit, h.deckFilter)
	if h.burySiblingsOn {
		due = BurySiblings(due)
	}

	r := rng.FromSeed(uint64(time.Now().UnixNano()))
	due = rng.Shuffle(due, r)
	due = FilterDue(due, h.cardLimit, h.newCardLimit, h.deckFilter)
	if h.burySiblingsOn {
		due = BurySiblings(due)
	}
	h.sess = state.New(due, r, h.fsrsCfg)

	h.cache = drillcache.Build(due, col.Root, fileMountBase)
	h.sessionSaved = false
	h.endedAt = time.Time{}

	logrus.WithField("cards_due", len(due)).Info("session reset")
}

func actionToGrade(action string) fsrs.Grade {
	switch action {
	case "Forgot":
		return fsrs.GradeForgot
	case "Hard":
		return fsrs.GradeHard
	case "Good":
		return fsrs.GradeGood
	case "Easy":
		return fsrs.GradeEasy
	default:
		return fsrs.GradeGood
	}
}

// loadMacros reads a macros.tex file and returns (name, definition) pairs,
// skipping comment lines that start with '%'.
func loadMacros(path string) [][2]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var macros [][2]string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "%") {
			continue
		}
		idx := strings.IndexByte(line, ' ')
		if idx < 0 {
			continue
		}
		name := line[:idx]
		definition := strings.TrimSpace(line[idx+1:])
		if name != "" && definition != "" {
			macros = append(macros, [2]string{name, definition})
		}
	}
	return macros
}

func macrosMap(pairs [][2]string) map[string]string {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		m[p[0]] = p[1]
	}
	return m
}

// -------------------------------------------------------------------------
// JSON response shapes
// -------------------------------------------------------------------------

type stateResponse struct {
	Status string         `json:"status"` // "no_cards" | "card" | "done"
	Card   *cardStateJSON `json:"card,omitempty"`
	Done   *doneStateJSON `json:"done,omitempty"`
}

type cardStateJSON struct {
	DeckName       string            `json:"deckName"`
	Front          string            `json:"front"`
	Back           string            `json:"back"`
	Revealed       bool              `json:"revealed"`
	CanUndo        bool              `json:"canUndo"`
	ReviewedCount  int               `json:"reviewedCount"`
	Total          int               `json:"total"`
	ProgressPct    int               `json:"progressPct"`
	AnswerControls string            `json:"answerControls"`
	Macros         map[string]string `json:"macros"`
	// LastReviewedAt is the previous review timestamp, or "" for a new card.
	LastReviewedAt string `json:"lastReviewedAt"`
}

type doneStateJSON struct {
	Reviewed    int   `json:"reviewed"`
	Total       int   `json:"total"`
	DurationSec int64 `json:"durationSec"`
}

func (h *handler) stateResponse() stateResponse {
	if h.sess.Total() == 0 {
		return stateResponse{Status: "no_cards"}
	}
	if h.sess.IsFinished() {
		var durationSec int64
		if !h.endedAt.IsZero() {
			durationSec = int64(h.endedAt.Sub(h.sess.StartedAt.Time()).Seconds())
		}
		return stateResponse{
			Status: "done",
			Done: &doneStateJSON{
				Reviewed:    h.sess.ReviewedCount(),
				Total:       h.sess.Total(),
				DurationSec: durationSec,
			},
		}
	}

	dc, _ := h.sess.Current()
	entry, ok := h.cache.Get(dc.Card.Hash())
	if !ok {
		// Card missing from the pre-render cache — fall back rather than
		// serving a broken response.
		return stateResponse{Status: "no_cards"}
	}

	// New cards have no prior review, so LastReviewedAt stays empty.
	var lastReviewedAt string
	if rp := dc.Performance.Reviewed(); rp != nil {
		lastReviewedAt = rp.LastReviewedAt.String()
	}

	total := h.sess.Total()
	done := h.sess.ReviewedCount()
	percent := 0
	if total > 0 {
		percent = done * 100 / total
	}
	return stateResponse{
		Status: "card",
		Card: &cardStateJSON{
			DeckName:       dc.Card.DeckName(),
			Front:          entry.Front,
			Back:           entry.Back,
			Revealed:       h.sess.Revealed,
			CanUndo:        len(h.sess.Done) > 0,
			ReviewedCount:  done,
			Total:          total,
			ProgressPct:    percent,
			AnswerControls: h.answerControls,
			Macros:         macrosMap(h.macros),
			LastReviewedAt: lastReviewedAt,
		},
	}
}
