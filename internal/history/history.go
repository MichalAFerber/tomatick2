// Package history is a timestamped event log backed by SQLite.
package history

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

// Event is one history row.
type Event struct {
	ID        int64   `json:"id"`
	TS        string  `json:"ts"`
	Kind      string  `json:"kind"`
	Label     *string `json:"label"`
	Action    string  `json:"action"`
	Details   *string `json:"details_json"`
	DurationS *int    `json:"duration_s"`
}

// History is a thin SQLite wrapper for appending and querying events.
type History struct {
	path string
	db   *sql.DB
}

// Open opens (or creates) the history database at path.
func Open(path string) (*History, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	h := &History{path: path, db: db}
	if err := h.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return h, nil
}

func (h *History) initSchema() error {
	_, err := h.db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			ts           TEXT    NOT NULL,
			kind         TEXT    NOT NULL,
			label        TEXT,
			action       TEXT    NOT NULL,
			details_json TEXT,
			duration_s   INTEGER
		);
		CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts);
	`)
	return err
}

// LogEvent appends an event. ts defaults to now if zero.
func (h *History) LogEvent(kind, action string, label string, details map[string]any, durationS *int, ts time.Time) (int64, error) {
	if ts.IsZero() {
		ts = time.Now()
	}
	when := ts.Format("2006-01-02T15:04:05")
	var detailsJSON *string
	if details != nil {
		b, err := json.Marshal(details)
		if err != nil {
			return 0, err
		}
		s := string(b)
		detailsJSON = &s
	}
	var labelArg any
	if label != "" {
		labelArg = label
	}
	res, err := h.db.Exec(
		`INSERT INTO events (ts, kind, label, action, details_json, duration_s)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		when, kind, labelArg, action, detailsJSON, durationS,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Recent returns the newest events, up to limit.
func (h *History) Recent(limit int) ([]Event, error) {
	rows, err := h.db.Query(`SELECT id, ts, kind, label, action, details_json, duration_s
		FROM events ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

// All returns every event oldest-first.
func (h *History) All() ([]Event, error) {
	rows, err := h.db.Query(`SELECT id, ts, kind, label, action, details_json, duration_s
		FROM events ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

// Count returns the number of stored events.
func (h *History) Count() (int, error) {
	var n int
	err := h.db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&n)
	return n, err
}

// Clear deletes all events.
func (h *History) Clear() error {
	_, err := h.db.Exec(`DELETE FROM events`)
	return err
}

// ExportCSV writes all events to path.
func (h *History) ExportCSV(path string) error {
	rows, err := h.All()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"id", "ts", "kind", "label", "action", "details_json", "duration_s"})
	for _, r := range rows {
		_ = w.Write([]string{
			strconv.FormatInt(r.ID, 10),
			r.TS,
			r.Kind,
			deref(r.Label),
			r.Action,
			deref(r.Details),
			intPtr(r.DurationS),
		})
	}
	w.Flush()
	return w.Error()
}

// ExportJSON writes all events to path.
func (h *History) ExportJSON(path string) error {
	rows, err := h.All()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Close closes the database.
func (h *History) Close() error {
	if h.db == nil {
		return nil
	}
	return h.db.Close()
}

func scanEvents(rows *sql.Rows) ([]Event, error) {
	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.TS, &e.Kind, &e.Label, &e.Action, &e.Details, &e.DurationS); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intPtr(n *int) string {
	if n == nil {
		return ""
	}
	return strconv.Itoa(*n)
}
