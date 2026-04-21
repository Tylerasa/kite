package out

import (
	"context"

	"github.com/google/uuid"
	"github.com/kite/internal/domain/models"
)

// UserRepository persists and retrieves users.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// AccountRepository manages the chart of accounts.
type AccountRepository interface {
	Create(ctx context.Context, account *models.Account) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Account, error)
	GetByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency models.Currency, accountType models.AccountType) (*models.Account, error)
	GetByTypeAndCurrency(ctx context.Context, accountType models.AccountType, currency models.Currency) (*models.Account, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.Account, error)
}

// LedgerRepository is the financial core — append-only, never updates entries.
type LedgerRepository interface {
	CreateTransaction(ctx context.Context, tx *models.LedgerTransaction) error
	CreateEntries(ctx context.Context, entries []*models.LedgerEntry) error
	GetBalanceForAccount(ctx context.Context, accountID uuid.UUID) (int64, error)
	// GetEntriesForUser returns paginated entries across all user accounts, newest first.
	GetEntriesForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.LedgerEntry, int, error)
}

// DepositRepository manages deposit records.
type DepositRepository interface {
	// CreateIfNotExists inserts a deposit if the idempotency key is new.
	// Returns (deposit, true, nil) if created, (existing, false, nil) if duplicate.
	CreateIfNotExists(ctx context.Context, deposit *models.Deposit) (*models.Deposit, bool, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Deposit, error)
}

// FXRateCacheRepository manages the FX rate cache.
type FXRateCacheRepository interface {
	Get(ctx context.Context, base, target models.Currency) (*models.FXRateCache, error)
	Upsert(ctx context.Context, entry *models.FXRateCache) error
}

// QuoteRepository manages FX quotes.
type QuoteRepository interface {
	Create(ctx context.Context, quote *models.FXQuote) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.FXQuote, error)
	// MarkExecuted sets executed_at atomically; returns false if already executed.
	MarkExecuted(ctx context.Context, id uuid.UUID) (bool, error)
}

// ConversionRepository persists executed conversions.
type ConversionRepository interface {
	Create(ctx context.Context, c *models.Conversion) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Conversion, error)
}

// PayoutRepository manages payout lifecycle.
type PayoutRepository interface {
	Create(ctx context.Context, p *models.Payout) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Payout, error)
	// ClaimPending atomically transitions one pending payout to processing.
	// Returns nil if none available.
	ClaimPending(ctx context.Context) (*models.Payout, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.PayoutStatus, failureReason *string) error
	MarkReversed(ctx context.Context, id uuid.UUID) error
	MarkComplianceReview(ctx context.Context, id uuid.UUID) error
}

// TransactionRepository provides the unified history feed.
type TransactionRepository interface {
	GetHistoryForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.LedgerTransaction, int, error)
}
