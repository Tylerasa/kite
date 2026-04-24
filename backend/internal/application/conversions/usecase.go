package conversions

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kite/internal/domain/exceptions"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/domain/ports/out"
	"github.com/kite/internal/domain/services"
	"github.com/shopspring/decimal"
)

const (
	quoteTTL = 45 * time.Second
)

// Config holds the runtime parameters for the conversions use case.
type Config struct {
	SpreadPct float64
	CacheTTL  time.Duration
}

type UseCase struct {
	quotes      out.QuoteRepository
	conversions out.ConversionRepository
	accounts    out.AccountRepository
	ledger      out.LedgerRepository
	fxProvider  out.FXRateProvider
	fxCache     out.FXRateCacheRepository
	audit       out.AuditLogRepository
	spreadPct   decimal.Decimal
	cacheTTL    time.Duration
}

func NewUseCase(
	quotes out.QuoteRepository,
	conversions out.ConversionRepository,
	accounts out.AccountRepository,
	ledger out.LedgerRepository,
	fxProvider out.FXRateProvider,
	fxCache out.FXRateCacheRepository,
	audit out.AuditLogRepository,
	cfg Config,
) *UseCase {
	return &UseCase{
		quotes:      quotes,
		conversions: conversions,
		accounts:    accounts,
		ledger:      ledger,
		fxProvider:  fxProvider,
		fxCache:     fxCache,
		audit:       audit,
		spreadPct:   decimal.NewFromFloat(cfg.SpreadPct),
		cacheTTL:    cfg.CacheTTL,
	}
}

func (uc *UseCase) CreateQuote(ctx context.Context, cmd in.QuoteCommand) (result *in.QuoteResult, err error) {
	logID := uuid.New()
	auditNow := time.Now().UTC()
	_ = uc.audit.Create(ctx, &models.AuditLog{
		ID: logID, UserID: cmd.UserID, Operation: "conversion_quote",
		Status: "pending", RequestID: services.RequestIDFromCtx(ctx),
		CreatedAt: auditNow, UpdatedAt: auditNow,
	})
	defer func() {
		status, errCode := "success", (*string)(nil)
		if err != nil {
			status = "failure"
			if de, ok := err.(*exceptions.DomainError); ok {
				errCode = &de.Code
			}
		}
		if auditErr := uc.audit.Update(ctx, logID, status, errCode); auditErr != nil {
			slog.Error("audit update failed", "op", "conversion_quote", "error", auditErr)
		}
	}()

	if !cmd.FromCurrency.Valid() || !cmd.ToCurrency.Valid() {
		return nil, exceptions.ErrInvalidCurrency
	}
	if cmd.FromCurrency == cmd.ToCurrency {
		return nil, exceptions.ErrInvalidCurrency.WithDetails(map[string]interface{}{
			"reason": "source and target currencies must differ",
		})
	}
	if cmd.AmountIn <= 0 {
		return nil, exceptions.ErrInvalidCurrency.WithDetails(map[string]interface{}{
			"field": "amount", "reason": "must be greater than 0",
		})
	}

	marketRate, err := uc.getRate(ctx, cmd.FromCurrency, cmd.ToCurrency)
	if err != nil {
		return nil, fmt.Errorf("get fx rate: %w", err)
	}

	// Apply sell-side spread: quoted_rate = market_rate * (1 - spread_pct)
	// This means user gets slightly fewer units of target currency.
	quotedRate := marketRate.Mul(decimal.NewFromInt(1).Sub(uc.spreadPct))

	amountInDec := decimal.NewFromInt(cmd.AmountIn)
	amountOutDec := amountInDec.Mul(quotedRate)
	amountOut := amountOutDec.Floor().IntPart() // always floor, never round up

	// Fee = what the spread captured in target currency minor units.
	amountOutNoSpread := amountInDec.Mul(marketRate).Floor().IntPart()
	fee := amountOutNoSpread - amountOut
	if fee < 0 {
		fee = 0
	}

	if amountOut <= 0 {
		return nil, exceptions.ErrInvalidCurrency.WithDetails(map[string]interface{}{
			"reason": "amount too small to convert",
		})
	}

	quote := &models.FXQuote{
		ID:           uuid.New(),
		UserID:       cmd.UserID,
		FromCurrency: cmd.FromCurrency,
		ToCurrency:   cmd.ToCurrency,
		MarketRate:   marketRate.String(),
		QuotedRate:   quotedRate.String(),
		SpreadPct:    uc.spreadPct.String(),
		AmountIn:     cmd.AmountIn,
		AmountOut:    amountOut,
		Fee:          fee,
		ExpiresAt:    time.Now().UTC().Add(quoteTTL),
		CreatedAt:    time.Now().UTC(),
	}

	if err := uc.quotes.Create(ctx, quote); err != nil {
		return nil, fmt.Errorf("save quote: %w", err)
	}

	return &in.QuoteResult{Quote: quote}, nil
}

