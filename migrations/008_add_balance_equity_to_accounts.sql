-- Add balance and equity columns to trader_accounts
ALTER TABLE trader_accounts 
ADD COLUMN IF NOT EXISTS balance DECIMAL(20,2) DEFAULT 0,
ADD COLUMN IF NOT EXISTS equity DECIMAL(20,2) DEFAULT 0,
ADD COLUMN IF NOT EXISTS margin DECIMAL(20,2) DEFAULT 0,
ADD COLUMN IF NOT EXISTS free_margin DECIMAL(20,2) DEFAULT 0,
ADD COLUMN IF NOT EXISTS margin_level DECIMAL(10,2) DEFAULT 0,
ADD COLUMN IF NOT EXISTS last_sync_at TIMESTAMP;

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_trader_accounts_balance ON trader_accounts(balance);
CREATE INDEX IF NOT EXISTS idx_trader_accounts_last_sync ON trader_accounts(last_sync_at);

COMMENT ON COLUMN trader_accounts.balance IS 'Current account balance from EA';
COMMENT ON COLUMN trader_accounts.equity IS 'Current account equity from EA';
COMMENT ON COLUMN trader_accounts.last_sync_at IS 'Last time EA synced account data';
