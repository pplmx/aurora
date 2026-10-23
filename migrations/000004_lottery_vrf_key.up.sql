-- Aurora v1.2: durable VRF evidence for lottery records
-- Version: 000004

-- Keep the proof key in a side table so the self-bootstrapping repository and
-- the migration can both create it idempotently. Existing draws remain
-- readable with no key and continue to use the deterministic fallback.
CREATE TABLE IF NOT EXISTS lottery_vrf_keys (
    record_id TEXT PRIMARY KEY,
    public_key TEXT NOT NULL,
    FOREIGN KEY (record_id) REFERENCES lottery_records(id) ON DELETE CASCADE
);
