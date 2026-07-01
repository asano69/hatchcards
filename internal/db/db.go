// Package db wraps the embedded PocketBase datastore that persists card performance and
// review history. All persistence is issued through this package; no other package
// touches the datastore directly.
package db

import (
	"database/sql"
	_ "embed"
	"errors"
	"os"

	"github.com/asano69/hashcards/internal/errs"
	"github.com/asano69/hashcards/internal/fsrs"
	"github.com/asano69/hashcards/internal/types"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed schema.sql
var schemaSQL string

type ReviewRecord struct {
	CardHash     types.CardHash
	ReviewedAt   types.Timestamp
	Grade        fsrs.Grade
	Stability    float64
	Difficulty   float64
	IntervalRaw  float64
	IntervalDays int64
	DueDate      types.Date
}
type SessionRow struct {
	SessionID string
	StartedAt types.Timestamp
	EndedAt   types.Timestamp
}
type ReviewRow struct {
	ReviewID string
	Data     ReviewRecord
}

type Database struct{ app *pocketbase.PocketBase }

// OpenScratch creates a Database backed by a fresh, disposable PocketBase
// instance in its own temporary directory. Each call returns an
// independent, empty database with no effect on any other Database.
// PocketBase always needs a directory on disk, so this is hashcards'
// equivalent of SQLite's ":memory:" mode.
func OpenScratch() (*Database, error) {
	dir, err := os.MkdirTemp("", "hashcards-pocketbase-*")
	if err != nil {
		return nil, errs.Newf("create temporary PocketBase data directory: %v", err)
	}
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: dir, HideStartBanner: true})
	if err := app.Bootstrap(); err != nil {
		return nil, errs.Newf("bootstrap PocketBase: %v", err)
	}
	return newDatabase(app)
}

// New wraps an already-bootstrapped PocketBase app and ensures the
// hashcards schema exists in it. app is expected to be the single instance
// shared by the whole CLI (see cmd/hashcards/main.go); its data directory is
// controlled by PocketBase's standard "--dir" flag, not by hashcards itself.
func New(app *pocketbase.PocketBase) (*Database, error) {
	return newDatabase(app)
}

// newDatabase wraps app in a Database and ensures the hashcards schema
// exists in it.
func newDatabase(app *pocketbase.PocketBase) (*Database, error) {
	db := &Database{app: app}
	if err := db.ensureSchema(); err != nil {
		return nil, err
	}
	db.registerHooks()
	return db, nil
}

// registerHooks wires up the PocketBase hooks that keep derived data in
// sync automatically, instead of relying on callers to remember to do it.
//
// Stage 1 of the reviews->cards hooks migration: whenever a "reviews" row
// is inserted, copy its FSRS output onto the matching "cards" row. This
// replaces the separate UpdateCardPerformance loop that the drill handler
// used to run after SaveSession.
func (db *Database) registerHooks() {
	db.app.OnRecordCreate("reviews").BindFunc(db.syncCardFromReview)
}

// syncCardFromReview updates the "cards" record referenced by a newly
// created "reviews" record with that review's FSRS output. It runs after
// e.Next() so it only applies once the review record has passed
// validation, and it uses e.App (not db.app) so the update participates in
// the same transaction as the review insert — if either write fails, both
// roll back together.
func (db *Database) syncCardFromReview(e *core.RecordEvent) error {
	if err := e.Next(); err != nil {
		return err
	}
	review := e.Record

	cardHash, err := types.ParseCardHash(review.GetString("card_hash"))
	if err != nil {
		return errs.Newf("sync card from review: %v", err)
	}

	card, err := e.App.FindFirstRecordByFilter(
		"cards", "card_hash = {:hash}", dbx.Params{"hash": cardHash.Hex()},
	)
	if err != nil {
		return errs.Newf("sync card from review: card %s not found: %v", cardHash.Hex(), err)
	}

	card.Set("last_reviewed_at", review.GetString("reviewed_at"))
	card.Set("stability", review.GetFloat("stability"))
	card.Set("difficulty", review.GetFloat("difficulty"))
	card.Set("interval_raw", review.GetFloat("interval_raw"))
	card.Set("interval_days", review.GetInt("interval_days"))
	card.Set("due_date", review.GetString("due_date"))
	card.Set("review_count", card.GetInt("review_count")+1)

	return e.App.Save(card)
}

