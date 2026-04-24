# Kite — Multi-Currency Wallet

A multi-currency wallet prototype built for the Grey fullstack engineering assessment.

Users can hold balances in USD, GBP, EUR, NGN, and KES; receive simulated deposits; convert between currencies at live FX rates; and send simulated payouts to local bank accounts.

---

## Quick start

```bash
docker compose up
```

| Service  | URL                       |
|----------|---------------------------|
| Frontend | http://localhost:3000     |
| Backend  | http://localhost:8081     |
| Postgres | localhost:5433 (external) |

> **Note:** If ports 8081 or 5433 conflict with local services, adjust the host-side port mappings in `docker-compose.yml`. The internal service ports (8080 / 5432) never change.

---

## Architecture

```
kite/
├── backend/
│   ├── cmd/http/main.go               # Uber FX entry point
│   └── internal/
│       ├── domain/
│       │   ├── models/                # All financial types (int64, Currency, Direction …)
│       │   ├── exceptions/            # Typed domain errors with machine-readable Code fields
│       │   ├── ports/in/              # Use case interfaces (inbound ports)
│       │   ├── ports/out/             # Repository + service interfaces (outbound ports)
│       │   │   └── fakes/             # In-memory implementations for unit tests
│       │   └── services/              # Pure stateless logic (ComputeBalance)
│       ├── application/               # One package per use case (auth, deposits, conversions, payouts, …)
│       └── infrastructure/
│           ├── rest/                  # Gin router, middleware (JWT, request-ID, logger), controllers
│           ├── repositories/          # pgx/v5 SQL implementations
│           ├── adapters/              # Frankfurter FX rate provider (open.er-api.com fallback)
│           ├── jobs/                  # Background payout processor goroutine
│           ├── bootstrap/             # Config (godotenv) + DB pool (pgxpool)
│           └── di/                    # Uber FX module wiring
└── frontend/
    └── src/
        ├── api/                       # axios client + Tanstack Query hooks
        ├── components/                # Layout, shared styles, formatters
        └── routes/                    # Login, Signup, Dashboard, Deposit, Convert, Payout, Transactions
```

### Hexagonal architecture

The `domain` layer has zero external dependencies — it defines all types, errors, and interfaces. `application` depends only on `domain`. `infrastructure` depends on both. This means all business logic can be tested against in-memory fakes (`ports/out/fakes`) without a database.

---

## Data model

```mermaid
erDiagram
    users {
        uuid id PK
        text email UK
        text password_hash
        text name
        text pin_hash
        timestamptz created_at
    }

    accounts {
        uuid id PK
        uuid user_id FK "nullable for system accounts"
        varchar currency
        varchar type "user_wallet | system_cash | fee_income | payout_pending"
        text name
    }

    ledger_transactions {
        uuid id PK
        varchar type "deposit | conversion | payout | reversal"
        uuid reference_id
        timestamptz created_at
    }

    ledger_entries {
        uuid id PK
        uuid transaction_id FK
        uuid account_id FK
        bigint amount "minor units, CHECK > 0"
        varchar direction "debit | credit"
        varchar currency
        timestamptz created_at
    }

    deposits {
        uuid id PK
        uuid user_id FK
        varchar idempotency_key UK
        varchar currency
        bigint amount
        varchar status
    }

    fx_rate_cache {
        uuid id PK
        varchar base_currency
        varchar target_currency UK "(base, target)"
        numeric rate
        timestamptz expires_at
    }

    fx_quotes {
        uuid id PK
        uuid user_id FK
        varchar from_currency
        varchar to_currency
        numeric market_rate
        numeric quoted_rate
        bigint amount_in
        bigint amount_out
        bigint fee
        timestamptz expires_at
        timestamptz executed_at "null until consumed"
    }

    conversions {
        uuid id PK
        uuid quote_id FK UK "UNIQUE prevents double-execute"
        uuid user_id FK
        varchar from_currency
        varchar to_currency
        bigint amount_in
        bigint amount_out
        numeric quoted_rate
        bigint fee
        varchar status
    }

    payouts {
        uuid id PK
        uuid user_id FK
        varchar source_currency
        bigint amount
        varchar status "pending | processing | successful | failed | review"
        boolean compliance_flagged
        text recipient_account_number
        text recipient_bank_code
        text recipient_account_name
        text failure_reason
        timestamptz reversed_at
    }

    audit_log {
        uuid id PK
        uuid user_id FK
        varchar operation
        uuid reference_id
        varchar status
        varchar error_code
        varchar request_id
        timestamptz created_at
        timestamptz updated_at
    }

    users ||--o{ accounts : owns
    accounts ||--o{ ledger_entries : receives
    ledger_transactions ||--o{ ledger_entries : groups
    users ||--o{ deposits : makes
    users ||--o{ fx_quotes : requests
    fx_quotes ||--o| conversions : executes
    users ||--o{ conversions : makes
    users ||--o{ payouts : initiates
    users ||--o{ audit_log : generates
```

