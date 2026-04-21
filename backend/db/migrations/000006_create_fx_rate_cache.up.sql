CREATE TABLE fx_rate_cache (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    base_currency   VARCHAR(3)   NOT NULL,
    target_currency VARCHAR(3)   NOT NULL,
    rate            NUMERIC(20, 8) NOT NULL,
    fetched_at      TIMESTAMPTZ  NOT NULL,
    expires_at      TIMESTAMPTZ  NOT NULL,
    CONSTRAINT fx_rate_cache_pair_unique UNIQUE (base_currency, target_currency)
);
