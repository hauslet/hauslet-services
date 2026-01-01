-- +goose Up
-- Create leads module tables for lead capture and management

-- Main leads table
CREATE TABLE IF NOT EXISTS leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL,
    business_id UUID,

    -- Contact information
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone_number VARCHAR(50),

    -- Lead details
    message TEXT NOT NULL,
    source VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'new',

    -- Spam detection
    spam_score DECIMAL(3,2) NOT NULL DEFAULT 0.0,
    is_spam BOOLEAN NOT NULL DEFAULT false,

    -- Assignment
    assigned_to UUID,
    assigned_at TIMESTAMPTZ,
    auto_assigned BOOLEAN NOT NULL DEFAULT false,

    -- Metadata (JSONB)
    user_agent TEXT,
    ip_address VARCHAR(45), -- IPv6 compatible
    referrer_url TEXT,
    utm_params JSONB,
    custom_metadata JSONB,

    -- Response tracking
    first_response_at TIMESTAMPTZ,
    response_time BIGINT, -- seconds to first response
    response_count INT NOT NULL DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    -- Foreign key constraints
    CONSTRAINT fk_leads_listing FOREIGN KEY (listing_id)
        REFERENCES listings(id) ON DELETE CASCADE,
    CONSTRAINT fk_leads_business FOREIGN KEY (business_id)
        REFERENCES businesses(id) ON DELETE SET NULL
    -- Note: assigned_to references users table but constraint omitted for flexibility
);

-- Indexes for leads table
CREATE INDEX idx_leads_listing ON leads(listing_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_leads_business ON leads(business_id) WHERE deleted_at IS NULL AND business_id IS NOT NULL;
CREATE INDEX idx_leads_email ON leads(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_leads_ip ON leads(ip_address) WHERE deleted_at IS NULL AND ip_address IS NOT NULL;
CREATE INDEX idx_leads_source ON leads(source) WHERE deleted_at IS NULL;
CREATE INDEX idx_leads_status ON leads(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_leads_spam ON leads(is_spam) WHERE deleted_at IS NULL;
CREATE INDEX idx_leads_assigned_to ON leads(assigned_to) WHERE deleted_at IS NULL AND assigned_to IS NOT NULL;
CREATE INDEX idx_leads_created ON leads(created_at DESC) WHERE deleted_at IS NULL;

-- Composite indexes for rate limiting queries (critical for performance)
CREATE INDEX idx_leads_email_created ON leads(email, created_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_leads_ip_created ON leads(ip_address, created_at) WHERE deleted_at IS NULL AND ip_address IS NOT NULL;
CREATE INDEX idx_leads_listing_email_created ON leads(listing_id, email, created_at) WHERE deleted_at IS NULL;

-- Lead events table (audit trail)
CREATE TABLE IF NOT EXISTS lead_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id UUID NOT NULL,

    -- Event details
    event_type VARCHAR(50) NOT NULL,
    actor_id UUID,
    actor_type VARCHAR(50) NOT NULL,

    -- Status transition
    old_status VARCHAR(50),
    new_status VARCHAR(50),

    -- Change details (JSONB)
    changes JSONB,
    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign key
    CONSTRAINT fk_lead_events_lead FOREIGN KEY (lead_id)
        REFERENCES leads(id) ON DELETE CASCADE
);

-- Indexes for lead_events table
CREATE INDEX idx_lead_events_lead ON lead_events(lead_id, created_at DESC);
CREATE INDEX idx_lead_events_created ON lead_events(created_at DESC);

-- Lead assignments table (routing history)
CREATE TABLE IF NOT EXISTS lead_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id UUID NOT NULL,

    -- Assignment details
    from_user_id UUID,
    to_user_id UUID NOT NULL,

    -- Assignment context
    reason VARCHAR(50) NOT NULL,
    notes TEXT,
    assigned_by UUID,

    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign key
    CONSTRAINT fk_lead_assignments_lead FOREIGN KEY (lead_id)
        REFERENCES leads(id) ON DELETE CASCADE
    -- Note: user ID references omitted for flexibility
);

-- Indexes for lead_assignments table
CREATE INDEX idx_lead_assignments_lead ON lead_assignments(lead_id, assigned_at DESC);
CREATE INDEX idx_lead_assignments_user ON lead_assignments(to_user_id, assigned_at DESC);

-- Update trigger for leads.updated_at
CREATE OR REPLACE FUNCTION update_leads_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_leads_updated_at
    BEFORE UPDATE ON leads
    FOR EACH ROW
    EXECUTE FUNCTION update_leads_updated_at();

-- Comments for documentation
COMMENT ON TABLE leads IS 'Stores lead inquiries from potential customers for listings';
COMMENT ON COLUMN leads.spam_score IS 'Spam probability score: 0.0 = clean, 1.0 = spam';
COMMENT ON COLUMN leads.utm_params IS 'UTM tracking parameters from lead source (JSONB)';
COMMENT ON COLUMN leads.custom_metadata IS 'Extensible metadata for future use (JSONB)';
COMMENT ON COLUMN leads.response_time IS 'Time to first response in seconds';
COMMENT ON COLUMN leads.response_count IS 'Number of times the lead has been contacted';

COMMENT ON TABLE lead_events IS 'Audit trail of all state changes and actions performed on leads';
COMMENT ON COLUMN lead_events.changes IS 'Field-level changes for detailed tracking (JSONB)';

COMMENT ON TABLE lead_assignments IS 'History of lead assignments for routing analytics';

-- +goose Down
-- Rollback migration

DROP TRIGGER IF EXISTS trigger_update_leads_updated_at ON leads;
DROP FUNCTION IF EXISTS update_leads_updated_at();

DROP TABLE IF EXISTS lead_assignments;
DROP TABLE IF EXISTS lead_events;
DROP TABLE IF EXISTS leads;
