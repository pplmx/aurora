-- Aurora v1.2: Token Transfer History
-- Version: 000002

CREATE TABLE IF NOT EXISTS token_transfers (
    id TEXT PRIMARY KEY,
    token_id TEXT NOT NULL,
    from_addr TEXT,
    to_addr TEXT NOT NULL,
    amount TEXT NOT NULL,
    fee TEXT DEFAULT '0',
    signature TEXT,
    block_height INTEGER NOT NULL,
    timestamp INTEGER NOT NULL,
    memo TEXT
);
CREATE INDEX IF NOT EXISTS idx_token_transfers_token ON token_transfers(token_id);
CREATE INDEX IF NOT EXISTS idx_token_transfers_from ON token_transfers(from_addr);
CREATE INDEX IF NOT EXISTS idx_token_transfers_to ON token_transfers(to_addr);
