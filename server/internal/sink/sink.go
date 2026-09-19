package sink

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Row is one score result
type Row struct {
	StreamSeq uint64 //JetSteam stream sequence number
	Subject   string
	TeamID    uint16
	CheckName string
	Message   string
	Passed    bool
	Points    uint8
	Details   map[string]string
	Timestamp string
}

type DB struct {
	conn *sql.DB
}

// Open opens/creates the sqlite database at path and creates the results table
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	conn.SetMaxOpenConns(1)

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
	} {
		if _, err := conn.Exec(pragma); err != nil {
			connCloseErr := conn.Close()
			if connCloseErr != nil {
				slog.Error("Issue closing database connection", "error", connCloseErr.Error())
			}
			return nil, fmt.Errorf("set %q: %w", pragma, err)
		}
	}

	const schema = `
CREATE TABLE IF NOT EXISTS results (
	stream_seq  INTEGER PRIMARY KEY,
	subject     TEXT NOT NULL,
	team_id     INTEGER NOT NULL,
	check_name  TEXT NOT NULL,
	message     TEXT NOT NULL,
	passed      INTEGER NOT NULL,
	points      INTEGER NOT NULL,
	details     TEXT NOT NULL,
	timestamp   TEXT NOT NULL,
	received_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS results_team_check_idx ON results (team_id, check_name);`

	if _, err := conn.Exec(schema); err != nil {
		connCloseErr := conn.Close()
		if connCloseErr != nil {
			slog.Error("Issue closing database connection", "error", connCloseErr.Error())
		}
		return nil, fmt.Errorf("create results table: %w", err)
	}

	return &DB{conn: conn}, nil
}

// InsertBatch inserts rows in a single transaction
func (db *DB) InsertBatch(ctx context.Context, rows []Row) error {
	if len(rows) == 0 {
		return nil
	}

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO results (stream_seq, subject, team_id, check_name, message, passed, points, details, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, row := range rows {
		details, err := json.Marshal(row.Details)
		if err != nil {
			return fmt.Errorf("marshal details for stream_seq %d: %w", row.StreamSeq, err)
		}

		if _, err := stmt.ExecContext(ctx, row.StreamSeq, row.Subject, row.TeamID, row.CheckName, row.Message, row.Passed, row.Points, string(details), row.Timestamp); err != nil {
			return fmt.Errorf("insert stream_seq %d: %w", row.StreamSeq, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// TeamScore is one team's summed points
type TeamScore struct {
	TeamID uint16
	Points int64
}

type ScoreQuery struct {
	Start     time.Time // Inclusive
	End       time.Time // Exclusive
	TeamID    *uint16
	CheckName string
}

// filterClause builds the queries WHERE clause
func filterClause(q ScoreQuery) (string, []any) {
	var b strings.Builder
	b.WriteString("datetime(timestamp) >= datetime(?) AND datetime(timestamp) < datetime(?)")
	args := []any{q.Start.UTC().Format(time.RFC3339), q.End.UTC().Format(time.RFC3339)}

	if q.TeamID != nil {
		b.WriteString(" AND team_id = ?")
		args = append(args, *q.TeamID)
	}
	if q.CheckName != "" {
		b.WriteString(" AND check_name = ?")
		args = append(args, q.CheckName)
	}

	return b.String(), args
}

// TeamScores sums points per team_id over a time range
func (db *DB) TeamScores(ctx context.Context, q ScoreQuery) ([]TeamScore, error) {
	where, args := filterClause(q)

	rows, err := db.conn.QueryContext(ctx, "SELECT team_id, SUM(points) FROM results WHERE "+where+" GROUP BY team_id ORDER BY team_id", args...) // #nosec G202
	if err != nil {
		return nil, fmt.Errorf("query team scores: %w", err)
	}
	defer rows.Close()

	var scores []TeamScore
	for rows.Next() {
		var score TeamScore
		if err := rows.Scan(&score.TeamID, &score.Points); err != nil {
			return nil, fmt.Errorf("scan team score: %w", err)
		}
		scores = append(scores, score)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate team scores: %w", err)
	}

	return scores, nil
}
