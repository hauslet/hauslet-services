-- Migration: Create user_2fa table for two-factor authentication

CREATE TABLE user_2fa (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    
    -- Method: 'email', 'sms', 'authenticator'
    method TEXT NOT NULL CHECK (method IN ('email', 'sms', 'authenticator')),
    
    -- For SMS method: phone number to send codes to
    phone_number TEXT,
    
    -- For Authenticator method: encrypted TOTP secret
    totp_secret_encrypted TEXT,
    
    -- Status
    is_enabled BOOLEAN NOT NULL DEFAULT false,
    enabled_at TIMESTAMP,
    
    -- Backup codes (hashed with bcrypt)
    backup_codes_hash TEXT[],
    backup_codes_remaining INT NOT NULL DEFAULT 8,
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_2fa_user_id ON user_2fa(user_id);

-- Trigger to update updated_at
CREATE TRIGGER update_user_2fa_updated_at
    BEFORE UPDATE ON user_2fa
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Rollback: DROP TABLE user_2fa;
