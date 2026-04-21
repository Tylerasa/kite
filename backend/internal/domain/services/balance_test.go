package services_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/services"
)

// TestLedgerReconciliation verifies that ComputeBalance correctly derives balances
// from ledger entries, matching exactly what the DB SUM query would produce.
func TestLedgerReconciliation(t *testing.T) {
	accountID := uuid.New()
	otherAccountID := uuid.New()
	now := time.Now()

	mkEntry := func(accountID uuid.UUID, amount int64, dir models.Direction) *models.LedgerEntry {
		return &models.LedgerEntry{
			ID:        uuid.New(),
			AccountID: accountID,
			Amount:    amount,
			Direction: dir,
			Currency:  models.USD,
			CreatedAt: now,
		}
	}

	tests := []struct {
		name     string
		entries  []*models.LedgerEntry
		expected int64
	}{
		{
			name:     "zero balance with no entries",
			entries:  []*models.LedgerEntry{},
			expected: 0,
		},
		{
			name: "deposit increases balance",
			entries: []*models.LedgerEntry{
				mkEntry(accountID, 10000, models.Credit), // +100.00 USD
			},
			expected: 10000,
		},
		{
			name: "multiple deposits accumulate",
			entries: []*models.LedgerEntry{
				mkEntry(accountID, 10000, models.Credit),
				mkEntry(accountID, 5000, models.Credit),
				mkEntry(accountID, 2500, models.Credit),
			},
			expected: 17500,
		},
		{
			name: "debit reduces balance",
			entries: []*models.LedgerEntry{
				mkEntry(accountID, 10000, models.Credit),
				mkEntry(accountID, 3000, models.Debit),
			},
			expected: 7000,
		},
		{
			name: "entries for other accounts are ignored",
			entries: []*models.LedgerEntry{
				mkEntry(accountID, 10000, models.Credit),
				mkEntry(otherAccountID, 99999, models.Credit), // different account, must be ignored
			},
			expected: 10000,
		},
		{
			name: "full deposit+conversion+payout cycle reconciles",
			entries: []*models.LedgerEntry{
				// Deposit
				mkEntry(accountID, 100_00, models.Credit),
				// Conversion out (debit)
				mkEntry(accountID, 50_00, models.Debit),
				// Payout hold (debit)
				mkEntry(accountID, 20_00, models.Debit),
				// Payout reversal (credit back)
				mkEntry(accountID, 20_00, models.Credit),
			},
			expected: 50_00, // 100 - 50 - 20 + 20
		},
		{
			name: "sum of all credits minus debits is zero for balanced set",
			entries: []*models.LedgerEntry{
				mkEntry(accountID, 500, models.Credit),
				mkEntry(accountID, 200, models.Debit),
				mkEntry(accountID, 300, models.Debit),
			},
			expected: 0, // 500 - 200 - 300
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := services.ComputeBalance(accountID, tt.entries)
			if got != tt.expected {
				t.Errorf("ComputeBalance() = %d, want %d", got, tt.expected)
			}
		})
	}
}
