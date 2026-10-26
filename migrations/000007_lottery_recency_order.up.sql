-- Aurora v1.102: deterministic newest-first lottery history
-- Version: 000007
--
-- GetAll ordered by `timestamp DESC` alone. Timestamps have one-second
-- resolution, so two draws created in the same second tied and SQLite returned
-- them in rowid (insertion) order: "newest first" showed the older draw first,
-- and two page requests could repeat or skip a row. The ordering is now
-- `timestamp DESC, block_height DESC, id`; block_height records the chain
-- append order and the id makes the order total for imported draws that share
-- both a timestamp and a height.
--
-- The composite index makes that ordering a direct index walk. With only the
-- single-column timestamp index the new ORDER BY needed
-- "USE TEMP B-TREE FOR LAST TERM OF ORDER BY" (measured); its leading column
-- serves every timestamp-only lookup, so the old index is dropped instead of
-- being maintained twice.
CREATE INDEX IF NOT EXISTS idx_lottery_recency
    ON lottery_records(timestamp DESC, block_height DESC, id);

DROP INDEX IF EXISTS idx_lottery_timestamp;
