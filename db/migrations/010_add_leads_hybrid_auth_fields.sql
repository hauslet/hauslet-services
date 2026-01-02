-- Migration: Add hybrid authentication fields to leads table
-- Description: Adds user_id and is_verified columns to support authenticated + anonymous lead creation
-- Date: 2026-01-02

-- Add user_id column for tracking authenticated users (nullable for backwards compatibility)
ALTER TABLE leads 
ADD COLUMN IF NOT EXISTS user_id UUID,
ADD COLUMN IF NOT EXISTS is_verified BOOLEAN DEFAULT FALSE NOT NULL;

-- Create index on user_id for fast lookups
CREATE INDEX IF NOT EXISTS idx_leads_user_id ON leads(user_id);

-- Create index on is_verified for filtering verified leads
CREATE INDEX IF NOT EXISTS idx_leads_verified ON leads(is_verified);

-- Add comment for documentation
COMMENT ON COLUMN leads.user_id IS 'User ID if lead created by authenticated user (null for anonymous leads)';
COMMENT ON COLUMN leads.is_verified IS 'True if lead created by authenticated user with verified profile';

-- Optional: Add foreign key constraint (uncomment if you want strict referential integrity)
-- ALTER TABLE leads ADD CONSTRAINT fk_leads_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
