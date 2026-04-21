CREATE TABLE conversions (
    id            UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID           NOT NULL REFERENCES users(id),
    quote_id      UUID           NOT NULL REFERENCES fx_quotes(id),
    from_currency VARCHAR(3)     NOT NULL,
    to_currency   VARCHAR(3)     NOT NULL,
    amount_in     BIGINT         NOT NULL CHECK (amount_in > 0),
    amount_out    BIGINT         NOT NULL CHECK (amount_out > 0),
    quoted_rate   NUMERIC(20, 8) NOT NULL,
    fee           BIGINT         NOT NULL DEFAULT 0,
    status        VARCHAR(50)    NOT NULL DEFAULT 'completed',
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    CONSTRAINT conversions_quote_id_unique UNIQUE (quote_id)
);

CREATE INDEX idx_conversions_user_id ON conversions(user_id);
