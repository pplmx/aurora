-- Aurora v1.101: index for the token collection listing
-- Version: 000006
--
-- `token list`, GET /api/v1/token/list and the desktop Tokens card read the
-- whole collection newest first. Measured with EXPLAIN QUERY PLAN against a
-- database the repository itself created:
--
--   SELECT ... FROM tokens ORDER BY created_at DESC, id
--     -> SCAN tokens, USE TEMP B-TREE FOR ORDER BY
--
-- Every token is materialised and sorted before the LIMIT can apply. The
-- composite index lets the same query walk the index directly, which also
-- makes the LIMIT a real early stop.
--
-- The NFT listing (`ORDER BY rowid DESC`) needs no index: SQLite walks the
-- implicit rowid key backwards for it, and its plan is a plain SCAN with no
-- temp B-tree (measured).
CREATE INDEX IF NOT EXISTS idx_tokens_created
    ON tokens(created_at DESC, id);
