CREATE TABLE deposits (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id),
    idempotency_key VARCHAR(255) NOT NULL,
    currency        VARCHAR(3)  NOT NULL,
    amount          BIGINT      NOT NULL CHECK (amount > 0),
    status          VARCHAR(50) NOT NULL DEFAULT 'completed',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT deposits_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX idx_deposits_user_id ON deposits(user_id);
