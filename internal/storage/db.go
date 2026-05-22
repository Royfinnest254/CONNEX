// internal/storage/db.go — SQLite append-only coordination event store
//
// Wraps go-sqlite3. Schema is loaded from schema.sql at Open() time.
// Append-only invariant is enforced by DB triggers (see schema.sql).

package storage

import (
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// DB wraps an SQLite connection.
type DB struct {
	conn *sql.DB
}

// Open opens (or creates) the SQLite database at the given path and
// applies the schema. Safe to call multiple times.
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := conn.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &DB{conn: conn}, nil
}

// Event is one coordination event to be stored.
type Event struct {
	BundleID       string
	Timestamp      time.Time
	OriginalHash   string // hex
	EnrichedHash   string // hex
	ChainHash      string // hex
	PrevChainHash  string // hex
	BundleJSON     string
	EnrichedXML    string
	EnrichmentLog  string
	QuorumStatus   string // "QUORUM_MET" | "QUORUM_FAILED"
}

// Insert writes a coordination event. Returns error if bundle_id already exists.
func (db *DB) Insert(e *Event) error {
	_, err := db.conn.Exec(`
		INSERT INTO coordination_events
		  (bundle_id, timestamp, original_hash, enriched_hash, chain_hash,
		   prev_chain_hash, bundle_json, enriched_xml, enrichment_log, quorum_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.BundleID,
		e.Timestamp.UTC().Format(time.RFC3339Nano),
		e.OriginalHash,
		e.EnrichedHash,
		e.ChainHash,
		e.PrevChainHash,
		e.BundleJSON,
		e.EnrichedXML,
		e.EnrichmentLog,
		e.QuorumStatus,
	)
	if err != nil {
		return fmt.Errorf("insert event %s: %w", e.BundleID, err)
	}
	return nil
}

// LatestChainHash returns the chain_hash of the most recent event.
// Returns 64 zero hex characters if the table is empty (genesis condition).
func (db *DB) LatestChainHash() (string, error) {
	var h string
	err := db.conn.QueryRow(`
		SELECT chain_hash FROM coordination_events
		ORDER BY sequence_id DESC LIMIT 1`).Scan(&h)
	if err == sql.ErrNoRows {
		return "0000000000000000000000000000000000000000000000000000000000000000", nil
	}
	if err != nil {
		return "", fmt.Errorf("latest chain hash: %w", err)
	}
	return h, nil
}

// Count returns the total number of stored events.
func (db *DB) Count() (int64, error) {
	var n int64
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM coordination_events`).Scan(&n)
	return n, err
}

// Close closes the underlying database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}
