CREATE TABLE fx_quotes (
    id            UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID           NOT NULL REFERENCES users(id),
    from_currency VARCHAR(3)     NOT NULL,
    to_currency   VARCHAR(3)     NOT NULL,
    market_rate   NUMERIC(20, 8) NOT NULL,
    quoted_rate   NUMERIC(20, 8) NOT NULL,
    spread_pct    NUMERIC(8, 6)  NOT NULL,
    amount_in     BIGINT         NOT NULL CHECK (amount_in > 0),
    amount_out    BIGINT         NOT NULL CHECK (amount_out > 0),
    fee           BIGINT         NOT NULL DEFAULT 0,
    expires_at    TIMESTAMPTZ    NOT NULL,
    executed_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fx_quotes_user_id ON fx_quotes(user_id);
