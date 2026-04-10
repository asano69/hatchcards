// Package db wraps the SQLite database that persists card performance and
// review history. All SQL is issued through this package; no other package
// touches the database directly.
package db

import (
	"database/sql"
	_ "embed"

	"github.com/asano69/hashcards/internal/errs"
	"github.com/asano69/hashcards/internal/fsrs"
	"github.com/asano69/hashcards/internal/types"

	_ "modernc.org/sqlite" // register the "sqlite" driver
)

//go:embed schema.sql
var schemaSQL string

// ReviewRecord holds the data for a single card review to be persisted.
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

// SessionRow is a row read from the sessions table.
type SessionRow struct {
	SessionID int64
	StartedAt types.Timestamp
	EndedAt   types.Timestamp
}

// ReviewRow is a row read from the reviews table, including its session-level data.
type ReviewRow struct {
	ReviewID int64
	Data     ReviewRecord
}

// Database wraps a *sql.DB and exposes the operations used by the rest of the
// application. It is safe to use from multiple goroutines provided the
// underlying SQLite driver serialises writes, which modernc.org/sqlite does.
type Database struct {
	conn *sql.DB
}

// Open opens the SQLite database at the given path, enabling foreign keys and
// initialising the schema if it does not yet exist.
func Open(path string) (*Database, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, errs.Newf("open database: %v", err)
	}

	// Enable foreign key enforcement for this connection.
	if _, err := conn.Exec("pragma foreign_keys = on;"); err != nil {
		conn.Close()
		return nil, errs.Newf("enable foreign keys: %v", err)
	}

	db := &Database{conn: conn}

	// Create the schema if the cards table does not yet exist.
	exists, err := db.probeSchemaExists()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if !exists {
		if _, err := conn.Exec(schemaSQL); err != nil {
			conn.Close()
			return nil, errs.Newf("initialise schema: %v", err)
		}
	}

	return db, nil
}

// Close releases the underlying database connection.
func (db *Database) Close() error {
	return db.conn.Close()
}

// probeSchemaExists returns true when the cards table already exists in the database.
func (db *Database) probeSchemaExists() (bool, error) {
	var count int
	err := db.conn.QueryRow(
		"select count(*) from sqlite_master where type='table' AND name=?;",
		"cards",
	).Scan(&count)
	if err != nil {
		return false, errs.Newf("probe schema: %v", err)
	}
	return count > 0, nil
}

// InsertCard adds a new card to the database with no review history.
// Returns an error if the card already exists.
func (db *Database) InsertCard(cardHash types.CardHash, addedAt types.Timestamp) error {
	exists, err := db.cardExists(cardHash)
	if err != nil {
		return err
	}
	if exists {
		return errs.New("Card already exists")
	}
	_, err = db.conn.Exec(
		"insert into cards (card_hash, added_at, review_count) values (?, ?, 0);",
		cardHash.Hex(),
		addedAt.String(),
	)
	if err != nil {
		return errs.Newf("insert card: %v", err)
	}
	return nil
}

// CardHashes returns the set of all card hashes currently stored in the database.
func (db *Database) CardHashes() (map[types.CardHash]struct{}, error) {
	rows, err := db.conn.Query("select card_hash from cards;")
	if err != nil {
		return nil, errs.Newf("query card hashes: %v", err)
	}
	defer rows.Close()

	hashes := make(map[types.CardHash]struct{})
	for rows.Next() {
		var hexStr string
		if err := rows.Scan(&hexStr); err != nil {
			return nil, errs.Newf("scan card hash: %v", err)
		}
		h, err := types.ParseCardHash(hexStr)
		if err != nil {
			return nil, err
		}
		hashes[h] = struct{}{}
	}
	return hashes, rows.Err()
}

// DueToday returns the set of card hashes that are due for review on or before today.
// Cards that have never been reviewed are always included.
func (db *Database) DueToday(today types.Date) (map[types.CardHash]struct{}, error) {
	rows, err := db.conn.Query("select card_hash, due_date from cards;")
	if err != nil {
		return nil, errs.Newf("query due today: %v", err)
	}
	defer rows.Close()

	due := make(map[types.CardHash]struct{})
	for rows.Next() {
		var hexStr string
		var dueDateStr *string
		if err := rows.Scan(&hexStr, &dueDateStr); err != nil {
			return nil, errs.Newf("scan due today row: %v", err)
		}
		h, err := types.ParseCardHash(hexStr)
		if err != nil {
			return nil, err
		}
		// A nil due_date means the card has never been reviewed, so it is due.
		if dueDateStr == nil {
			due[h] = struct{}{}
			continue
		}
		dueDate, err := types.ParseDate(*dueDateStr)
		if err != nil {
			return nil, err
		}
		if dueDate.LessOrEqual(today) {
			due[h] = struct{}{}
		}
	}
	return due, rows.Err()
}

