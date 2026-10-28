-- Aurora v1.103: enforce one ballot per voter in each voting session
-- Version: 000008
--
-- voters.has_voted was the only duplicate-ballot guard, so a voter who used
-- their key once could never participate in a later election. The claim table
-- scopes that guard to (session_id, voter_pk) while the legacy voter flag
-- remains a lifetime "has ever voted" profile attribute.
CREATE TABLE IF NOT EXISTS session_vote_claims (
    session_id TEXT NOT NULL,
    voter_pk TEXT NOT NULL,
    vote_id TEXT NOT NULL UNIQUE,
    claimed_at INTEGER NOT NULL,
    PRIMARY KEY (session_id, voter_pk)
);

CREATE INDEX IF NOT EXISTS idx_session_vote_claims_voter
    ON session_vote_claims(voter_pk);

-- Existing session-bound ballots become claims without changing the vote or
-- vote_sessions rows. If a historical database contains more than one ballot
-- from the same voter in one session, the earliest row wins the claim slot;
-- OR IGNORE preserves the audit rows rather than deleting election history.
INSERT OR IGNORE INTO session_vote_claims
    (session_id, voter_pk, vote_id, claimed_at)
SELECT vs.session_id, v.voter_pk, v.id, v.timestamp
FROM votes v
JOIN vote_sessions vs ON vs.vote_id = v.id
ORDER BY v.timestamp, v.id;

UPDATE voters
SET has_voted = 1
WHERE has_voted = 0
  AND EXISTS (
      SELECT 1
      FROM session_vote_claims c
      WHERE c.voter_pk = voters.public_key
  );
