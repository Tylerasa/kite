CREATE TABLE IF NOT EXISTS audit_log (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        REFERENCES users(id),
    operation    VARCHAR(50) NOT NULL,
    reference_id UUID,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_code   VARCHAR(100),
    request_id   VARCHAR(100),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON audit_log(user_id, created_at DESC);
CREATE INDEX ON audit_log(reference_id) WHERE reference_id IS NOT NULL;
