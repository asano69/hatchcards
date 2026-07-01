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
	return db, nil
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

// SaveSession / DeleteCard 以下:

func (db *Database) SaveSession(startedAt types.Timestamp, endedAt types.Timestamp, reviews []ReviewRecord) error {
	var sid string
	err := db.app.RunInTransaction(func(txApp core.App) error {
		if err := txApp.DB().NewQuery("insert into sessions (started_at, ended_at) values ({:s}, {:e}) returning id;").Bind(dbx.Params{"s": startedAt.String(), "e": endedAt.String()}).Row(&sid); err != nil {
			return errs.Newf("insert session: %v", err)
		}
		for _, r := range reviews {
			_, err := txApp.DB().NewQuery(`insert into reviews (session_id, card_hash, reviewed_at, grade, stability, difficulty, interval_raw, interval_days, due_date) values ({:sid}, {:hash}, {:at}, {:grade}, {:st}, {:diff}, {:raw}, {:days}, {:due});`).Bind(dbx.Params{"sid": sid, "hash": r.CardHash.Hex(), "at": r.ReviewedAt.String(), "grade": r.Grade.String(), "st": r.Stability, "diff": r.Difficulty, "raw": r.IntervalRaw, "days": r.IntervalDays, "due": r.DueDate.String()}).Execute()
			if err != nil {
				return errs.Newf("insert review: %v", err)
			}
		}
		return nil
	})
	return err
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
	var c int
	err := db.q("select count(*) from reviews where substr(reviewed_at, 1, 10)={:d};", dbx.Params{"d": date.String()}).Row(&c)
	if err != nil {
		return 0, errs.Newf("count reviews in date: %v", err)
	}
	return c, nil
}

func (db *Database) GetAllSessions() ([]SessionRow, error) {
	rows, err := db.q("select id, started_at, ended_at from sessions order by started_at;", nil).Rows()
	if err != nil {
		return nil, errs.Newf("query sessions: %v", err)
	}
	defer rows.Close()
	var out []SessionRow
	for rows.Next() {
		var id string
		var ss, es string
		if err := rows.Scan(&id, &ss, &es); err != nil {
			return nil, errs.Newf("scan session row: %v", err)
		}
		st, err := types.ParseTimestamp(ss)
		if err != nil {
			return nil, err
		}
		en, err := types.ParseTimestamp(es)
		if err != nil {
			return nil, err
		}
		out = append(out, SessionRow{id, st, en})
	}
	return out, rows.Err()
}

func (db *Database) GetReviewsForSession(sessionID string) ([]ReviewRow, error) {
	rows, err := db.q(`select id, card_hash, reviewed_at, grade, stability, difficulty, interval_raw, interval_days, due_date from reviews where session_id={:sid} order by reviewed_at;`, dbx.Params{"sid": sessionID}).Rows()
	if err != nil {
		return nil, errs.Newf("query reviews for session: %v", err)
	}
	defer rows.Close()
	var out []ReviewRow
	for rows.Next() {
		var id string
		var hh, at, gr, dueS string
		var st, diff, raw float64
		var days int64
		if err := rows.Scan(&id, &hh, &at, &gr, &st, &diff, &raw, &days, &dueS); err != nil {
			return nil, errs.Newf("scan review row: %v", err)
		}
		h, err := types.ParseCardHash(hh)
		if err != nil {
			return nil, err
		}
		ts, err := types.ParseTimestamp(at)
		if err != nil {
			return nil, err
		}
		grade, err := fsrs.ParseGrade(gr)
		if err != nil {
			return nil, errs.Newf("parse grade: %v", err)
		}
		due, err := types.ParseDate(dueS)
		if err != nil {
			return nil, err
		}
		out = append(out, ReviewRow{ReviewID: id, Data: ReviewRecord{CardHash: h, ReviewedAt: ts, Grade: grade, Stability: st, Difficulty: diff, IntervalRaw: raw, IntervalDays: days, DueDate: due}})
	}
	return out, rows.Err()
}
