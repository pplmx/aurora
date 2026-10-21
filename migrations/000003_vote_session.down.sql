-- Aurora v1.2: session-scoped vote tallies - Rollback
-- Version: 000003

DROP INDEX IF EXISTS idx_vote_sessions_session_vote;
DROP TABLE IF EXISTS vote_sessions;
