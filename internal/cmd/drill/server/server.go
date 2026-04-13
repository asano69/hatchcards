// Package server wires together the drill HTTP server: it creates the session
// state, builds the HTML cache, registers routes, and runs the listener.
package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/asano69/hashcards/internal/cmd/drill/cache"
	"github.com/asano69/hashcards/internal/cmd/drill/handlers"
	"github.com/asano69/hashcards/internal/cmd/drill/state"
	"github.com/asano69/hashcards/internal/collection"
	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/rng"
	"github.com/asano69/hashcards/internal/types"
)

// AnswerControls specifies which grade buttons are shown during a drill session.
type AnswerControls string

const (
	// AnswerControlsFull shows all four buttons: Forgot / Hard / Good / Easy.
	AnswerControlsFull AnswerControls = "full"
	// AnswerControlsBinary shows only two buttons: Forgot / Good.
	AnswerControlsBinary AnswerControls = "binary"
)

// Config holds the parameters needed to start the drill server.
type Config struct {
	// Root is the collection root directory.
	Root string
	// DBPath is the path to the performance database.
	DBPath string
	// Port is the TCP port to listen on (e.g. 8080).
	Port int
	// Seed is the PRNG seed used to shuffle the card queue.
	Seed uint64
	// StaticDir is the path to the directory containing static assets.
	StaticDir string
	// Out receives informational messages (e.g. "Listening on :8080").
	Out io.Writer

	// CardLimit, when non-nil, caps the total number of cards in the session.
	CardLimit *int
	// NewCardLimit, when non-nil, caps the number of new (never reviewed) cards.
	NewCardLimit *int
	// DeckFilter, when non-nil, restricts the session to cards from the named deck.
	DeckFilter *string
	// AnswerControls selects which grade buttons are shown. Defaults to full.
	AnswerControls AnswerControls
	// BurySiblings removes all but the first sibling cloze card from the queue.
	BurySiblings bool
}

// Run starts the drill server and blocks until the session is complete or the
// context is cancelled. It persists completed reviews before returning.
func Run(ctx context.Context, cfg Config) error {
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer database.Close()

	col, err := collection.Load(cfg.Root, database)
	if err != nil {
		return err
	}

	due, err := col.DueToday(types.Today())
	if err != nil {
		return err
	}

	due = filterDeck(due, cfg)

	if cfg.BurySiblings {
		due = burySiblings(due)
	}

	if len(due) == 0 {
		fmt.Fprintln(cfg.Out, "No cards due today.")
		return nil
	}

	r := rng.FromSeed(cfg.Seed)
	sess := state.New(due, r)
	htmlCache := cache.Build(due)

	// shared is accessed from HTTP handlers; the mutex serialises all access.
	var mu sync.Mutex

	// done is closed by the POST handler when the last card is reviewed.
	done := make(chan struct{})

	// Default to full answer controls when not specified.
	ac := cfg.AnswerControls
	if ac == "" {
		ac = AnswerControlsFull
	}

	router := mux.NewRouter()
	handlers.Register(router, &mu, sess, htmlCache, database, done, cfg.StaticDir, cfg.Root, string(ac))

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{Addr: addr, Handler: router}

	// Start listening before printing the URL so the port is bound.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	fmt.Fprintf(cfg.Out, "Listening on http://localhost:%d\n", cfg.Port)

	// Serve in a goroutine so we can wait for session completion or ctx cancel.
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ln)
	}()

	// Wait for the session to finish, the context to be cancelled, or a serve error.
	select {
	case <-done:
		fmt.Fprintln(cfg.Out, "Session complete.")
	case <-ctx.Done():
	case err := <-serveErr:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	}

	// Graceful shutdown with a short timeout.
	shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx) //nolint:errcheck

	return nil
}

// filterDeck applies card-limit, new-card-limit, and deck-filter from cfg,
// matching the Rust filter_deck() function in server.rs.
func filterDeck(due []collection.DueCard, cfg Config) []collection.DueCard {
	// Apply deck filter.
	if cfg.DeckFilter != nil {
		filtered := due[:0]
		for _, dc := range due {
			if dc.Card.DeckName() == *cfg.DeckFilter {
				filtered = append(filtered, dc)
			}
		}
		due = filtered
	}

	// Apply total card limit.
	if cfg.CardLimit != nil && len(due) > *cfg.CardLimit {
		due = due[:*cfg.CardLimit]
	}

	// Apply new card limit.
	if cfg.NewCardLimit != nil {
		limit := *cfg.NewCardLimit
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

// burySiblings removes all but the first cloze card sharing the same family
// hash. This matches the Rust implementation's bury_siblings() in server.rs.
func burySiblings(due []collection.DueCard) []collection.DueCard {
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
