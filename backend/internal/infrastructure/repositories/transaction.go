package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kite/internal/domain/models"
)

type TransactionRepo struct {
	db *pgxpool.Pool
}

func NewTransactionRepo(db *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{db: db}
}

func (r *TransactionRepo) GetHistoryForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.LedgerTransaction, int, error) {
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
		`SELECT DISTINCT ON (lt.id, lt.created_at) lt.id, lt.type, lt.reference_id, lt.created_at
		 FROM ledger_transactions lt
		 JOIN ledger_entries le ON le.transaction_id = lt.id
		 JOIN accounts a ON le.account_id = a.id
		 WHERE a.user_id = $1
		 ORDER BY lt.created_at DESC, lt.id
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var txs []*models.LedgerTransaction
	for rows.Next() {
		var tx models.LedgerTransaction
		var txType string
		if err := rows.Scan(&tx.ID, &txType, &tx.ReferenceID, &tx.CreatedAt); err != nil {
			return nil, 0, err
		}
		tx.Type = models.TransactionType(txType)
		txs = append(txs, &tx)
	}
	return txs, total, rows.Err()
}
