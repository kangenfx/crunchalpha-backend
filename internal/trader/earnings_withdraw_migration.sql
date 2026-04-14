CREATE TABLE IF NOT EXISTS earnings_withdrawals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('trader', 'analyst')),
    amount NUMERIC(18,2) NOT NULL,
    method TEXT NOT NULL DEFAULT 'bank_transfer',
    notes TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'paid')),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_earnings_withdrawals_user ON earnings_withdrawals(user_id);
CREATE INDEX IF NOT EXISTS idx_earnings_withdrawals_status ON earnings_withdrawals(status);
