package wallet

import (
	"context"

	"github.com/google/uuid"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/ports/out"
)

type UseCase struct {
	accounts out.AccountRepository
	ledger   out.LedgerRepository
}

func NewUseCase(accounts out.AccountRepository, ledger out.LedgerRepository) *UseCase {
	return &UseCase{accounts: accounts, ledger: ledger}
}

func (uc *UseCase) GetBalances(ctx context.Context, userID uuid.UUID) ([]*models.BalanceEntry, error) {
	accounts, err := uc.accounts.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	balances := make([]*models.BalanceEntry, 0, len(models.SupportedCurrencies))
	for _, currency := range models.SupportedCurrencies {
		var balance int64
		for _, acct := range accounts {
			if acct.Currency == currency && acct.Type == models.AccountTypeUserWallet {
				b, err := uc.ledger.GetBalanceForAccount(ctx, acct.ID)
				if err != nil {
					return nil, err
				}
				balance = b
				break
			}
		}
		balances = append(balances, &models.BalanceEntry{Currency: currency, Amount: balance})
	}
	return balances, nil
}
