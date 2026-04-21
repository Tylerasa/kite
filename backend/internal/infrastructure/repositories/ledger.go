package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kite/internal/domain/models"
)

type LedgerRepo struct {
	db *pgxpool.Pool
}

func NewLedgerRepo(db *pgxpool.Pool) *LedgerRepo {
	return &LedgerRepo{db: db}
}

func (r *LedgerRepo) CreateTransaction(ctx context.Context, tx *models.LedgerTransaction) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO ledger_transactions (id, type, reference_id, created_at)
		 VALUES ($1, $2, $3, $4)`,
		tx.ID, string(tx.Type), tx.ReferenceID, tx.CreatedAt,
	)
	return err
}

func (r *LedgerRepo) CreateEntries(ctx context.Context, entries []*models.LedgerEntry) error {
	if len(entries) == 0 {
		return nil
	}

	batch := &pgxBatch{}
	for _, e := range entries {
		batch.Queue(
			`INSERT INTO ledger_entries (id, transaction_id, account_id, amount, direction, currency, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			e.ID, e.TransactionID, e.AccountID, e.Amount,
			string(e.Direction), string(e.Currency), e.CreatedAt,
		)
	}

	br := r.db.SendBatch(ctx, batch.Batch())
	defer br.Close()

	for range entries {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// GetBalanceForAccount computes balance via a SUM query — identical logic to ComputeBalance.
// Credits add, debits subtract.
func (r *LedgerRepo) GetBalanceForAccount(ctx context.Context, accountID uuid.UUID) (int64, error) {
	var balance int64
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(CASE WHEN direction = 'credit' THEN amount ELSE -amount END), 0)
		 FROM ledger_entries WHERE account_id = $1`,
		accountID,
	).Scan(&balance)
	return balance, err
}

func (r *LedgerRepo) GetEntriesForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.LedgerEntry, int, error) {
	// Count total
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT lt.id)
		 FROM ledger_transactions lt
		 JOIN ledger_entries le ON le.transaction_id = lt.id
		 JOIN accounts a ON le.account_id = a.id
		 WHERE a.user_id = $1`,
		userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT le.id, le.transaction_id, le.account_id, le.amount, le.direction, le.currency, le.created_at
		 FROM ledger_entries le
		 JOIN accounts a ON le.account_id = a.id
		 WHERE a.user_id = $1
		 ORDER BY le.created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []*models.LedgerEntry
	for rows.Next() {
		var e models.LedgerEntry
		var direction, currency string
		if err := rows.Scan(&e.ID, &e.TransactionID, &e.AccountID, &e.Amount, &direction, &currency, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		e.Direction = models.Direction(direction)
		e.Currency = models.Currency(currency)
		entries = append(entries, &e)
	}
	return entries, total, rows.Err()
}
