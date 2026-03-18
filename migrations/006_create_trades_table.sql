-- Create trades table
CREATE TABLE IF NOT EXISTS trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES trader_accounts(id) ON DELETE CASCADE,
    ticket BIGINT NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    type VARCHAR(10) NOT NULL,
    lots DECIMAL(10,2) NOT NULL,
    open_price DECIMAL(20,5) NOT NULL,
    close_price DECIMAL(20,5),
    profit DECIMAL(20,2) DEFAULT 0,
    swap DECIMAL(20,2) DEFAULT 0,
    commission DECIMAL(20,2) DEFAULT 0,
    open_time TIMESTAMP NOT NULL,
    close_time TIMESTAMP,
    status VARCHAR(10) NOT NULL DEFAULT 'open',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP,
    
    UNIQUE(account_id, ticket)
);

CREATE INDEX idx_trades_account ON trades(account_id);
CREATE INDEX idx_trades_status ON trades(status);
CREATE INDEX idx_trades_close_time ON trades(close_time);
CREATE INDEX idx_trades_symbol ON trades(symbol);
