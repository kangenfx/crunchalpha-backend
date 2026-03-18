-- Investor Module Migration - FINAL CORRECT VERSION
-- Team B: UUID + trader_accounts table

-- Table: investor_follows
CREATE TABLE IF NOT EXISTS investor_follows (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  investor_id UUID NOT NULL REFERENCES investors(id) ON DELETE CASCADE,
  trader_account_id UUID NOT NULL REFERENCES trader_accounts(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','PAUSED')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_investor_follows
  ON investor_follows(investor_id, trader_account_id);

-- Table: investor_allocations
CREATE TABLE IF NOT EXISTS investor_allocations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  investor_id UUID NOT NULL REFERENCES investors(id) ON DELETE CASCADE,
  trader_account_id UUID NOT NULL REFERENCES trader_accounts(id) ON DELETE CASCADE,
  allocation_mode TEXT NOT NULL DEFAULT 'PERCENT' CHECK (allocation_mode IN ('PERCENT','FIXED_USD')),
  allocation_value NUMERIC NOT NULL DEFAULT 0,
  max_risk_pct NUMERIC NOT NULL DEFAULT 5,
  max_positions INT NOT NULL DEFAULT 5,
  status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','PAUSED')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_investor_allocations
  ON investor_allocations(investor_id, trader_account_id);

-- Table: investor_subscriptions
CREATE TABLE IF NOT EXISTS investor_subscriptions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  investor_id UUID NOT NULL REFERENCES investors(id) ON DELETE CASCADE,
  trader_account_id UUID NOT NULL REFERENCES trader_accounts(id) ON DELETE CASCADE,
  start_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  end_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','EXPIRED','CANCELLED')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_investor_follows_investor ON investor_follows(investor_id);
CREATE INDEX IF NOT EXISTS idx_investor_allocations_investor ON investor_allocations(investor_id);
CREATE INDEX IF NOT EXISTS idx_investor_subscriptions_investor ON investor_subscriptions(investor_id);
CREATE INDEX IF NOT EXISTS idx_investor_subscriptions_status ON investor_subscriptions(status);

COMMENT ON TABLE investor_follows IS 'Which traders investor follows';
COMMENT ON TABLE investor_allocations IS 'Allocation settings per trader';
COMMENT ON TABLE investor_subscriptions IS 'Active subscriptions';