func (db *Database) Close() error                        { return db.app.ResetBootstrapState() }
func (db *Database) App() *pocketbase.PocketBase         { return db.app }
func (db *Database) q(s string, p dbx.Params) *dbx.Query { return db.app.DB().NewQuery(s).Bind(p) }

func (db *Database) ensureSchema() error {
	if err := db.ensureCollection("cards", func(c *core.Collection) {
		c.Fields.Add(
			&core.TextField{Name: "card_hash", Required: true, Presentable: true},
			&core.TextField{Name: "added_at", Required: true},
			&core.TextField{Name: "last_reviewed_at"},
			&core.NumberField{Name: "stability"},
			&core.NumberField{Name: "difficulty"},
			&core.NumberField{Name: "interval_raw"},
			&core.NumberField{Name: "interval_days", OnlyInt: true},
			&core.TextField{Name: "due_date"},
			&core.NumberField{Name: "review_count", OnlyInt: true},
		)
		c.AddIndex("idx_cards_card_hash_unique", true, "card_hash", "")
	}); err != nil {
		return err
	}
	if err := db.ensureCollection("sessions", func(c *core.Collection) {
		c.Fields.Add(
			&core.TextField{Name: "started_at", Required: true},
			&core.TextField{Name: "ended_at", Required: true},
		)
		c.AddIndex("idx_sessions_started_at", false, "started_at", "")
	}); err != nil {
		return err
	}
	if err := db.ensureCollection("reviews", func(c *core.Collection) {
		c.Fields.Add(
			&core.TextField{Name: "session_id", Required: true},
			&core.TextField{Name: "card_hash", Required: true},
			&core.TextField{Name: "reviewed_at", Required: true},
			&core.TextField{Name: "grade", Required: true},
			&core.NumberField{Name: "stability", Required: true},
			&core.NumberField{Name: "difficulty", Required: true},
			&core.NumberField{Name: "interval_raw", Required: true},
			&core.NumberField{Name: "interval_days", OnlyInt: true, Required: true},
			&core.TextField{Name: "due_date", Required: true},
		)
		c.AddIndex("idx_reviews_session_id", false, "session_id", "")
		c.AddIndex("idx_reviews_card_hash", false, "card_hash", "")
	}); err != nil {
		return err
	}
	return nil
}

func (db *Database) ensureCollection(name string, configure func(*core.Collection)) error {
	if _, err := db.app.FindCollectionByNameOrId(name); err == nil {
		return nil
	}
	c := core.NewBaseCollection(name)
	configure(c)
	if err := db.app.Save(c); err != nil {
		return errs.Newf("ensure PocketBase collection %q: %v", name, err)
	}
	return nil
}

func (db *Database) probeSchemaExists() (bool, error) {
	var count int
	err := db.q("select count(*) from sqlite_master where type='table' AND name={:name};", dbx.Params{"name": "cards"}).Row(&count)
	if err != nil {
		return false, errs.Newf("probe schema: %v", err)
	}
	return count > 0, nil
}

