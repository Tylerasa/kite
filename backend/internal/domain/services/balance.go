package services

import (
	"github.com/google/uuid"
	"github.com/kite/internal/domain/models"
)

// ComputeBalance calculates the balance for a specific account from a slice of ledger entries.
// Credits increase the balance; debits decrease it. All amounts are int64 minor units.
// This pure function is the source of truth — the DB query mirrors this exact logic.
func ComputeBalance(accountID uuid.UUID, entries []*models.LedgerEntry) int64 {
	var balance int64
	for _, e := range entries {
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

// ComputeBalanceByCurrency groups entries by account and currency.
func ComputeBalancesByCurrency(entries []*models.LedgerEntry, accounts []*models.Account) map[models.Currency]int64 {
	// Build account -> currency map
	accountCurrency := make(map[uuid.UUID]models.Currency, len(accounts))
	for _, a := range accounts {
		if a.Type == models.AccountTypeUserWallet {
			accountCurrency[a.ID] = a.Currency
		}
	}

	balances := make(map[models.Currency]int64)
	for _, e := range entries {
		currency, ok := accountCurrency[e.AccountID]
		if !ok {
			continue
		}
		if e.Direction == models.Credit {
			balances[currency] += e.Amount
		} else {
			balances[currency] -= e.Amount
		}
	}
	return balances
}
