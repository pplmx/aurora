-- Aurora v1.1: Initial Schema Migration - Rollback
-- Version: 000001

-- Drop tables in reverse order of creation (respecting foreign keys)
DROP TABLE IF EXISTS oracle_data;
DROP TABLE IF EXISTS data_sources;
DROP TABLE IF EXISTS allowances;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS tokens;
DROP TABLE IF EXISTS nft_operations;
DROP TABLE IF EXISTS nfts;
DROP TABLE IF EXISTS voting_sessions;
DROP TABLE IF EXISTS candidates;
DROP TABLE IF EXISTS voters;
DROP TABLE IF EXISTS votes;
DROP TABLE IF EXISTS lottery_records;
DROP TABLE IF EXISTS blocks;