---

## Money representation

All monetary amounts are stored as **`int64` minor units** (cents, pence, kobo, etc.). There is no float anywhere in the system.

FX rate arithmetic (multiplication + spread) uses `shopspring/decimal` to avoid floating-point error, then the result is floor-divided back to `int64` before writing to the database.

---

## Double-entry ledger

Every balance change is expressed as a pair of ledger entries that sum to zero:

**Deposit 100 USD**
```
DEBIT   system_cash_USD     10000
CREDIT  user_wallet_USD     10000
```

**Convert 50 USD → EUR (0.75% spread)**
```
DEBIT   user_wallet_USD     5000
CREDIT  system_cash_USD     5000
DEBIT   system_cash_EUR     4249   (market amount)
CREDIT  user_wallet_EUR     4217   (amount after spread)
CREDIT  fee_income_EUR        32   (spread captured; skipped if zero)
```

**Payout 500 NGN (hold on initiation)**
```
DEBIT   user_wallet_NGN     50000
CREDIT  payout_pending_NGN  50000
```

**Payout reversal on failure (new entries — ledger is append-only)**
```
DEBIT   payout_pending_NGN  50000
CREDIT  user_wallet_NGN     50000
```

User balance = `SUM(credits) − SUM(debits)` on their `user_wallet_{CURRENCY}` account. There is no separate balance column that can drift.

---

## Concurrency safety

All balance-mutating operations use `SELECT … FOR UPDATE` inside a Postgres transaction:

- **Deposit**: idempotency enforced by `UNIQUE(idempotency_key)` with `ON CONFLICT DO UPDATE RETURNING`. If the row already exists, no ledger entries are written.
- **Conversion execute**: `UPDATE fx_quotes SET executed_at = NOW() WHERE id = $1 AND executed_at IS NULL` — if zero rows affected, another process already executed this quote.
- **Payout claim**: `SELECT id FROM payouts WHERE status = 'pending' FOR UPDATE SKIP LOCKED` — each payout is claimed by exactly one processor instance.
- **Deadlock prevention**: account IDs are sorted before acquiring locks so concurrent transactions touching the same set of accounts always acquire locks in the same order.

---

## FX rates

1. `POST /conversions/quote` checks `fx_rate_cache` for a non-expired row.
2. Cache miss → `open.er-api.com/v6/latest/<base>` is called (free, no API key required, covers all five currencies).
3. The rate is upserted into `fx_rate_cache` with a 5-minute TTL.
4. A sell-side spread of **0.75%** is applied: `quoted_rate = market_rate × (1 − 0.0075)`.
5. The quote is valid for **45 seconds**. Executing an expired quote returns `400 quote_expired`.

---

## Payout lifecycle

```
pending ──► processing ──► successful
                │
                └──────────► failed ──► (reversal entries written)
pending ──► review            (compliance hold, NGN > ₦500,000)
```

A background goroutine (ticker every 5 s) claims `pending` payouts atomically and simulates async settlement with a 2–5 s delay. 80% succeed; 20% fail and trigger an automatic reversal.

### Compliance rule (bonus)

NGN payouts exceeding ₦500,000 (configurable via `COMPLIANCE_NGN_THRESHOLD` env var, default `50000000` kobo) are moved to `review` with `compliance_flagged = true` instead of being processed. The API returns `202 Accepted`.

---

## Observability (bonus)

- Every request gets a UUID `X-Request-ID` header (generated by middleware, echoed in responses).
- `slog` structured logging: every request logs `method`, `path`, `status`, `duration_ms`, `request_id`.
- Domain errors log at `WARN`; unexpected errors log at `ERROR`.
- In production (`APP_ENV=production`) logs are JSON; in development they are human-readable text.

---

## API reference

All protected endpoints require `Authorization: Bearer <token>`.

| Method | Path | Auth | Notes |
|--------|------|------|-------|
| `POST` | `/auth/signup` | — | Returns JWT |
| `POST` | `/auth/login` | — | Returns JWT |
| `GET` | `/wallets/balances` | JWT | All 5 currencies |
| `GET` | `/wallets/transactions` | JWT | `?page=1&limit=20` |
| `GET` | `/wallets/transactions/:id` | JWT | Full detail with ledger entries |
| `POST` | `/deposits` | JWT | `Idempotency-Key` header required |
| `GET` | `/institutions` | JWT | `?currency=NGN\|KES` — bank/mobile-money registry |
| `POST` | `/payouts/inquiry` | JWT | Account name resolution before payout |
| `POST` | `/conversions/quote` | JWT | Returns quote with `seconds_left` |
| `POST` | `/conversions/execute` | JWT | `{"quote_id": "…"}` |
| `POST` | `/payouts` | JWT | 202 if compliance hold |
| `GET` | `/payouts/:id` | JWT | Poll for status |

