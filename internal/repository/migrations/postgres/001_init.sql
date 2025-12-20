-- Migration 001: Initialize core schema for PostgreSQL
-- Created: 2025-12-09
-- Purpose: Create channels, users, and messages tables

-- Channels table
CREATE TABLE IF NOT EXISTS channels (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    retain_history_on_delete BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_message_at TIMESTAMPTZ,
    total_messages BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_channels_enabled ON channels(enabled);
CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    display_name TEXT,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    total_messages BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_last_seen ON users(last_seen_at);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tags JSONB
);

CREATE INDEX IF NOT EXISTS idx_messages_channel_id ON messages(channel_id);
CREATE INDEX IF NOT EXISTS idx_messages_user_id ON messages(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_sent_at ON messages(sent_at);
CREATE INDEX IF NOT EXISTS idx_messages_channel_sent ON messages(channel_id, sent_at);

-- Full-text search index on message text
CREATE INDEX IF NOT EXISTS idx_messages_text_search ON messages USING GIN (to_tsvector('english', text));

-- Function to update channel stats on message insert
CREATE OR REPLACE FUNCTION update_channel_stats()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE channels 
    SET total_messages = total_messages + 1,
        last_message_at = NEW.sent_at,
        updated_at = NOW()
    WHERE id = NEW.channel_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to update user stats on message insert
CREATE OR REPLACE FUNCTION update_user_stats()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE users 
    SET total_messages = total_messages + 1,
        last_seen_at = NEW.sent_at
    WHERE id = NEW.user_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop triggers if they exist (for idempotency)
DROP TRIGGER IF EXISTS trigger_update_channel_stats ON messages;
DROP TRIGGER IF EXISTS trigger_update_user_stats ON messages;

-- Create triggers
CREATE TRIGGER trigger_update_channel_stats
    AFTER INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_channel_stats();

CREATE TRIGGER trigger_update_user_stats
    AFTER INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_user_stats();

-- Profiles
CREATE TABLE IF NOT EXISTS profiles (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_profiles_name ON profiles(name);

-- Channel <-> Profile (enforces channel_id unique)
CREATE TABLE IF NOT EXISTS profile_channels (
    profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (profile_id, channel_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_profile_channels_channel_unique ON profile_channels(channel_id);
CREATE INDEX IF NOT EXISTS idx_profile_channels_profile_id ON profile_channels(profile_id);

-- Organizations
CREATE TABLE IF NOT EXISTS organizations (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_organizations_name ON organizations(name);

-- Profile <-> Organization
CREATE TABLE IF NOT EXISTS organization_members (
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (organization_id, profile_id)
);

CREATE INDEX IF NOT EXISTS idx_organization_members_org_id ON organization_members(organization_id);
CREATE INDEX IF NOT EXISTS idx_organization_members_profile_id ON organization_members(profile_id);

-- Events
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_start_at ON events(start_at);

-- Profile <-> Event
CREATE TABLE IF NOT EXISTS event_participants (
    event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, profile_id)
);

CREATE INDEX IF NOT EXISTS idx_event_participants_event_id ON event_participants(event_id);
CREATE INDEX IF NOT EXISTS idx_event_participants_profile_id ON event_participants(profile_id);

-- Collaborations
CREATE TABLE IF NOT EXISTS collaborations (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    shared_chat BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_collaborations_name ON collaborations(name);

-- Profile <-> Collaboration
CREATE TABLE IF NOT EXISTS collaboration_participants (
    collaboration_id BIGINT NOT NULL REFERENCES collaborations(id) ON DELETE CASCADE,
    profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (collaboration_id, profile_id)
);

CREATE INDEX IF NOT EXISTS idx_collaboration_participants_collab_id ON collaboration_participants(collaboration_id);
CREATE INDEX IF NOT EXISTS idx_collaboration_participants_profile_id ON collaboration_participants(profile_id);
