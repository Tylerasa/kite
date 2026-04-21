CREATE TABLE ledger_transactions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    type         VARCHAR(50) NOT NULL,
    reference_id UUID        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ledger_tx_reference ON ledger_transactions(reference_id);
CREATE INDEX idx_ledger_tx_created_at ON ledger_transactions(created_at DESC);
