-- Scope idempotency keys to the owning user so two different users
-- can use the same key without colliding, and one user cannot poison
-- another user's idempotency slot.
ALTER TABLE deposits DROP CONSTRAINT deposits_idempotency_key_unique;
ALTER TABLE deposits ADD CONSTRAINT deposits_idempotency_key_unique UNIQUE (user_id, idempotency_key);
