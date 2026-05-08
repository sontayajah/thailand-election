-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Ballot categories (PRD §1.3.2)
CREATE TYPE ballot_type AS ENUM ('CONSTITUENCY', 'PARTY_LIST', 'REFERENDUM');

-- Referendum vote options (PRD §1.3.2)
CREATE TYPE referendum_vote AS ENUM ('AGREE', 'DISAGREE', 'ABSTAIN');

-- Vote origin (PRD §6.2)
CREATE TYPE vote_source AS ENUM ('physical', 'online', 'admin_batch', 'simulator');

-- Voter authentication session lifecycle (PRD §6.2)
CREATE TYPE voter_session_status AS ENUM (
    'otp_pending',
    'authenticated',
    'voting',
    'completed',
    'expired'
);
