// Package fakes provides in-memory implementations of all repository interfaces
// for use in application-layer unit tests.
package fakes

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kite/internal/domain/exceptions"
	"github.com/kite/internal/domain/models"
	"github.com/shopspring/decimal"
)

// --- UserRepo ---

type UserRepo struct {
	mu    sync.Mutex
	users map[string]*models.User // keyed by email
	byID  map[uuid.UUID]*models.User
}

func NewUserRepo() *UserRepo {
	return &UserRepo{users: make(map[string]*models.User), byID: make(map[uuid.UUID]*models.User)}
}

func (r *UserRepo) Create(_ context.Context, user *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.users[user.Email]; exists {
		return exceptions.ErrUserAlreadyExists
	}
	r.users[user.Email] = user
	r.byID[user.ID] = user
	return nil
}

func (r *UserRepo) GetByEmail(_ context.Context, email string) (*models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[email]
	if !ok {
		return nil, exceptions.ErrUserNotFound
	}
	return u, nil
}

func (r *UserRepo) GetByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byID[id]
	if !ok {
		return nil, exceptions.ErrUserNotFound
	}
	return u, nil
}

// --- AccountRepo ---

type AccountRepo struct {
	mu       sync.Mutex
	accounts map[uuid.UUID]*models.Account
}

func NewAccountRepo() *AccountRepo {
	return &AccountRepo{accounts: make(map[uuid.UUID]*models.Account)}
}

func (r *AccountRepo) Create(_ context.Context, account *models.Account) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.accounts[account.ID] = account
	return nil
}

func (r *AccountRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.accounts[id]
	if !ok {
		return nil, exceptions.ErrAccountNotFound
	}
	return a, nil
}

func (r *AccountRepo) GetByUserAndCurrency(_ context.Context, userID uuid.UUID, currency models.Currency, accountType models.AccountType) (*models.Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.accounts {
		if a.UserID != nil && *a.UserID == userID && a.Currency == currency && a.Type == accountType {
			return a, nil
		}
	}
	return nil, exceptions.ErrAccountNotFound
}

func (r *AccountRepo) GetByTypeAndCurrency(_ context.Context, accountType models.AccountType, currency models.Currency) (*models.Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.accounts {
		if a.UserID == nil && a.Type == accountType && a.Currency == currency {
			return a, nil
		}
	}
	return nil, exceptions.ErrAccountNotFound
}

func (r *AccountRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*models.Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.Account
	for _, a := range r.accounts {
		if a.UserID != nil && *a.UserID == userID {
			out = append(out, a)
		}
	}
	return out, nil
}

// SeedSystemAccount adds a system account directly (for test setup).
func (r *AccountRepo) SeedSystemAccount(accountType models.AccountType, currency models.Currency) *models.Account {
	a := &models.Account{
		ID:       uuid.New(),
		Currency: currency,
		Type:     accountType,
		Name:     string(accountType) + " - " + string(currency),
	}
	r.accounts[a.ID] = a
	return a
}

// --- LedgerRepo ---

// LedgerRepo simulates the double-entry ledger with mutex-protected balance operations.
// The mutex in ExecuteWithLock ensures the concurrent conversion test works correctly.
type LedgerRepo struct {
	mu           sync.Mutex
	transactions map[uuid.UUID]*models.LedgerTransaction
	entries      []*models.LedgerEntry
}

func NewLedgerRepo() *LedgerRepo {
	return &LedgerRepo{
		transactions: make(map[uuid.UUID]*models.LedgerTransaction),
	}
}

func (r *LedgerRepo) CreateTransaction(_ context.Context, tx *models.LedgerTransaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transactions[tx.ID] = tx
	return nil
}

func (r *LedgerRepo) CreateEntries(_ context.Context, entries []*models.LedgerEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, entries...)
	return nil
}

func (r *LedgerRepo) GetBalanceForAccount(_ context.Context, accountID uuid.UUID) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.balanceLocked(accountID), nil
}

func (r *LedgerRepo) balanceLocked(accountID uuid.UUID) int64 {
	var balance int64
	for _, e := range r.entries {
		if e.AccountID != accountID {
			continue
		}
		if e.Direction == models.Credit {
			balance += e.Amount
		} else {
			balance -= e.Amount
		}
	}
	return balance
}

func (r *LedgerRepo) GetEntriesForUser(_ context.Context, userID uuid.UUID, limit, offset int) ([]*models.LedgerEntry, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.entries, len(r.entries), nil
}

// CheckAndWriteEntries atomically checks balance and writes entries (simulates FOR UPDATE).
// Returns ErrInsufficientFunds if accountID balance < requiredAmount.
func (r *LedgerRepo) CheckAndWriteEntries(ctx context.Context, accountID uuid.UUID, requiredAmount int64, entries []*models.LedgerEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	balance := r.balanceLocked(accountID)
	if balance < requiredAmount {
		return exceptions.ErrInsufficientFunds
	}
	r.entries = append(r.entries, entries...)
	return nil
}

// --- DepositRepo ---

type DepositRepo struct {
	mu       sync.Mutex
	deposits map[string]*models.Deposit // keyed by idempotency key
}

func NewDepositRepo() *DepositRepo {
	return &DepositRepo{deposits: make(map[string]*models.Deposit)}
}

