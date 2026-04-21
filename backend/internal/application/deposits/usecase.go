package deposits

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kite/internal/domain/exceptions"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/domain/ports/out"
)

type UseCase struct {
	deposits out.DepositRepository
	accounts out.AccountRepository
	ledger   out.LedgerRepository
}

func NewUseCase(deposits out.DepositRepository, accounts out.AccountRepository, ledger out.LedgerRepository) *UseCase {
	return &UseCase{deposits: deposits, accounts: accounts, ledger: ledger}
}

func (uc *UseCase) Deposit(ctx context.Context, cmd in.DepositCommand) (*in.DepositResult, error) {
	if !cmd.Currency.Valid() {
		return nil, exceptions.ErrInvalidCurrency
	}
	if cmd.Amount <= 0 {
		return nil, exceptions.ErrInvalidCurrency.WithDetails(map[string]interface{}{
			"field": "amount", "reason": "must be greater than 0",
		})
	}
	if cmd.IdempotencyKey == "" {
		return nil, exceptions.ErrInternal.WithDetails(map[string]interface{}{
			"field": "idempotency_key", "reason": "required",
		})
	}

	deposit := &models.Deposit{
		ID:             uuid.New(),
		UserID:         cmd.UserID,
		IdempotencyKey: cmd.IdempotencyKey,
		Currency:       cmd.Currency,
		Amount:         cmd.Amount,
		Status:         "completed",
		CreatedAt:      time.Now().UTC(),
	}

	created, isNew, err := uc.deposits.CreateIfNotExists(ctx, deposit)
	if err != nil {
		return nil, fmt.Errorf("create deposit: %w", err)
	}

	// Idempotent return — don't write ledger entries again.
	if !isNew {
		return &in.DepositResult{Deposit: created, IsNew: false}, nil
	}

	// Write double-entry ledger entries.
	userWallet, err := uc.accounts.GetByUserAndCurrency(ctx, cmd.UserID, cmd.Currency, models.AccountTypeUserWallet)
	if err != nil {
		return nil, fmt.Errorf("get user wallet: %w", err)
	}

	systemCash, err := uc.accounts.GetByTypeAndCurrency(ctx, models.AccountTypeSystemCash, cmd.Currency)
	if err != nil {
		return nil, fmt.Errorf("get system cash: %w", err)
	}

	ledgerTx := &models.LedgerTransaction{
		ID:          uuid.New(),
		Type:        models.TxTypeDeposit,
		ReferenceID: created.ID,
		CreatedAt:   time.Now().UTC(),
	}
	if err := uc.ledger.CreateTransaction(ctx, ledgerTx); err != nil {
		return nil, fmt.Errorf("create ledger tx: %w", err)
	}

	now := time.Now().UTC()
	entries := []*models.LedgerEntry{
		{
			ID:            uuid.New(),
			TransactionID: ledgerTx.ID,
			AccountID:     systemCash.ID,
			Amount:        cmd.Amount,
			Direction:     models.Debit,
			Currency:      cmd.Currency,
			CreatedAt:     now,
		},
		{
			ID:            uuid.New(),
			TransactionID: ledgerTx.ID,
			AccountID:     userWallet.ID,
			Amount:        cmd.Amount,
			Direction:     models.Credit,
			Currency:      cmd.Currency,
			CreatedAt:     now,
		},
	}
	if err := uc.ledger.CreateEntries(ctx, entries); err != nil {
		return nil, fmt.Errorf("write ledger entries: %w", err)
	}

	return &in.DepositResult{Deposit: created, IsNew: true}, nil
}
