-- Aurora v1.103: enforce one ballot per voter in each voting session - Rollback
-- Version: 000008
--
-- Ballots and their session mappings are the audit source of truth and remain
-- untouched. Only the duplicate-prevention claims disappear.
DROP INDEX IF EXISTS idx_session_vote_claims_voter;
DROP TABLE IF EXISTS session_vote_claims;
