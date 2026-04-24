# Risk, Observations, and Recommendations

This document captures current technical risks and follow-up recommendations. These are intentionally documented for later work; no backend fixes are included here.

## Risks

- Conversion execution is not fully atomic in the real Postgres path. The use case falls back to a plain balance `SUM` query before creating ledger rows, while the repository does not implement the documented `FOR UPDATE`-style atomic check-and-write operation. This can allow overspending when multiple balance-changing requests run concurrently.
- Deposit idempotency is not atomic with ledger creation. A deposit row is inserted before the ledger transaction and entries are written. If the process fails after the deposit insert but before ledger writes, a retry will return the existing deposit and skip crediting the wallet.
- Frontend API routing is inconsistent across environments. Docker exposes the backend on `8081`, Vite proxies to `8085`, the frontend axios client uses relative URLs, and the production nginx config does not proxy API routes to the backend.
- The Compose `VITE_API_URL` value is not used by the frontend code, so production builds served by nginx will not automatically target the backend service.
- Audit logging is best-effort. Audit create/update failures are mostly ignored or logged, which is acceptable for a prototype but not sufficient for a financial audit trail.

## Observations

- The backend follows a clear hexagonal structure: domain models and ports, application use cases, and infrastructure repositories/controllers/jobs.
- Money is represented as `int64` minor units, and FX calculations use decimal arithmetic rather than floats.
- The ledger is append-only and uses double-entry semantics.
- The current unit tests pass and cover the assessment-critical cases: ledger reconciliation, deposit idempotency, concurrent single-quote execution, expired quotes, and payout reversal.
- The README is detailed and accurately explains the intended system behavior, but some concurrency guarantees are stronger than the current implementation.
- The frontend is functional and builds successfully, but the dashboard design is still generic compared with the Wise-like product direction.
- The repository currently has many modified and untracked files; future fixes should avoid broad refactors and isolate changes by area.

## Recommendations

- Add transactional repository methods for money-moving operations: deposit creation with ledger entries, conversion execution, payout hold, payout settlement, and payout reversal.
- Move balance checks and ledger writes into the same database transaction, using deterministic account locking to avoid deadlocks.
- Add Postgres-backed integration tests for concurrent multi-quote conversions, payout/conversion races, and deposit retry behavior after partial failure.
- Align backend URLs and ports across README, Swagger annotations, Vite proxy, Docker Compose, and frontend API configuration.
- Either configure the production frontend nginx image to proxy API routes to the backend or make the client use an injected API base URL consistently.
- Strengthen audit logging once core transaction safety is fixed, especially for failed and partially failed financial operations.
- Continue evolving the frontend toward the Wise design language: restrained layout, strong green primary actions, simple account rows, clear money hierarchy, and minimal decorative card nesting.
