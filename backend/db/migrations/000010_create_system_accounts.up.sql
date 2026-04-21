-- Seed the system chart of accounts.
-- These are global accounts (no user_id) used for double-entry balancing.
-- ON CONFLICT DO NOTHING makes this migration idempotent.

INSERT INTO accounts (id, user_id, currency, type, name) VALUES
    -- System cash (asset side of deposits/payouts)
    (gen_random_uuid(), NULL, 'USD', 'system_cash',    'System Cash - USD'),
    (gen_random_uuid(), NULL, 'GBP', 'system_cash',    'System Cash - GBP'),
    (gen_random_uuid(), NULL, 'EUR', 'system_cash',    'System Cash - EUR'),
    (gen_random_uuid(), NULL, 'NGN', 'system_cash',    'System Cash - NGN'),
    (gen_random_uuid(), NULL, 'KES', 'system_cash',    'System Cash - KES'),
    -- Fee income (captures spread on conversions)
    (gen_random_uuid(), NULL, 'USD', 'fee_income',     'Fee Income - USD'),
    (gen_random_uuid(), NULL, 'GBP', 'fee_income',     'Fee Income - GBP'),
    (gen_random_uuid(), NULL, 'EUR', 'fee_income',     'Fee Income - EUR'),
    (gen_random_uuid(), NULL, 'NGN', 'fee_income',     'Fee Income - NGN'),
    (gen_random_uuid(), NULL, 'KES', 'fee_income',     'Fee Income - KES'),
    -- Payout pending (suspense for in-flight payouts)
    (gen_random_uuid(), NULL, 'USD', 'payout_pending', 'Payout Pending - USD'),
    (gen_random_uuid(), NULL, 'GBP', 'payout_pending', 'Payout Pending - GBP'),
    (gen_random_uuid(), NULL, 'EUR', 'payout_pending', 'Payout Pending - EUR'),
    (gen_random_uuid(), NULL, 'NGN', 'payout_pending', 'Payout Pending - NGN'),
    (gen_random_uuid(), NULL, 'KES', 'payout_pending', 'Payout Pending - KES')
ON CONFLICT DO NOTHING;
