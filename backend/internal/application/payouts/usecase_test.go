package payouts_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/kite/internal/application/payouts"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/domain/ports/out/fakes"
	"github.com/kite/internal/domain/services"
)

func buildUseCase(complianceThreshold int64) (*payouts.UseCase, *fakes.AccountRepo, *fakes.LedgerRepo, *fakes.PayoutRepo) {
	accountRepo := fakes.NewAccountRepo()
	ledgerRepo := fakes.NewLedgerRepo()
	payoutRepo := fakes.NewPayoutRepoWithLedger(ledgerRepo)

	// Seed system accounts for all supported currencies.
	for _, curr := range models.SupportedCurrencies {
		accountRepo.SeedSystemAccount(models.AccountTypeSystemCash, curr)
		accountRepo.SeedSystemAccount(models.AccountTypePayoutPending, curr)
	}

	uc := payouts.NewUseCase(payoutRepo, accountRepo, ledgerRepo, complianceThreshold, services.NewTransactionGuard(), fakes.NewAuditLogRepo())
	return uc, accountRepo, ledgerRepo, payoutRepo
}

func seedUserWallet(accountRepo *fakes.AccountRepo, ledgerRepo *fakes.LedgerRepo, userID uuid.UUID, currency models.Currency, balance int64) uuid.UUID {
	uid := userID
	acct := &models.Account{
		ID: uuid.New(), UserID: &uid, Currency: currency,
		Type: models.AccountTypeUserWallet, Name: "Wallet",
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

// TestFailedPayoutReversal verifies that when a payout fails, the user's balance
// is restored via reversal ledger entries — not by mutating any existing entry.
func TestFailedPayoutReversal(t *testing.T) {
	uc, accountRepo, ledgerRepo, _ := buildUseCase(50_000_000)
	ctx := context.Background()

	userID := uuid.New()
	walletID := seedUserWallet(accountRepo, ledgerRepo, userID, models.NGN, 100_000_00) // ₦100,000

	cmd := in.PayoutCommand{
		UserID:                 userID,
		SourceCurrency:         models.NGN,
		Amount:                 40_000_00, // ₦40,000
		RecipientAccountNumber: "0123456789",
		RecipientBankCode:      "058",
		RecipientAccountName:   "John Doe",
	}

	result, err := uc.InitiatePayout(ctx, cmd)
	if err != nil {
		t.Fatalf("initiate payout: %v", err)
	}

	// Balance should be reduced (hold in place).
	balanceAfterHold, _ := ledgerRepo.GetBalanceForAccount(ctx, walletID)
	expectedAfterHold := int64(100_000_00 - 40_000_00)
	if balanceAfterHold != expectedAfterHold {
		t.Errorf("after hold: expected balance %d, got %d", expectedAfterHold, balanceAfterHold)
	}

	// Force this payout to fail.
	payouts.ForceFailPayout(result.Payout.ID)

	// Process the payout (will fail and trigger reversal).
	if err := uc.ProcessPayout(ctx, result.Payout.ID); err != nil {
		t.Fatalf("process payout: %v", err)
	}

	// Balance should be restored to original.
	balanceAfterReversal, _ := ledgerRepo.GetBalanceForAccount(ctx, walletID)
	if balanceAfterReversal != 100_000_00 {
		t.Errorf("after reversal: expected balance %d, got %d", int64(100_000_00), balanceAfterReversal)
	}
}

// TestComplianceHold verifies that NGN payouts above the threshold are flagged for review.
func TestComplianceHold(t *testing.T) {
	threshold := int64(50_000_000) // ₦500,000 in kobo
	uc, accountRepo, ledgerRepo, payoutRepo := buildUseCase(threshold)
	ctx := context.Background()

	userID := uuid.New()
	seedUserWallet(accountRepo, ledgerRepo, userID, models.NGN, 1_000_000_00) // ₦1M

	cmd := in.PayoutCommand{
		UserID:                 userID,
		SourceCurrency:         models.NGN,
		Amount:                 60_000_000, // ₦600,000 — above threshold
		RecipientAccountNumber: "0123456789",
		RecipientBankCode:      "058",
		RecipientAccountName:   "John Doe",
	}

	result, err := uc.InitiatePayout(ctx, cmd)
	if err != nil {
		t.Fatalf("initiate payout: %v", err)
	}

	if result.Payout.Status != models.PayoutStatusReview {
		t.Errorf("expected status 'review', got '%s'", result.Payout.Status)
	}
	if !result.Payout.ComplianceFlagged {
		t.Error("expected compliance_flagged=true")
	}

	// Confirm payout is stored as review in repository.
	stored, _ := payoutRepo.GetByID(ctx, result.Payout.ID)
	if stored.Status != models.PayoutStatusReview {
		t.Errorf("stored payout status: expected review, got %s", stored.Status)
	}
}

// TestInsufficientFundsRejected verifies payout is rejected when balance is insufficient.
func TestInsufficientFundsRejected(t *testing.T) {
	uc, accountRepo, ledgerRepo, _ := buildUseCase(50_000_000)
	ctx := context.Background()

	userID := uuid.New()
	seedUserWallet(accountRepo, ledgerRepo, userID, models.USD, 10_00) // only $10

	cmd := in.PayoutCommand{
		UserID:                 userID,
		SourceCurrency:         models.USD,
		Amount:                 50_00, // $50 — more than balance
		RecipientAccountNumber: "0123456789",
		RecipientBankCode:      "058",
		RecipientAccountName:   "John Doe",
	}

	_, err := uc.InitiatePayout(ctx, cmd)
	if err == nil {
		t.Fatal("expected insufficient funds error, got nil")
	}
	if err.Error() != "[insufficient_funds] Insufficient balance to complete this operation." {
		t.Errorf("unexpected error: %v", err)
	}
}