func (r *DepositRepo) CreateIfNotExists(_ context.Context, deposit *models.Deposit) (*models.Deposit, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.deposits[deposit.IdempotencyKey]; ok {
		return existing, false, nil
	}
	r.deposits[deposit.IdempotencyKey] = deposit
	return deposit, true, nil
}

func (r *DepositRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Deposit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, d := range r.deposits {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, nil
}

// --- FXRateCacheRepo ---

type FXRateCacheRepo struct {
	mu    sync.Mutex
	cache map[string]*models.FXRateCache
}

func NewFXRateCacheRepo() *FXRateCacheRepo {
	return &FXRateCacheRepo{cache: make(map[string]*models.FXRateCache)}
}

func (r *FXRateCacheRepo) Get(_ context.Context, base, target models.Currency) (*models.FXRateCache, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := string(base) + "_" + string(target)
	entry, ok := r.cache[key]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return nil, nil
	}
	return entry, nil
}

func (r *FXRateCacheRepo) Upsert(_ context.Context, entry *models.FXRateCache) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := string(entry.BaseCurrency) + "_" + string(entry.TargetCurrency)
	r.cache[key] = entry
	return nil
}

// --- QuoteRepo ---

type QuoteRepo struct {
	mu       sync.Mutex
	quotes   map[uuid.UUID]*models.FXQuote
	executed map[uuid.UUID]bool
}

func NewQuoteRepo() *QuoteRepo {
	return &QuoteRepo{
		quotes:   make(map[uuid.UUID]*models.FXQuote),
		executed: make(map[uuid.UUID]bool),
	}
}

func (r *QuoteRepo) Create(_ context.Context, quote *models.FXQuote) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.quotes[quote.ID] = quote
	return nil
}

func (r *QuoteRepo) GetByID(_ context.Context, id uuid.UUID) (*models.FXQuote, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	q, ok := r.quotes[id]
	if !ok {
		return nil, nil
	}
	return q, nil
}

func (r *QuoteRepo) MarkExecuted(_ context.Context, id uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.executed[id] {
		return false, nil
	}
	r.executed[id] = true
	now := time.Now()
	if q, ok := r.quotes[id]; ok {
		q.ExecutedAt = &now
	}
	return true, nil
}

// --- ConversionRepo ---

type ConversionRepo struct {
	mu          sync.Mutex
	conversions map[uuid.UUID]*models.Conversion
}

func NewConversionRepo() *ConversionRepo {
	return &ConversionRepo{conversions: make(map[uuid.UUID]*models.Conversion)}
}

func (r *ConversionRepo) Create(_ context.Context, c *models.Conversion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conversions[c.ID] = c
	return nil
}

func (r *ConversionRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Conversion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.conversions[id]
	if !ok {
		return nil, nil
	}
	return c, nil
}

// --- PayoutRepo ---

type PayoutRepo struct {
	mu      sync.Mutex
	payouts map[uuid.UUID]*models.Payout
}

func NewPayoutRepo() *PayoutRepo {
	return &PayoutRepo{payouts: make(map[uuid.UUID]*models.Payout)}
}

func (r *PayoutRepo) Create(_ context.Context, p *models.Payout) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.payouts[p.ID] = p
	return nil
}

func (r *PayoutRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Payout, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.payouts[id]
	if !ok {
		return nil, exceptions.ErrPayoutNotFound
	}
	return p, nil
}

func (r *PayoutRepo) ClaimPending(_ context.Context) (*models.Payout, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.payouts {
		if p.Status == models.PayoutStatusPending {
			p.Status = models.PayoutStatusProcessing
			return p, nil
		}
	}
	return nil, nil
}

func (r *PayoutRepo) UpdateStatus(_ context.Context, id uuid.UUID, status models.PayoutStatus, failureReason *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.payouts[id]
	if !ok {
		return exceptions.ErrPayoutNotFound
	}
	p.Status = status
	p.FailureReason = failureReason
	return nil
}

func (r *PayoutRepo) MarkReversed(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.payouts[id]
	if !ok {
		return exceptions.ErrPayoutNotFound
	}
	now := time.Now()
	p.ReversedAt = &now
	return nil
}

func (r *PayoutRepo) MarkComplianceReview(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.payouts[id]
	if !ok {
		return exceptions.ErrPayoutNotFound
	}
	p.Status = models.PayoutStatusReview
	p.ComplianceFlagged = true
	return nil
}

// --- FXRateProvider (fake for tests) ---

type FXRateProvider struct {
	rates map[string]decimal.Decimal
}

func NewFXRateProvider(rates map[string]decimal.Decimal) *FXRateProvider {
	return &FXRateProvider{rates: rates}
}

func (p *FXRateProvider) GetRate(_ context.Context, from, to models.Currency) (decimal.Decimal, error) {
	key := string(from) + "_" + string(to)
	rate, ok := p.rates[key]
	if !ok {
		return decimal.Zero, exceptions.ErrInternal
	}
	return rate, nil
}

// --- TransactionRepo ---

type TransactionRepo struct{}

func NewTransactionRepo() *TransactionRepo { return &TransactionRepo{} }

func (r *TransactionRepo) GetHistoryForUser(_ context.Context, userID uuid.UUID, limit, offset int) ([]*models.LedgerTransaction, int, error) {
	return nil, 0, nil
}
