CREATE TABLE ledger_entries (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID        NOT NULL REFERENCES ledger_transactions(id),
    account_id     UUID        NOT NULL REFERENCES accounts(id),
    amount         BIGINT      NOT NULL CHECK (amount > 0),
    direction      VARCHAR(10) NOT NULL CHECK (direction IN ('debit', 'credit')),
    currency       VARCHAR(3)  NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Primary index for balance queries (SUM per account)
CREATE INDEX idx_ledger_entries_account_id ON ledger_entries(account_id);
CREATE INDEX idx_ledger_entries_transaction_id ON ledger_entries(transaction_id);
