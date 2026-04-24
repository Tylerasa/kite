package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kite/internal/domain/models"
)

type TransactionRepo struct {
	db *pgxpool.Pool
}

func NewTransactionRepo(db *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{db: db}
}

func (r *TransactionRepo) GetHistoryForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.TransactionRecord, int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT lt.id)
		 FROM ledger_transactions lt
		 JOIN ledger_entries le ON le.transaction_id = lt.id
		 JOIN accounts a ON le.account_id = a.id
		 WHERE a.user_id = $1 AND a.type = 'user_wallet'`,
		userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT ON (lt.created_at, lt.id)
		     lt.id, lt.type, lt.reference_id, lt.created_at,
		     le.amount, le.direction, le.currency,
		     CASE lt.type
		         WHEN 'deposit'    THEN d.status
		         WHEN 'conversion' THEN conv.status
		         WHEN 'payout'     THEN p.status
		         ELSE 'completed'
		     END AS status
		 FROM ledger_transactions lt
		 JOIN ledger_entries le ON le.transaction_id = lt.id
		 JOIN accounts a        ON le.account_id = a.id
		 LEFT JOIN deposits    d    ON lt.type = 'deposit'    AND lt.reference_id = d.id
		 LEFT JOIN conversions conv ON lt.type = 'conversion' AND lt.reference_id = conv.id
		 LEFT JOIN payouts     p    ON lt.type = 'payout'     AND lt.reference_id = p.id
		 WHERE a.user_id = $1 AND a.type = 'user_wallet'
		 ORDER BY lt.created_at DESC, lt.id, le.direction DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var records []*models.TransactionRecord
	for rows.Next() {
		var (
			r         models.TransactionRecord
			txType    string
			direction string
			currency  string
			status    string
		)
		if err := rows.Scan(&r.ID, &txType, &r.ReferenceID, &r.CreatedAt,
			&r.Amount, &direction, &currency, &status); err != nil {
			return nil, 0, err
		}
		r.Type = models.TransactionType(txType)
		r.Direction = models.Direction(direction)
		r.Currency = models.Currency(currency)
		r.Status = status
		records = append(records, &r)
	}
	return records, total, rows.Err()
}

func (r *TransactionRepo) GetByIDForUser(ctx context.Context, userID, transactionID uuid.UUID) (*models.LedgerTransaction, []*models.LedgerEntryWithAccount, error) {
	var tx models.LedgerTransaction
	var txType string
	err := r.db.QueryRow(ctx,
		`SELECT DISTINCT lt.id, lt.type, lt.reference_id, lt.created_at
		 FROM ledger_transactions lt
		 JOIN ledger_entries le ON le.transaction_id = lt.id
		 JOIN accounts a ON le.account_id = a.id
		 WHERE a.user_id = $1 AND lt.id = $2`,
		userID, transactionID,
	).Scan(&tx.ID, &txType, &tx.ReferenceID, &tx.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	tx.Type = models.TransactionType(txType)

	rows, err := r.db.Query(ctx,
		`SELECT le.id, le.amount, le.direction, le.currency, le.created_at, a.type
		 FROM ledger_entries le
		 JOIN accounts a ON le.account_id = a.id
		 WHERE le.transaction_id = $1
		 ORDER BY le.created_at ASC, le.id ASC`,
		transactionID,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var entries []*models.LedgerEntryWithAccount
	for rows.Next() {
		var (
			entry      models.LedgerEntryWithAccount
			direction  string
			currency   string
			accountTyp string
			createdAt  time.Time
		)
		if err := rows.Scan(&entry.ID, &entry.Amount, &direction, &currency, &createdAt, &accountTyp); err != nil {
			return nil, nil, err
		}
		entry.Direction = models.Direction(direction)
		entry.Currency = models.Currency(currency)
		entry.AccountType = models.AccountType(accountTyp)
		entry.CreatedAt = createdAt
		entries = append(entries, &entry)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return &tx, entries, nil
}