// findCardRecord looks up the "cards" record with the given hash.
// It returns (nil, nil) when no such card exists.
func (db *Database) findCardRecord(cardHash types.CardHash) (*core.Record, error) {
	record, err := db.app.FindFirstRecordByFilter(
		"cards", "card_hash = {:hash}", dbx.Params{"hash": cardHash.Hex()},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return record, nil
}

func (db *Database) InsertCard(cardHash types.CardHash, addedAt types.Timestamp) error {
	existing, err := db.findCardRecord(cardHash)
	if err != nil {
		return errs.Newf("check card existence: %v", err)
	}
	if existing != nil {
		return errs.New("Card already exists")
	}

	collection, err := db.app.FindCollectionByNameOrId("cards")
	if err != nil {
		return errs.Newf("find cards collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Set("card_hash", cardHash.Hex())
	record.Set("added_at", addedAt.String())
	record.Set("review_count", 0)
	if err := db.app.Save(record); err != nil {
		return errs.Newf("insert card: %v", err)
	}
	return nil
}

func (db *Database) CardHashes() (map[types.CardHash]struct{}, error) {
	records, err := db.app.FindAllRecords("cards")
	if err != nil {
		return nil, errs.Newf("query card hashes: %v", err)
	}
	m := map[types.CardHash]struct{}{}
	for _, r := range records {
		h, err := types.ParseCardHash(r.GetString("card_hash"))
		if err != nil {
			return nil, err
		}
		m[h] = struct{}{}
	}
	return m, nil
}

// DueToday is unchanged in this stage (see Stage 4).
func (db *Database) DueToday(today types.Date) (map[types.CardHash]struct{}, error) {
	rows, err := db.q("select card_hash, due_date from cards;", nil).Rows()
	if err != nil {
		return nil, errs.Newf("query due today: %v", err)
	}
	defer rows.Close()
	due := map[types.CardHash]struct{}{}
	for rows.Next() {
		var hex string
		var dueStr sql.NullString
		if err := rows.Scan(&hex, &dueStr); err != nil {
			return nil, errs.Newf("scan due today row: %v", err)
		}
		h, err := types.ParseCardHash(hex)
		if err != nil {
			return nil, err
		}
		if !dueStr.Valid || dueStr.String == "" {
			due[h] = struct{}{}
			continue
		}
		dd, err := types.ParseDate(dueStr.String)
		if err != nil {
			return nil, err
		}
		if dd.LessOrEqual(today) {
			due[h] = struct{}{}
		}
	}
	return due, rows.Err()
}

func (db *Database) GetCardPerformanceOpt(cardHash types.CardHash) (*types.Performance, error) {
	record, err := db.findCardRecord(cardHash)
	if err != nil {
		return nil, errs.Newf("get card performance: %v", err)
	}
	if record == nil {
		return nil, nil
	}

	lra := record.GetString("last_reviewed_at")
	st := record.GetFloat("stability")
	diff := record.GetFloat("difficulty")
	raw := record.GetFloat("interval_raw")
	days := int64(record.GetInt("interval_days"))
	dd := record.GetString("due_date")
	count := record.GetInt("review_count")

	if lra == "" || st == 0 || diff == 0 || raw == 0 || dd == "" {
		p := types.NewCardPerformance()
		return &p, nil
	}
	ts, err := types.ParseTimestamp(lra)
	if err != nil {
		return nil, err
	}
	due, err := types.ParseDate(dd)
	if err != nil {
		return nil, err
	}
	p := types.ReviewedCardPerformance(types.ReviewedPerformance{
		LastReviewedAt: ts,
		Stability:      st,
		Difficulty:     diff,
		IntervalRaw:    raw,
		IntervalDays:   days,
		DueDate:        due,
		ReviewCount:    count,
	})
	return &p, nil
}

// GetCardPerformance is unchanged (wraps GetCardPerformanceOpt).
func (db *Database) GetCardPerformance(cardHash types.CardHash) (types.Performance, error) {
	p, err := db.GetCardPerformanceOpt(cardHash)
	if err != nil {
		return types.Performance{}, err
	}
	if p == nil {
		return types.Performance{}, errs.Newf("No performance data found for card with hash %s", cardHash)
	}
	return *p, nil
}

func (db *Database) UpdateCardPerformance(cardHash types.CardHash, perf types.Performance) error {
	record, err := db.findCardRecord(cardHash)
	if err != nil {
		return errs.Newf("find card: %v", err)
	}
	if record == nil {
		return errs.New("Card not found")
	}

	if rp := perf.Reviewed(); rp != nil {
		record.Set("last_reviewed_at", rp.LastReviewedAt.String())
		record.Set("stability", rp.Stability)
		record.Set("difficulty", rp.Difficulty)
		record.Set("interval_raw", rp.IntervalRaw)
		record.Set("interval_days", rp.IntervalDays)
		record.Set("due_date", rp.DueDate.String())
		record.Set("review_count", rp.ReviewCount)
	} else {
		record.Set("last_reviewed_at", "")
		record.Set("stability", 0)
		record.Set("difficulty", 0)
		record.Set("interval_raw", 0)
		record.Set("interval_days", 0)
		record.Set("due_date", "")
		record.Set("review_count", 0)
	}

	if err := db.app.Save(record); err != nil {
		return errs.Newf("update card performance: %v", err)
	}
	return nil
}

func (db *Database) SaveSession(startedAt types.Timestamp, endedAt types.Timestamp, reviews []ReviewRecord) error {
	sessionsCollection, err := db.app.FindCollectionByNameOrId("sessions")
	if err != nil {
		return errs.Newf("find sessions collection: %v", err)
	}
	reviewsCollection, err := db.app.FindCollectionByNameOrId("reviews")
	if err != nil {
		return errs.Newf("find reviews collection: %v", err)
	}

	return db.app.RunInTransaction(func(txApp core.App) error {
		session := core.NewRecord(sessionsCollection)
		session.Set("started_at", startedAt.String())
		session.Set("ended_at", endedAt.String())
		if err := txApp.Save(session); err != nil {
			return errs.Newf("insert session: %v", err)
		}

		for _, r := range reviews {
			review := core.NewRecord(reviewsCollection)
			review.Set("session_id", session.Id)
			review.Set("card_hash", r.CardHash.Hex())
			review.Set("reviewed_at", r.ReviewedAt.String())
			review.Set("grade", r.Grade.String())
			review.Set("stability", r.Stability)
			review.Set("difficulty", r.Difficulty)
			review.Set("interval_raw", r.IntervalRaw)
			review.Set("interval_days", r.IntervalDays)
			review.Set("due_date", r.DueDate.String())
			if err := txApp.Save(review); err != nil {
				return errs.Newf("insert review: %v", err)
			}
		}
		return nil
	})
}

func (db *Database) DeleteCard(cardHash types.CardHash) error {
	record, err := db.findCardRecord(cardHash)
	if err != nil {
		return errs.Newf("find card: %v", err)
	}
	if record == nil {
		return errs.New("Card not found")
	}

	// schema.sql's ON DELETE CASCADE does not apply to Record API deletes,
	// so associated reviews are removed explicitly.
	reviews, err := db.app.FindAllRecords("reviews", dbx.HashExp{"card_hash": cardHash.Hex()})
	if err != nil {
		return errs.Newf("find reviews for card: %v", err)
	}
	for _, r := range reviews {
		if err := db.app.Delete(r); err != nil {
			return errs.Newf("delete review: %v", err)
		}
	}

	if err := db.app.Delete(record); err != nil {
		return errs.Newf("delete card: %v", err)
	}
	return nil
}

func (db *Database) CountReviewsInDate(date types.Date) (int, error) {
	// Reviews on `date` have reviewed_at in [date 00:00:00.000, nextDate 00:00:00.000),
	// which works as a plain string range since the timestamp format is ISO8601-sortable.
	start := date.String() + "T00:00:00.000"
	end := types.NewDate(date.Time().AddDate(0, 0, 1)).String() + "T00:00:00.000"
	records, err := db.app.FindRecordsByFilter(
		"reviews", "reviewed_at >= {:start} && reviewed_at < {:end}", "", 0, 0,
		dbx.Params{"start": start, "end": end},
	)
	if err != nil {
		return 0, errs.Newf("count reviews in date: %v", err)
	}
	return len(records), nil
}

func (db *Database) GetAllSessions() ([]SessionRow, error) {
	records, err := db.app.FindRecordsByFilter("sessions", "", "+started_at", 0, 0)
	if err != nil {
		return nil, errs.Newf("query sessions: %v", err)
	}
	out := make([]SessionRow, 0, len(records))
	for _, r := range records {
		st, err := types.ParseTimestamp(r.GetString("started_at"))
		if err != nil {
			return nil, err
		}
		en, err := types.ParseTimestamp(r.GetString("ended_at"))
		if err != nil {
			return nil, err
		}
		out = append(out, SessionRow{SessionID: r.Id, StartedAt: st, EndedAt: en})
	}
	return out, nil
}

func (db *Database) GetReviewsForSession(sessionID string) ([]ReviewRow, error) {
	records, err := db.app.FindRecordsByFilter(
		"reviews", "session_id = {:sid}", "+reviewed_at", 0, 0, dbx.Params{"sid": sessionID},
	)
	if err != nil {
		return nil, errs.Newf("query reviews for session: %v", err)
	}
	out := make([]ReviewRow, 0, len(records))
	for _, r := range records {
		h, err := types.ParseCardHash(r.GetString("card_hash"))
		if err != nil {
			return nil, err
		}
		ts, err := types.ParseTimestamp(r.GetString("reviewed_at"))
		if err != nil {
			return nil, err
		}
		grade, err := fsrs.ParseGrade(r.GetString("grade"))
		if err != nil {
			return nil, errs.Newf("parse grade: %v", err)
		}
		due, err := types.ParseDate(r.GetString("due_date"))
		if err != nil {
			return nil, err
		}
		out = append(out, ReviewRow{
			ReviewID: r.Id,
			Data: ReviewRecord{
				CardHash:     h,
				ReviewedAt:   ts,
				Grade:        grade,
				Stability:    r.GetFloat("stability"),
				Difficulty:   r.GetFloat("difficulty"),
				IntervalRaw:  r.GetFloat("interval_raw"),
				IntervalDays: int64(r.GetInt("interval_days")),
				DueDate:      due,
			},
		})
	}
	return out, nil
}
