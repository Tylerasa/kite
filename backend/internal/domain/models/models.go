package models

import (
	"time"

	"github.com/google/uuid"
)

// Currency represents a supported currency code.
type Currency string

const (
	USD Currency = "USD"
	GBP Currency = "GBP"
	EUR Currency = "EUR"
	NGN Currency = "NGN"
	KES Currency = "KES"
)

var SupportedCurrencies = []Currency{USD, GBP, EUR, NGN, KES}

func (c Currency) Valid() bool {
	for _, s := range SupportedCurrencies {
		if c == s {
			return true
		}
	}
	return false
}

// Direction is the side of a ledger entry.
type Direction string

const (
	Debit  Direction = "debit"
	Credit Direction = "credit"
)

// AccountType classifies a ledger account.
type AccountType string

const (
	AccountTypeUserWallet    AccountType = "user_wallet"
	AccountTypeSystemCash    AccountType = "system_cash"
	AccountTypeFeeIncome     AccountType = "fee_income"
	AccountTypePayoutPending AccountType = "payout_pending"
)

// PayoutStatus is the state machine for payouts.
type PayoutStatus string

const (
	PayoutStatusPending    PayoutStatus = "pending"
	PayoutStatusProcessing PayoutStatus = "processing"
	PayoutStatusSuccessful PayoutStatus = "successful"
	PayoutStatusFailed     PayoutStatus = "failed"
	PayoutStatusReview     PayoutStatus = "review"
)

// TransactionType classifies a ledger transaction.
type TransactionType string

const (
	TxTypeDeposit    TransactionType = "deposit"
	TxTypeConversion TransactionType = "conversion"
	TxTypePayout     TransactionType = "payout"
	TxTypeReversal   TransactionType = "reversal"
)

// --- Entities ---

type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	PinHash      string
	CreatedAt    time.Time
}

// Account is a ledger account in the chart of accounts.
// Amounts are always int64 minor units (cents, kobo, pence).
type Account struct {
	ID        uuid.UUID
	UserID    *uuid.UUID  // nil for system accounts
	Currency  Currency
	Type      AccountType
	Name      string
	CreatedAt time.Time
}

// LedgerTransaction groups a set of debit/credit entries that sum to zero.
type LedgerTransaction struct {
	ID          uuid.UUID
	Type        TransactionType
	ReferenceID uuid.UUID // FK to deposit/conversion/payout
	CreatedAt   time.Time
}

// LedgerEntry is a single line in the double-entry ledger.
// Amount is always positive; Direction determines its sign effect on the account.
type LedgerEntry struct {
	ID            uuid.UUID
	TransactionID uuid.UUID
	AccountID     uuid.UUID
	Amount        int64 // always > 0, minor units
	Direction     Direction
	Currency      Currency
	CreatedAt     time.Time
}

type Deposit struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	IdempotencyKey string
	Currency       Currency
	Amount         int64
	Status         string
	CreatedAt      time.Time
}

type FXRateCache struct {
	ID             uuid.UUID
	BaseCurrency   Currency
	TargetCurrency Currency
	Rate           string // stored as string, parsed to decimal on use
	FetchedAt      time.Time
	ExpiresAt      time.Time
}

type FXQuote struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	FromCurrency Currency
	ToCurrency   Currency
	MarketRate   string // decimal string
	QuotedRate   string // decimal string, after spread
	SpreadPct    string // decimal string e.g. "0.0075"
	AmountIn     int64
	AmountOut    int64
	Fee          int64
	ExpiresAt    time.Time
	ExecutedAt   *time.Time
	CreatedAt    time.Time
}

func (q *FXQuote) IsExpired() bool {
	return time.Now().After(q.ExpiresAt)
}

func (q *FXQuote) IsExecuted() bool {
	return q.ExecutedAt != nil
}

type Conversion struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	QuoteID      uuid.UUID
	FromCurrency Currency
	ToCurrency   Currency
	AmountIn     int64
	AmountOut    int64
	QuotedRate   string
	Fee          int64
	Status       string
	CreatedAt    time.Time
}

type Payout struct {
	ID                    uuid.UUID
	UserID                uuid.UUID
	SourceCurrency        Currency
	Amount                int64
	Status                PayoutStatus
	RecipientAccountNumber string
	RecipientBankCode     string
	RecipientAccountName  string
	ComplianceFlagged     bool
	FailureReason         *string
	ReversedAt            *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// LedgerEntryWithAccount enriches a ledger entry with the account type so
// API consumers can understand where money moved without knowing account UUIDs.
type LedgerEntryWithAccount struct {
	ID          uuid.UUID
	Amount      int64
	Direction   Direction
	Currency    Currency
	AccountType AccountType
	CreatedAt   time.Time
}

// PayoutLedgerTransaction is a ledger transaction with its entries,
// used to show the full money trail on a payout detail response.
type PayoutLedgerTransaction struct {
	ID        uuid.UUID
	Type      TransactionType
	CreatedAt time.Time
	Entries   []*LedgerEntryWithAccount
}

// BalanceEntry is a computed balance per currency for a user.
type BalanceEntry struct {
	Currency Currency
	Amount   int64
}

// TransactionRecord is the unified history feed item.
type TransactionRecord struct {
	ID          uuid.UUID `json:"id"`
	Type        TransactionType `json:"type"`
	ReferenceID uuid.UUID `json:"reference_id"`
	Currency    Currency `json:"currency"`
	Amount      int64 `json:"amount"`
	Direction   Direction `json:"direction"`
	Status      string `json:"status"`
	CreatedAt   time.Time `json:"time"`
	// Type-specific details populated by the use case
	Details any  `json:"details"`
}

type DepositDetails struct {
	IdempotencyKey string
}

type ConversionDetails struct {
	FromCurrency Currency
	ToCurrency   Currency
	AmountIn     int64
	AmountOut    int64
	QuotedRate   string
	Fee          int64
}

type PayoutDetails struct {
	RecipientAccountName   string
	RecipientAccountNumber string
	RecipientBankCode      string
	SourceCurrency         Currency
	Amount                 int64
	Status                 PayoutStatus
	ComplianceFlagged      bool
}

// Institution represents a bank or mobile money provider available for payouts.
type Institution struct {
	Type     string   // "BANK_TRANSFER" or "MOBILE_MONEY"
	BankCode string
	Name     string
	Currency Currency
	Logo     string // optional, used for mobile money providers
}

// AuditLog records every financial operation attempt and its outcome.
// Follows Cinnamon's deferred logging pattern: Create on entry, Update on exit.
type AuditLog struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Operation   string  // "deposit", "conversion_quote", "conversion_execute", "payout"
	ReferenceID *uuid.UUID
	Status      string  // "pending", "success", "failure"
	ErrorCode   *string
	RequestID   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