func (uc *UseCase) ExecuteConversion(ctx context.Context, cmd in.ExecuteCommand) (result *in.ConversionResult, err error) {
	logID := uuid.New()
	auditNow := time.Now().UTC()
	_ = uc.audit.Create(ctx, &models.AuditLog{
		ID: logID, UserID: cmd.UserID, Operation: "conversion_execute",
		Status: "pending", RequestID: services.RequestIDFromCtx(ctx),
		CreatedAt: auditNow, UpdatedAt: auditNow,
	})
	defer func() {
		status, errCode := "success", (*string)(nil)
		if err != nil {
			status = "failure"
			if de, ok := err.(*exceptions.DomainError); ok {
				errCode = &de.Code
			}
		}
		if auditErr := uc.audit.Update(ctx, logID, status, errCode); auditErr != nil {
			slog.Error("audit update failed", "op", "conversion_execute", "error", auditErr)
		}
	}()

	var quote *models.FXQuote
	quote, err = uc.quotes.GetByID(ctx, cmd.QuoteID)
	if err != nil {
		return nil, err
	}
	if quote == nil {
		return nil, exceptions.ErrQuoteNotFound
	}
	if quote.UserID != cmd.UserID {
		return nil, exceptions.ErrQuoteNotFound // don't leak other users' quotes
	}
	if quote.IsExpired() {
		return nil, exceptions.ErrQuoteExpired.WithDetails(map[string]interface{}{
			"quote_id":   quote.ID,
			"expired_at": quote.ExpiresAt,
		})
	}
	if quote.IsExecuted() {
		return nil, exceptions.ErrQuoteAlreadyExecuted
	}

	// Atomically mark the quote as executed (prevents double-execute).
	var marked bool
	marked, err = uc.quotes.MarkExecuted(ctx, cmd.QuoteID)
	if err != nil {
		return nil, fmt.Errorf("mark quote executed: %w", err)
	}
	if !marked {
		return nil, exceptions.ErrQuoteAlreadyExecuted
	}

	// Get all accounts involved (lock order: sort by type+currency for deadlock prevention).
	userFrom, err := uc.accounts.GetByUserAndCurrency(ctx, cmd.UserID, quote.FromCurrency, models.AccountTypeUserWallet)
	if err != nil {
		return nil, fmt.Errorf("get source wallet: %w", err)
	}
	userTo, err := uc.accounts.GetByUserAndCurrency(ctx, cmd.UserID, quote.ToCurrency, models.AccountTypeUserWallet)
	if err != nil {
		return nil, fmt.Errorf("get target wallet: %w", err)
	}
	sysCashFrom, err := uc.accounts.GetByTypeAndCurrency(ctx, models.AccountTypeSystemCash, quote.FromCurrency)
	if err != nil {
		return nil, fmt.Errorf("get system cash from: %w", err)
	}
	sysCashTo, err := uc.accounts.GetByTypeAndCurrency(ctx, models.AccountTypeSystemCash, quote.ToCurrency)
	if err != nil {
		return nil, fmt.Errorf("get system cash to: %w", err)
	}

	now := time.Now().UTC()
	ledgerTxID := uuid.New()

	// Build entries: debit source wallet, credit target wallet, capture fee.
	entries := []*models.LedgerEntry{
		// Source wallet: debit (user's from-currency goes down)
		{ID: uuid.New(), TransactionID: ledgerTxID, AccountID: userFrom.ID, Amount: quote.AmountIn, Direction: models.Debit, Currency: quote.FromCurrency, CreatedAt: now},
		// System cash from: credit (system absorbs the from-currency)
		{ID: uuid.New(), TransactionID: ledgerTxID, AccountID: sysCashFrom.ID, Amount: quote.AmountIn, Direction: models.Credit, Currency: quote.FromCurrency, CreatedAt: now},
		// System cash to: debit (system pays out the to-currency)
		{ID: uuid.New(), TransactionID: ledgerTxID, AccountID: sysCashTo.ID, Amount: quote.AmountOut + quote.Fee, Direction: models.Debit, Currency: quote.ToCurrency, CreatedAt: now},
		// Target wallet: credit (user's to-currency goes up)
		{ID: uuid.New(), TransactionID: ledgerTxID, AccountID: userTo.ID, Amount: quote.AmountOut, Direction: models.Credit, Currency: quote.ToCurrency, CreatedAt: now},
	}

	// Add fee entry only if non-zero (CHECK constraint rejects amount=0).
	feeAccount, feeErr := uc.accounts.GetByTypeAndCurrency(ctx, models.AccountTypeFeeIncome, quote.ToCurrency)
	if feeErr == nil && quote.Fee > 0 {
		entries = append(entries, &models.LedgerEntry{
			ID: uuid.New(), TransactionID: ledgerTxID, AccountID: feeAccount.ID,
			Amount: quote.Fee, Direction: models.Credit, Currency: quote.ToCurrency, CreatedAt: now,
		})
	}

	ledgerTx := &models.LedgerTransaction{
		ID:          ledgerTxID,
		Type:        models.TxTypeConversion,
		ReferenceID: uuid.New(), // will be set to conversion ID below
		CreatedAt:   now,
	}

	// Create the conversion record first so we have its ID for the ledger reference.
	conversion := &models.Conversion{
		ID:           uuid.New(),
		UserID:       cmd.UserID,
		QuoteID:      cmd.QuoteID,
		FromCurrency: quote.FromCurrency,
		ToCurrency:   quote.ToCurrency,
		AmountIn:     quote.AmountIn,
		AmountOut:    quote.AmountOut,
		QuotedRate:   quote.QuotedRate,
		Fee:          quote.Fee,
		Status:       "completed",
		CreatedAt:    now,
	}
	ledgerTx.ReferenceID = conversion.ID

	// Atomically: lock the source account, verify balance, write ledger tx + entries.
	// Uses SELECT FOR UPDATE in the real DB and a mutex in tests — no overdraft possible.
	if err := uc.ledger.CheckAndWrite(ctx, userFrom.ID, quote.AmountIn, ledgerTx, entries); err != nil {
		return nil, err
	}

	if err := uc.conversions.Create(ctx, conversion); err != nil {
		return nil, fmt.Errorf("save conversion: %w", err)
	}

	return &in.ConversionResult{Conversion: conversion}, nil
}

func (uc *UseCase) getRate(ctx context.Context, from, to models.Currency) (decimal.Decimal, error) {
	// Check DB cache first.
	cached, err := uc.fxCache.Get(ctx, from, to)
	if err != nil {
		return decimal.Zero, err
	}
	if cached != nil {
		rate, err := decimal.NewFromString(cached.Rate)
		if err != nil {
			return decimal.Zero, fmt.Errorf("parse cached rate: %w", err)
		}
		return rate, nil
	}

	// Cache miss — fetch from provider.
	rate, err := uc.fxProvider.GetRate(ctx, from, to)
	if err != nil {
		return decimal.Zero, fmt.Errorf("fetch rate: %w", err)
	}

	// Store in cache.
	now := time.Now().UTC()
	entry := &models.FXRateCache{
		ID:             uuid.New(),
		BaseCurrency:   from,
		TargetCurrency: to,
		Rate:           rate.String(),
		FetchedAt:      now,
		ExpiresAt:      now.Add(uc.cacheTTL),
	}
	_ = uc.fxCache.Upsert(ctx, entry) // cache failure is non-fatal

	return rate, nil
}
