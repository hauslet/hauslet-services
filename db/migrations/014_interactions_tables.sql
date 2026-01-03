-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- Create interactions table with partitioning by created_at
CREATE TABLE IF NOT EXISTS interactions (
    id UUID NOT NULL,
    user_id UUID,
    session_id VARCHAR(64) NOT NULL,

    interaction_type VARCHAR(32) NOT NULL,
    entity_type VARCHAR(32) NOT NULL,
    entity_id UUID,

    context JSONB DEFAULT '{}',

    -- Metadata
    device_type VARCHAR(16),
    platform VARCHAR(16),
    ip_hash VARCHAR(64),
    user_agent TEXT,
    referrer TEXT,

    -- Technical
    is_bot BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Composite primary key for partitioning
    PRIMARY KEY (created_at, id)
) PARTITION BY RANGE (created_at);

-- Create indexes (will be inherited by partitions)
CREATE INDEX IF NOT EXISTS idx_interactions_lookup ON interactions (entity_id, entity_type, created_at);
CREATE INDEX IF NOT EXISTS idx_interactions_session ON interactions (session_id, created_at);
CREATE INDEX IF NOT EXISTS idx_interactions_user ON interactions (user_id, created_at) WHERE user_id IS NOT NULL;

-- Create partitions for current and next 3 months
-- January 2026
CREATE TABLE IF NOT EXISTS interactions_2026_01 PARTITION OF interactions
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

-- February 2026
CREATE TABLE IF NOT EXISTS interactions_2026_02 PARTITION OF interactions
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

-- March 2026
CREATE TABLE IF NOT EXISTS interactions_2026_03 PARTITION OF interactions
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

-- April 2026
CREATE TABLE IF NOT EXISTS interactions_2026_04 PARTITION OF interactions
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

-- Create interaction_aggregates table for rollups
CREATE TABLE IF NOT EXISTS interaction_aggregates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(32) NOT NULL,
    entity_id UUID NOT NULL,
    period_type VARCHAR(10) NOT NULL,
    period_start TIMESTAMP NOT NULL,

    -- Metrics
    views_total BIGINT DEFAULT 0,
    views_unique BIGINT DEFAULT 0,
    saves_total BIGINT DEFAULT 0,
    unsaves_total BIGINT DEFAULT 0,
    shares_total BIGINT DEFAULT 0,
    contacts_total BIGINT DEFAULT 0,
    booking_requests BIGINT DEFAULT 0,

    -- Derived Metrics
    avg_time_on_page_sec INT DEFAULT 0,
    engagement_score FLOAT DEFAULT 0,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(entity_type, entity_id, period_type, period_start)
);

-- Create index for aggregate lookups
CREATE INDEX IF NOT EXISTS idx_aggregates_lookup ON interaction_aggregates (entity_type, entity_id, period_type, period_start);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

DROP TABLE IF EXISTS interaction_aggregates;
DROP TABLE IF EXISTS interactions_2026_01;
DROP TABLE IF NOT EXISTS interactions_2026_02;
DROP TABLE IF NOT EXISTS interactions_2026_03;
DROP TABLE IF NOT EXISTS interactions_2026_04;
DROP TABLE IF EXISTS interactions;
