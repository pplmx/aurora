-- Aurora v1.101: index for the token collection listing - Rollback
-- Version: 000006
--
-- Dropping the index restores the previous (correct, if slower) query plan.
DROP INDEX IF EXISTS idx_tokens_created;
