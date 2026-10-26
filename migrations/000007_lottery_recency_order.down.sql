-- Aurora v1.102: deterministic newest-first lottery history - Rollback
-- Version: 000007
--
-- Restores the single-column timestamp index and drops the composite one. The
-- previous (non-deterministic for same-second draws) query plan comes back
-- with it; the repository's own bootstrap re-applies the composite index on
-- the next open.
CREATE INDEX IF NOT EXISTS idx_lottery_timestamp
    ON lottery_records(timestamp);

DROP INDEX IF EXISTS idx_lottery_recency;
