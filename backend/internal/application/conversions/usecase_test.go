package conversions_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kite/internal/application/conversions"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/domain/ports/out/fakes"
	"github.com/shopspring/decimal"
)

func buildUseCase() (*conversions.UseCase, *fakes.AccountRepo, *fakes.LedgerRepo, *fakes.QuoteRepo) {
	accountRepo := fakes.NewAccountRepo()
	ledgerRepo := fakes.NewLedgerRepo()
	quoteRepo := fakes.NewQuoteRepo()
	convRepo := fakes.NewConversionRepo()
	cacheRepo := fakes.NewFXRateCacheRepo()
	fxProvider := fakes.NewFXRateProvider(map[string]decimal.Decimal{
		"USD_NGN": decimal.NewFromFloat(1600.0),
		"USD_EUR": decimal.NewFromFloat(0.92),
		"NGN_USD": decimal.NewFromFloat(0.000625),
	})

	// Seed system accounts
	for _, curr := range models.SupportedCurrencies {
		accountRepo.SeedSystemAccount(models.AccountTypeSystemCash, curr)
		accountRepo.SeedSystemAccount(models.AccountTypeFeeIncome, curr)
	}

	uc := conversions.NewUseCase(
		quoteRepo, convRepo, accountRepo, ledgerRepo,
		fxProvider, cacheRepo,
		0.0075,          // 0.75% spread
		5*time.Minute,
	)
	return uc, accountRepo, ledgerRepo, quoteRepo
}

func seedWallet(accountRepo *fakes.AccountRepo, ledgerRepo *fakes.LedgerRepo, userID uuid.UUID, currency models.Currency, balance int64) uuid.UUID {
	uid := userID
	acct := &models.Account{
		ID:       uuid.New(),
		UserID:   &uid,
		Currency: currency,
		Type:     models.AccountTypeUserWallet,
		Name:     "Wallet",
	}
	accountRepo.Create(context.Background(), acct)

	if balance > 0 {
		txID := uuid.New()
		ledgerRepo.CreateTransaction(context.Background(), &models.LedgerTransaction{
			ID: txID, Type: models.TxTypeDeposit, ReferenceID: uuid.New(),
		})
		ledgerRepo.CreateEntries(context.Background(), []*models.LedgerEntry{
			{ID: uuid.New(), TransactionID: txID, AccountID: acct.ID, Amount: balance, Direction: models.Credit, Currency: currency},
		})
	}
	return acct.ID
}

// TestExpiredQuote verifies that executing a quote after its expiry returns ErrQuoteExpired.
func TestExpiredQuote(t *testing.T) {
	uc, accountRepo, ledgerRepo, quoteRepo := buildUseCase()
	ctx := context.Background()

	userID := uuid.New()
	seedWallet(accountRepo, ledgerRepo, userID, models.USD, 100_00)
	seedWallet(accountRepo, ledgerRepo, userID, models.NGN, 0)

	// Create a quote manually with a past expiry.
	expiredQuote := &models.FXQuote{
		ID:           uuid.New(),
		UserID:       userID,
		FromCurrency: models.USD,
		ToCurrency:   models.NGN,
		MarketRate:   "1600",
		QuotedRate:   "1588",
		SpreadPct:    "0.0075",
		AmountIn:     10_00,
		AmountOut:    158800,
		Fee:          1200,
		ExpiresAt:    time.Now().Add(-2 * time.Second), // already expired
		CreatedAt:    time.Now().Add(-47 * time.Second),
	}
	quoteRepo.Create(ctx, expiredQuote)

	_, err := uc.ExecuteConversion(ctx, in.ExecuteCommand{UserID: userID, QuoteID: expiredQuote.ID})
	if err == nil {
		t.Fatal("expected error for expired quote, got nil")
	}

	if err.Error() != "[quote_expired] The FX quote has expired. Please request a new quote." {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestConcurrentConversion verifies that 10 goroutines firing simultaneously against
// a single quote can only succeed once (idempotency via MarkExecuted).
func TestConcurrentConversion(t *testing.T) {
	uc, accountRepo, ledgerRepo, quoteRepo := buildUseCase()
	ctx := context.Background()

	userID := uuid.New()
	seedWallet(accountRepo, ledgerRepo, userID, models.USD, 100_00) // 100 USD
	seedWallet(accountRepo, ledgerRepo, userID, models.NGN, 0)

	// Create one valid quote.
	quote := &models.FXQuote{
		ID:           uuid.New(),
		UserID:       userID,
		FromCurrency: models.USD,
		ToCurrency:   models.NGN,
		MarketRate:   "1600",
		QuotedRate:   "1588",
		SpreadPct:    "0.0075",
		AmountIn:     10_00,
		AmountOut:    158800,
		Fee:          1200,
		ExpiresAt:    time.Now().Add(45 * time.Second),
		CreatedAt:    time.Now(),
	}
	quoteRepo.Create(ctx, quote)

	const numGoroutines = 10
	var wg sync.WaitGroup
	var successCount atomic.Int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := uc.ExecuteConversion(ctx, in.ExecuteCommand{UserID: userID, QuoteID: quote.ID})
			if err == nil {
				successCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if successCount.Load() != 1 {
		t.Errorf("expected exactly 1 successful conversion, got %d (race condition!)", successCount.Load())
	}
}

// TestQuoteAndExecuteHappyPath verifies the full quote→execute flow with balance changes.
func TestQuoteAndExecuteHappyPath(t *testing.T) {
	uc, accountRepo, ledgerRepo, _ := buildUseCase()
	ctx := context.Background()

	userID := uuid.New()
	usdAcctID := seedWallet(accountRepo, ledgerRepo, userID, models.USD, 100_00)
	seedWallet(accountRepo, ledgerRepo, userID, models.NGN, 0)

	// Create quote.
	quoteResult, err := uc.CreateQuote(ctx, in.QuoteCommand{
		UserID:       userID,
		FromCurrency: models.USD,
		ToCurrency:   models.NGN,
		AmountIn:     50_00,
	})
	if err != nil {
		t.Fatalf("create quote: %v", err)
	}
	if quoteResult.Quote.AmountOut <= 0 {
		t.Fatalf("expected positive amount_out, got %d", quoteResult.Quote.AmountOut)
	}

	// Execute.
	convResult, err := uc.ExecuteConversion(ctx, in.ExecuteCommand{
		UserID:  userID,
		QuoteID: quoteResult.Quote.ID,
	})
	if err != nil {
		t.Fatalf("execute conversion: %v", err)
	}
	if convResult.Conversion.AmountIn != 50_00 {
		t.Errorf("expected amount_in 5000, got %d", convResult.Conversion.AmountIn)
	}

	// USD balance should have decreased by amount_in.
	usdBalance, _ := ledgerRepo.GetBalanceForAccount(ctx, usdAcctID)
	expected := int64(100_00 - 50_00)
	if usdBalance != expected {
		t.Errorf("expected USD balance %d, got %d", expected, usdBalance)
	}
}
