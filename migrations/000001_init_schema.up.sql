-- Aurora v1.1: Initial Schema Migration
-- Version: 000001

-- Enable WAL mode for better concurrency
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

-- Blockchain blocks table
CREATE TABLE IF NOT EXISTS blocks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    height INTEGER NOT NULL UNIQUE,
    hash TEXT NOT NULL,
    previous_hash TEXT NOT NULL,
    data TEXT NOT NULL,
    nonce INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_blocks_height ON blocks(height);
CREATE INDEX IF NOT EXISTS idx_blocks_hash ON blocks(hash);
CREATE INDEX IF NOT EXISTS idx_blocks_created_at ON blocks(created_at);

-- Lottery records table
CREATE TABLE IF NOT EXISTS lottery_records (
    id TEXT PRIMARY KEY,
    block_height INTEGER NOT NULL,
    seed TEXT NOT NULL,
    participants TEXT NOT NULL,
    winners TEXT NOT NULL,
    winner_addresses TEXT NOT NULL,
    vrf_proof TEXT NOT NULL,
    vrf_output TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    verified INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_lottery_block_height ON lottery_records(block_height);
CREATE INDEX IF NOT EXISTS idx_lottery_timestamp ON lottery_records(timestamp);

-- Voting tables
CREATE TABLE IF NOT EXISTS votes (
    id TEXT PRIMARY KEY,
    voter_pk TEXT NOT NULL,
    candidate_id TEXT NOT NULL,
    signature TEXT NOT NULL,
    message TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    block_height INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_votes_voter ON votes(voter_pk);
CREATE INDEX IF NOT EXISTS idx_votes_candidate ON votes(candidate_id);

CREATE TABLE IF NOT EXISTS voters (
    public_key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    has_voted INTEGER NOT NULL DEFAULT 0,
    vote_hash TEXT,
    registered_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS candidates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    party TEXT NOT NULL,
    program TEXT NOT NULL,
    description TEXT,
    image_url TEXT,
    vote_count INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_candidates_created ON candidates(created_at);

CREATE TABLE IF NOT EXISTS voting_sessions (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    start_time INTEGER NOT NULL,
    end_time INTEGER NOT NULL,
    status TEXT NOT NULL,
    candidates TEXT NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON voting_sessions(status);

-- NFT tables
CREATE TABLE IF NOT EXISTS nfts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    image_url TEXT,
    token_uri TEXT,
    owner TEXT NOT NULL,
    creator TEXT NOT NULL,
    block_height INTEGER NOT NULL,
    timestamp INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_nft_owner ON nfts(owner);
CREATE INDEX IF NOT EXISTS idx_nft_creator ON nfts(creator);

CREATE TABLE IF NOT EXISTS nft_operations (
    id TEXT PRIMARY KEY,
    nft_id TEXT NOT NULL,
    type TEXT NOT NULL,
    from_addr TEXT,
    to_addr TEXT,
    signature TEXT,
    block_height INTEGER NOT NULL,
    timestamp INTEGER NOT NULL,
    FOREIGN KEY (nft_id) REFERENCES nfts(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_nft_ops_nft_id ON nft_operations(nft_id);

-- Token tables
CREATE TABLE IF NOT EXISTS tokens (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    total_supply TEXT NOT NULL,
    decimals INTEGER DEFAULT 8,
    owner TEXT NOT NULL,
    is_mintable INTEGER DEFAULT 1,
    is_burnable INTEGER DEFAULT 1,
    created_at INTEGER
);

CREATE TABLE IF NOT EXISTS accounts (
    id TEXT PRIMARY KEY,
    token_id TEXT NOT NULL,
    owner TEXT NOT NULL,
    balance TEXT NOT NULL DEFAULT '0',
    updated_at INTEGER,
    UNIQUE(token_id, owner)
);
CREATE INDEX IF NOT EXISTS idx_accounts_token_owner ON accounts(token_id, owner);

CREATE TABLE IF NOT EXISTS allowances (
    id TEXT PRIMARY KEY,
    token_id TEXT NOT NULL,
    owner TEXT NOT NULL,
    spender TEXT NOT NULL,
    amount TEXT NOT NULL DEFAULT '0',
    expires_at INTEGER,
    updated_at INTEGER,
    UNIQUE(token_id, owner, spender)
);
CREATE INDEX IF NOT EXISTS idx_allowances_owner ON allowances(token_id, owner, spender);

-- Oracle tables
CREATE TABLE IF NOT EXISTS data_sources (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    type TEXT,
    method TEXT DEFAULT 'GET',
    headers TEXT,
    path TEXT,
    interval INTEGER DEFAULT 60,
    enabled INTEGER DEFAULT 1,
    created_at INTEGER
);

CREATE TABLE IF NOT EXISTS oracle_data (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    value TEXT,
    raw_response TEXT,
    timestamp INTEGER,
    block_height INTEGER
);
CREATE INDEX IF NOT EXISTS idx_oracle_data_source ON oracle_data(source_id);
CREATE INDEX IF NOT EXISTS idx_oracle_data_timestamp ON oracle_data(timestamp);