### Error format

```json
{
  "error": {
    "code": "quote_expired",
    "message": "The FX quote has expired. Please request a new quote.",
    "details": {}
  }
}
```

---

## Tests

Five required tests, all passing (`go test ./...`):

| Test | Package | What it proves |
|------|---------|----------------|
| `TestLedgerReconciliation` | `domain/services` | `ComputeBalance` is correct across debits and credits |
| `TestDepositIdempotency` | `application/deposits` | Same idempotency key → balance moves exactly once |
| `TestConcurrentConversion` | `application/conversions` | 10 goroutines on one quote → only 1 succeeds |
| `TestExpiredQuote` | `application/conversions` | Past-expiry quote → `ErrQuoteExpired` |
| `TestFailedPayoutReversal` | `application/payouts` | Failed payout → reversal entries → balance fully restored |

Run them:

```bash
cd backend && go test ./...
```

---

## Configuration

| Env var | Default | Description |
|---------|---------|-------------|
| `DATABASE_URL` | — | Postgres DSN |
| `JWT_SECRET` | — | HS256 signing key (required in production) |
| `PORT` | `8080` | HTTP listen port |
| `APP_ENV` | `development` | `production` enables JSON logs |
| `FX_SPREAD_PCT` | `0.0075` | Sell-side FX spread (75 bps) |
| `FX_CACHE_TTL_MINUTES` | `5` | FX rate cache lifetime |
| `COMPLIANCE_NGN_THRESHOLD` | `50000000` | Max NGN payout in kobo before review |

Copy `.env.example` to `.env` before running locally outside Docker.

---

## Design trade-offs and scaling notes

**Balance computed from ledger, not stored separately**

Balances are derived via `SUM` over `ledger_entries`. This is correct by construction (no cache to invalidate) but becomes expensive at scale. The standard mitigation is a materialised balance updated inside the same transaction as the ledger write — a straightforward addition once throughput demands it.

**Postgres as queue for payouts**

The background job uses `FOR UPDATE SKIP LOCKED` to turn a Postgres table into a work queue. This works well at low-to-moderate volume and avoids an external broker dependency for a prototype. At scale, this would be replaced with a proper queue (e.g. SQS, NATS) and the payout worker would be a separate, horizontally scalable service.

**No balance table → no read replica lag risk**

Because balances are always computed from the primary, there is no risk of a replica lag serving a stale balance during a concurrent conversion. A materialised balance would need to be read from the primary as well (or tolerate brief staleness with appropriate UX).

**FX rate cache in Postgres**

Rate caching in the same database keeps the architecture simple. A Redis cache would give lower latency and avoid a DB round-trip on every quote, but adds an operational dependency. For a prototype with a 5-minute TTL the Postgres approach is adequate.

**JWT with no revocation**

Tokens are valid for 24 hours with no server-side session state. A logout endpoint would need a short-lived blocklist (Redis set keyed by `jti`) to invalidate tokens before expiry.

## Scaling to 1M users — what breaks first

**1. Balance reads (breaks first)**
Balances are computed via `SUM(amount)` over `ledger_entries` on every request. With 1M users averaging hundreds of entries each, this becomes a full index scan per balance read. Fix: materialise a `balances` table updated inside the same DB transaction as every ledger write. Reads hit the materialised row; the ledger remains the source of truth for audits.

**2. Postgres as a payout queue (breaks second)**
The background job polls `payouts` with `FOR UPDATE SKIP LOCKED` every few seconds. At volume this creates lock contention and high DB CPU. Fix: publish payout jobs to SQS or NATS on creation; run stateless worker pods that consume from the queue. The payout table still records state — the queue is just the trigger.

**3. FX rate cache in Postgres (next)**
Each quote request that misses the 5-minute cache hits the external FX API and writes to Postgres. At thousands of quotes per second this adds unnecessary write load. Fix: move the cache to Redis with a TTL key; fall back to Postgres then the live API on a cold miss.

**4. Single-region Postgres (last)**
At true scale, read traffic overwhelms a single primary. Fix: PgBouncer for connection pooling + read replicas for balance reads (using the materialised balance table so stale replica lag isn't a concern). Multi-region active-passive comes later once those are saturated.

---

## Loom walkthrough

_[https://www.loom.com/share/887f06d45e4b43cfafe5e3459700585f]_
