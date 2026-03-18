-- API Keys table for EA authentication
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash TEXT NOT NULL UNIQUE,
    key_prefix TEXT NOT NULL, -- First 8 chars for display (crunch_abc...)
    name TEXT NOT NULL,
    
    -- Security
    allowed_ips TEXT[], -- NULL = allow all IPs
    active BOOLEAN DEFAULT true,
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMP,
    revoked_at TIMESTAMP,
    
    -- Indexes
    CONSTRAINT api_keys_name_check CHECK (char_length(name) <= 100)
);

-- Link API keys to specific accounts
CREATE TABLE IF NOT EXISTS api_key_accounts (
    api_key_id UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES trader_accounts(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (api_key_id, account_id)
);

-- Indexes for fast lookups
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_active ON api_keys(active);
CREATE INDEX idx_api_key_accounts_key ON api_key_accounts(api_key_id);

-- API key usage log (for audit)
CREATE TABLE IF NOT EXISTS api_key_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    ip_address TEXT,
    endpoint TEXT,
    method TEXT,
    status_code INT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_key_logs_key_id ON api_key_logs(api_key_id);
CREATE INDEX idx_api_key_logs_created_at ON api_key_logs(created_at);

COMMENT ON TABLE api_keys IS 'Permanent API keys for EA/bot authentication';
COMMENT ON TABLE api_key_accounts IS 'Links API keys to specific trading accounts';
COMMENT ON TABLE api_key_logs IS 'Audit log of API key usage';
