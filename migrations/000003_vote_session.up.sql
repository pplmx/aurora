-- Aurora v1.2: session-scoped vote tallies
-- Version: 000003

-- Keep the session mapping separate from votes. Rebuilding votes would
-- discard session IDs already written by a self-bootstrapped repository,
-- and legacy rows cannot be attributed to an election retroactively.
CREATE TABLE IF NOT EXISTS vote_sessions (
    vote_id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_vote_sessions_session_vote
    ON vote_sessions(session_id, vote_id);
