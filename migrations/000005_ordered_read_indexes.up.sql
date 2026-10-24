-- Aurora v1.100: composite indexes for ordered reads
-- Version: 000005
--
-- Oracle data and NFT operation history are read per parent row in
-- descending timestamp order. The single-column indexes on source_id and
-- nft_id force SQLite to build a temporary B-tree for every ORDER BY
-- (measured with EXPLAIN QUERY PLAN: "USE TEMP B-TREE FOR ORDER BY" on
-- oracle_data and nft_operations). Adding the sort column to the index lets
-- the same queries walk the index directly, which also makes the LIMIT a
-- real early stop instead of a post-sort truncation.
CREATE INDEX IF NOT EXISTS idx_oracle_data_source_ts
    ON oracle_data(source_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_nft_ops_nft_ts
    ON nft_operations(nft_id, timestamp DESC);
