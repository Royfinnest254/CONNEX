-- internal/storage/schema.sql
-- Connex coordination events ledger.
-- Append-only enforced at the database level — UPDATE and DELETE raise errors
-- even if application code attempts them. This is a non-negotiable invariant.

CREATE TABLE IF NOT EXISTS coordination_events (
    sequence_id      INTEGER PRIMARY KEY AUTOINCREMENT,
    bundle_id        TEXT UNIQUE NOT NULL,
    timestamp        TEXT NOT NULL,
    original_hash    TEXT NOT NULL,   -- hex SHA-256 of raw ISO 8583 bytes
    enriched_hash    TEXT NOT NULL,   -- hex SHA-256 of enriched XML bytes
    chain_hash       TEXT NOT NULL,   -- hex SHA-256(original||enriched||prev)
    prev_chain_hash  TEXT NOT NULL,   -- chain_hash of the previous event (zeros for genesis)
    bundle_json      TEXT NOT NULL,   -- full proof bundle as JSON
    enriched_xml     TEXT NOT NULL,   -- validated pacs.008.001.08 XML
    enrichment_log   TEXT NOT NULL,   -- JSON: field → {value, source, rule_id, confidence}
    quorum_status    TEXT NOT NULL    -- QUORUM_MET | QUORUM_FAILED
);

-- Append-only triggers: any UPDATE or DELETE raises an abort error.
CREATE TRIGGER IF NOT EXISTS no_update
BEFORE UPDATE ON coordination_events
BEGIN
    SELECT RAISE(ABORT, 'coordination_events is append-only: UPDATE rejected');
END;

CREATE TRIGGER IF NOT EXISTS no_delete
BEFORE DELETE ON coordination_events
BEGIN
    SELECT RAISE(ABORT, 'coordination_events is append-only: DELETE rejected');
END;

-- Indexes for verifier and benchmark queries
CREATE INDEX IF NOT EXISTS idx_timestamp   ON coordination_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_chain_hash  ON coordination_events(chain_hash);
CREATE INDEX IF NOT EXISTS idx_quorum      ON coordination_events(quorum_status);
