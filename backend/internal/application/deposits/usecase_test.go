package deposits_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/kite/internal/application/deposits"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/domain/ports/out/fakes"
)

func setup() (*deposits.UseCase, *fakes.AccountRepo, *fakes.LedgerRepo) {
	accountRepo := fakes.NewAccountRepo()
	ledgerRepo := fakes.NewLedgerRepo()
	depositRepo := fakes.NewDepositRepo()

	// Seed required system accounts
	accountRepo.SeedSystemAccount(models.AccountTypeSystemCash, models.USD)
	accountRepo.SeedSystemAccount(models.AccountTypeSystemCash, models.NGN)

	uc := deposits.NewUseCase(depositRepo, accountRepo, ledgerRepo, fakes.NewAuditLogRepo())
	return uc, accountRepo, ledgerRepo
}

func seedUserWallet(accountRepo *fakes.AccountRepo, userID uuid.UUID, currency models.Currency) {
	uid := userID
	accountRepo.Create(context.Background(), &models.Account{
		ID:       uuid.New(),
		UserID:   &uid,
		Currency: currency,
		Type:     models.AccountTypeUserWallet,
		Name:     "Wallet - " + string(currency),
	})
}

// TestDepositIdempotency verifies that submitting the same deposit twice
// with the same idempotency key results in the balance moving exactly once.
func TestDepositIdempotency(t *testing.T) {
	uc, accountRepo, ledgerRepo := setup()
	ctx := context.Background()

	userID := uuid.New()
	seedUserWallet(accountRepo, userID, models.USD)

	cmd := in.DepositCommand{
		UserID:         userID,
		IdempotencyKey: "idem-key-abc123",
		Currency:       models.USD,
		Amount:         10000, // 100.00 USD
	}

	// First submission — should succeed and write ledger entries.
	result1, err := uc.Deposit(ctx, cmd)
	if err != nil {
		t.Fatalf("first deposit failed: %v", err)
	}
	if !result1.IsNew {
		t.Fatal("expected first deposit to be new")
	}

	// Second submission — same idempotency key, must NOT write entries again.
	result2, err := uc.Deposit(ctx, cmd)
	if err != nil {
		t.Fatalf("second deposit failed: %v", err)
	}
	if result2.IsNew {
		t.Fatal("expected second deposit to be a duplicate (not new)")
	}
	if result2.Deposit.ID != result1.Deposit.ID {
		t.Fatal("expected second deposit to return the same deposit ID")
	}

	// Balance should reflect exactly ONE deposit.
	userWallet, _ := accountRepo.GetByUserAndCurrency(ctx, userID, models.USD, models.AccountTypeUserWallet)
	balance, _ := ledgerRepo.GetBalanceForAccount(ctx, userWallet.ID)
	if balance != 10000 {
		t.Errorf("expected balance 10000, got %d (idempotency failure)", balance)
	}
}

// TestConcurrentDeposit stress-tests that concurrent deposits with different
// idempotency keys all succeed and each credits the balance exactly once.
func TestConcurrentDeposit(t *testing.T) {
	uc, accountRepo, ledgerRepo := setup()
	ctx := context.Background()

	userID := uuid.New()
	seedUserWallet(accountRepo, userID, models.USD)

	const numDeposits = 20
	var wg sync.WaitGroup
	var successCount atomic.Int32

	for i := 0; i < numDeposits; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cmd := in.DepositCommand{
				UserID:         userID,
				IdempotencyKey: uuid.New().String(),
				Currency:       models.USD,
				Amount:         1000, // 10.00 USD each
			}
			_, err := uc.Deposit(ctx, cmd)
			if err == nil {
				successCount.Add(1)
			}
		}(i)
	}
	wg.Wait()

	if int(successCount.Load()) != numDeposits {
		t.Errorf("expected %d successful deposits, got %d", numDeposits, successCount.Load())
	}

	userWallet, _ := accountRepo.GetByUserAndCurrency(ctx, userID, models.USD, models.AccountTypeUserWallet)
	balance, _ := ledgerRepo.GetBalanceForAccount(ctx, userWallet.ID)
	expected := int64(numDeposits * 1000)
	if balance != expected {
		t.Errorf("expected balance %d, got %d", expected, balance)
	}
}
