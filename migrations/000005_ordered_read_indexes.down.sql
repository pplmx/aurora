-- Aurora v1.100: composite indexes for ordered reads - Rollback
-- Version: 000005
--
-- The single-column indexes from 000001 remain, so dropping the composites
-- restores the previous (correct, if slower) query plans.
DROP INDEX IF EXISTS idx_oracle_data_source_ts;
DROP INDEX IF EXISTS idx_nft_ops_nft_ts;