// GetCardPerformanceOpt returns the performance for a card, or nil when the
// card does not exist in the database.
func (db *Database) GetCardPerformanceOpt(cardHash types.CardHash) (*types.Performance, error) {
	var (
		lastReviewedAtStr *string
		stability         *float64
		difficulty        *float64
		intervalRaw       *float64
		intervalDays      *int64
		dueDateStr        *string
		reviewCount       int
	)
	err := db.conn.QueryRow(
		`select last_reviewed_at, stability, difficulty, interval_raw,
		        interval_days, due_date, review_count
		 from cards where card_hash = ?;`,
		cardHash.Hex(),
	).Scan(
		&lastReviewedAtStr,
		&stability,
		&difficulty,
		&intervalRaw,
		&intervalDays,
		&dueDateStr,
		&reviewCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errs.Newf("get card performance: %v", err)
	}

	// All scheduling columns are set together; if any is null the card is new.
	if lastReviewedAtStr == nil || stability == nil || difficulty == nil ||
		intervalRaw == nil || intervalDays == nil || dueDateStr == nil {
		p := types.NewCardPerformance()
		return &p, nil
	}

	lastReviewedAt, err := types.ParseTimestamp(*lastReviewedAtStr)
	if err != nil {
		return nil, err
	}
	dueDate, err := types.ParseDate(*dueDateStr)
	if err != nil {
		return nil, err
	}

	p := types.ReviewedCardPerformance(types.ReviewedPerformance{
		LastReviewedAt: lastReviewedAt,
		Stability:      *stability,
		Difficulty:     *difficulty,
		IntervalRaw:    *intervalRaw,
		IntervalDays:   *intervalDays,
		DueDate:        dueDate,
		ReviewCount:    reviewCount,
	})
	return &p, nil
}

// GetCardPerformance returns the performance for a card.
// Returns an error when the card does not exist in the database.
func (db *Database) GetCardPerformance(cardHash types.CardHash) (types.Performance, error) {
	p, err := db.GetCardPerformanceOpt(cardHash)
	if err != nil {
		return types.Performance{}, err
	}
	if p == nil {
		return types.Performance{}, errs.Newf(
			"No performance data found for card with hash %s", cardHash,
		)
	}
	return *p, nil
}

// UpdateCardPerformance persists updated FSRS scheduling data for an existing card.
// Returns an error when the card does not exist.
func (db *Database) UpdateCardPerformance(cardHash types.CardHash, perf types.Performance) error {
	exists, err := db.cardExists(cardHash)
	if err != nil {
		return err
	}
	if !exists {
		return errs.New("Card not found")
	}

	var (
		lastReviewedAt *string
		stability      *float64
		difficulty     *float64
		intervalRaw    *float64
		intervalDays   *int64
		dueDate        *string
		reviewCount    int
	)

	if rp := perf.Reviewed(); rp != nil {
		lra := rp.LastReviewedAt.String()
		dd := rp.DueDate.String()
		id := rp.IntervalDays
		lastReviewedAt = &lra
		stability = &rp.Stability
		difficulty = &rp.Difficulty
		intervalRaw = &rp.IntervalRaw
		intervalDays = &id
		dueDate = &dd
		reviewCount = rp.ReviewCount
	}

	_, err = db.conn.Exec(
		`update cards
		 set last_reviewed_at = ?, stability = ?, difficulty = ?,
		     interval_raw = ?, interval_days = ?, due_date = ?, review_count = ?
		 where card_hash = ?;`,
		lastReviewedAt,
		stability,
		difficulty,
		intervalRaw,
		intervalDays,
		dueDate,
		reviewCount,
		cardHash.Hex(),
	)
	if err != nil {
		return errs.Newf("update card performance: %v", err)
	}
	return nil
}

