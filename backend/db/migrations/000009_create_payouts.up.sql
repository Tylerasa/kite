CREATE TABLE payouts (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                  UUID        NOT NULL REFERENCES users(id),
    source_currency          VARCHAR(3)  NOT NULL,
    amount                   BIGINT      NOT NULL CHECK (amount > 0),
    status                   VARCHAR(50) NOT NULL DEFAULT 'pending',
    recipient_account_number VARCHAR(255) NOT NULL,
    recipient_bank_code      VARCHAR(100) NOT NULL,
    recipient_account_name   VARCHAR(255) NOT NULL,
    compliance_flagged       BOOLEAN     NOT NULL DEFAULT FALSE,
    failure_reason           TEXT,
    reversed_at              TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payouts_user_id ON payouts(user_id);
CREATE INDEX idx_payouts_status ON payouts(status) WHERE status IN ('pending', 'processing', 'review');
