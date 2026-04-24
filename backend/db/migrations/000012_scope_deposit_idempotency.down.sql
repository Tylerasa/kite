ALTER TABLE deposits DROP CONSTRAINT deposits_idempotency_key_unique;
ALTER TABLE deposits ADD CONSTRAINT deposits_idempotency_key_unique UNIQUE (idempotency_key);
