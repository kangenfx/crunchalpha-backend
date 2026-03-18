-- Analyst Module Tables
CREATE TABLE IF NOT EXISTS analyst_signal_sets (
    id          TEXT PRIMARY KEY,
    analyst_id  BIGINT NOT NULL,
    name        TEXT NOT NULL,
    market      TEXT NOT NULL,
    style       TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'Active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_analyst_signal_sets_analyst
    ON analyst_signal_sets(analyst_id);

CREATE TABLE IF NOT EXISTS analyst_signals (
    id            BIGSERIAL PRIMARY KEY,
    analyst_id    BIGINT NOT NULL,
    set_id        TEXT REFERENCES analyst_signal_sets(id) ON DELETE SET NULL,
    pair          TEXT NOT NULL,
    direction     TEXT NOT NULL CHECK (direction IN ('BUY','SELL')),
    entry         TEXT NOT NULL,
    sl            TEXT NOT NULL,
    tp            TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'OPEN',
    issued_at     TEXT,
    analyst_name  TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_analyst_signals_analyst
    ON analyst_signals(analyst_id);
CREATE INDEX IF NOT EXISTS idx_analyst_signals_set
    ON analyst_signals(set_id);
CREATE INDEX IF NOT EXISTS idx_analyst_signals_status
    ON analyst_signals(status);