// SaveSession persists a completed review session and all of its reviews
// atomically inside a single transaction.
func (db *Database) SaveSession(
	startedAt types.Timestamp,
	endedAt types.Timestamp,
	reviews []ReviewRecord,
) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return errs.Newf("begin transaction: %v", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var sessionID int64
	err = tx.QueryRow(
		"insert into sessions (started_at, ended_at) values (?, ?) returning session_id;",
		startedAt.String(),
		endedAt.String(),
	).Scan(&sessionID)
	if err != nil {
		return errs.Newf("insert session: %v", err)
	}

	for _, r := range reviews {
		_, err = tx.Exec(
			`insert into reviews
			 (session_id, card_hash, reviewed_at, grade, stability, difficulty,
			  interval_raw, interval_days, due_date)
			 values (?, ?, ?, ?, ?, ?, ?, ?, ?);`,
			sessionID,
			r.CardHash.Hex(),
			r.ReviewedAt.String(),
			r.Grade.String(),
			r.Stability,
			r.Difficulty,
			r.IntervalRaw,
			r.IntervalDays,
			r.DueDate.String(),
		)
		if err != nil {
			return errs.Newf("insert review: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return errs.Newf("commit session: %v", err)
	}
	return nil
}

// DeleteCard removes a card and all its associated reviews from the database.
// Returns an error when the card does not exist.
func (db *Database) DeleteCard(cardHash types.CardHash) error {
	exists, err := db.cardExists(cardHash)
	if err != nil {
		return err
	}
	if !exists {
		return errs.New("Card not found")
	}
	if _, err := db.conn.Exec(
		"delete from reviews where card_hash = ?;", cardHash.Hex(),
	); err != nil {
		return errs.Newf("delete reviews: %v", err)
	}
	if _, err := db.conn.Exec(
		"delete from cards where card_hash = ?;", cardHash.Hex(),
	); err != nil {
		return errs.Newf("delete card: %v", err)
	}
	return nil
}

// CountReviewsInDate returns the number of reviews recorded on the given date.
// The date is matched against the first ten characters of the reviewed_at column.
func (db *Database) CountReviewsInDate(date types.Date) (int, error) {
	var count int
	err := db.conn.QueryRow(
		"select count(*) from reviews where substr(reviewed_at, 1, 10) = ?;",
		date.String(),
	).Scan(&count)
	if err != nil {
		return 0, errs.Newf("count reviews in date: %v", err)
	}
	return count, nil
}

// GetAllSessions returns every session ordered chronologically by start time.
func (db *Database) GetAllSessions() ([]SessionRow, error) {
	rows, err := db.conn.Query(
		"select session_id, started_at, ended_at from sessions order by started_at;",
	)
	if err != nil {
		return nil, errs.Newf("query sessions: %v", err)
	}
	defer rows.Close()

	var sessions []SessionRow
	for rows.Next() {
		var (
			sessionID  int64
			startedStr string
			endedStr   string
		)
		if err := rows.Scan(&sessionID, &startedStr, &endedStr); err != nil {
			return nil, errs.Newf("scan session row: %v", err)
		}
		startedAt, err := types.ParseTimestamp(startedStr)
		if err != nil {
			return nil, err
		}
		endedAt, err := types.ParseTimestamp(endedStr)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, SessionRow{
			SessionID: sessionID,
			StartedAt: startedAt,
			EndedAt:   endedAt,
		})
	}
	return sessions, rows.Err()
}

// GetReviewsForSession returns all reviews belonging to the given session,
// ordered chronologically by review time.
func (db *Database) GetReviewsForSession(sessionID int64) ([]ReviewRow, error) {
	rows, err := db.conn.Query(
		`select review_id, card_hash, reviewed_at, grade, stability, difficulty,
		        interval_raw, interval_days, due_date
		 from reviews where session_id = ? order by reviewed_at;`,
		sessionID,
	)
	if err != nil {
		return nil, errs.Newf("query reviews for session: %v", err)
	}
	defer rows.Close()

	var reviews []ReviewRow
	for rows.Next() {
		var (
			reviewID     int64
			cardHashHex  string
			reviewedStr  string
			gradeStr     string
			stability    float64
			difficulty   float64
			intervalRaw  float64
			intervalDays int64
			dueDateStr   string
		)
		if err := rows.Scan(
			&reviewID, &cardHashHex, &reviewedStr, &gradeStr,
			&stability, &difficulty, &intervalRaw, &intervalDays, &dueDateStr,
		); err != nil {
			return nil, errs.Newf("scan review row: %v", err)
		}

		cardHash, err := types.ParseCardHash(cardHashHex)
		if err != nil {
			return nil, err
		}
		reviewedAt, err := types.ParseTimestamp(reviewedStr)
		if err != nil {
			return nil, err
		}
		grade, err := fsrs.ParseGrade(gradeStr)
		if err != nil {
			return nil, errs.Newf("parse grade: %v", err)
		}
		dueDate, err := types.ParseDate(dueDateStr)
		if err != nil {
			return nil, err
		}

		reviews = append(reviews, ReviewRow{
			ReviewID: reviewID,
			Data: ReviewRecord{
				CardHash:     cardHash,
				ReviewedAt:   reviewedAt,
				Grade:        grade,
				Stability:    stability,
				Difficulty:   difficulty,
				IntervalRaw:  intervalRaw,
				IntervalDays: intervalDays,
				DueDate:      dueDate,
			},
		})
	}
	return reviews, rows.Err()
}

// cardExists returns true when a card with the given hash is present in the database.
func (db *Database) cardExists(cardHash types.CardHash) (bool, error) {
	var count int
	err := db.conn.QueryRow(
		"select count(*) from cards where card_hash = ?;",
		cardHash.Hex(),
	).Scan(&count)
	if err != nil {
		return false, errs.Newf("check card existence: %v", err)
	}
	return count > 0, nil
}
