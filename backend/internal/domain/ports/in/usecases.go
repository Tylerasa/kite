package in

import (
	"context"

	"github.com/google/uuid"
	"github.com/kite/internal/domain/models"
)

// --- Auth ---

type SignupCommand struct {
	Name     string
	Email    string
	Password string
	Pin      string
}

type LoginCommand struct {
	Email    string
	Password string
}

type TokenResult struct {
	Token  string
	UserID uuid.UUID
	Name   string
}

type AuthUseCase interface {
	Signup(ctx context.Context, cmd SignupCommand) (*TokenResult, error)
	Login(ctx context.Context, cmd LoginCommand) (*TokenResult, error)
	VerifyPin(ctx context.Context, userID uuid.UUID, pin string) error
}

// --- Wallet ---

type WalletUseCase interface {
	GetBalances(ctx context.Context, userID uuid.UUID) ([]*models.BalanceEntry, error)
}

// --- Deposits ---

type DepositCommand struct {
	UserID         uuid.UUID
	IdempotencyKey string
	Currency       models.Currency
	Amount         int64
}

type DepositResult struct {
	Deposit *models.Deposit
	IsNew   bool
}

type DepositUseCase interface {
	Deposit(ctx context.Context, cmd DepositCommand) (*DepositResult, error)
}

// --- FX Conversions ---

type QuoteCommand struct {
	UserID       uuid.UUID
	FromCurrency models.Currency
	ToCurrency   models.Currency
	AmountIn     int64
}

type QuoteResult struct {
	Quote *models.FXQuote
}

type ExecuteCommand struct {
	UserID  uuid.UUID
	QuoteID uuid.UUID
}

type ConversionResult struct {
	Conversion *models.Conversion
}

type ConversionUseCase interface {
	CreateQuote(ctx context.Context, cmd QuoteCommand) (*QuoteResult, error)
	ExecuteConversion(ctx context.Context, cmd ExecuteCommand) (*ConversionResult, error)
}

// --- Payouts ---

type PayoutCommand struct {
	UserID                 uuid.UUID
	SourceCurrency         models.Currency
	Amount                 int64
	RecipientAccountNumber string
	RecipientBankCode      string
	RecipientAccountName   string
}

type PayoutResult struct {
	Payout *models.Payout
}

type PayoutDetail struct {
	Payout *models.Payout
	Ledger []*models.PayoutLedgerTransaction
}

type PayoutUseCase interface {
	InitiatePayout(ctx context.Context, cmd PayoutCommand) (*PayoutResult, error)
	GetPayout(ctx context.Context, userID, payoutID uuid.UUID) (*PayoutDetail, error)
	ProcessPayout(ctx context.Context, payoutID uuid.UUID) error
	ReversePayout(ctx context.Context, payoutID uuid.UUID) error
}

// --- Institutions & Recipient Inquiry ---

type InquiryCommand struct {
	UserID        uuid.UUID
	Currency      models.Currency
	BankCode      string
	AccountNumber string
}

type InquiryResult struct {
	AccountName     string
	AccountNumber   string
	BankCode        string
	BankName        string
	InstitutionType string
}

type InstitutionUseCase interface {
	ListInstitutions(ctx context.Context, currency models.Currency) ([]*models.Institution, error)
	ResolveRecipient(ctx context.Context, cmd InquiryCommand) (*InquiryResult, error)
}

// --- Transactions ---

type HistoryQuery struct {
	UserID uuid.UUID
	Page   int
	Limit  int
}

type HistoryResult struct {
	Items      []*models.TransactionRecord
	Total      int
	Page       int
	TotalPages int
}

type TransactionDetail struct {
	Transaction *models.LedgerTransaction
	Entries     []*models.LedgerEntryWithAccount
}

type TransactionUseCase interface {
	GetHistory(ctx context.Context, q HistoryQuery) (*HistoryResult, error)
	GetByID(ctx context.Context, userID, transactionID uuid.UUID) (*TransactionDetail, error)
}
