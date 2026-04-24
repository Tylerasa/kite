package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kite/internal/domain/exceptions"
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

// CheckAndWrite locks the account row with SELECT FOR UPDATE, verifies the balance
// is sufficient, then writes the ledger transaction and all entries — all inside
// a single database transaction. Concurrent calls for the same account serialise
// at the lock, preventing overdrafts.
func (r *LedgerRepo) CheckAndWrite(ctx context.Context, accountID uuid.UUID, requiredAmount int64, ledgerTx *models.LedgerTransaction, entries []*models.LedgerEntry) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Lock the account row. Any other CheckAndWrite targeting this account blocks
	// here until we commit or roll back — this is the serialisation point.
	var lockedID uuid.UUID
	if err := tx.QueryRow(ctx,
		`SELECT id FROM accounts WHERE id = $1 FOR UPDATE`,
		accountID,
	).Scan(&lockedID); err != nil {
		return err
	}

	// Compute the current balance within the locked transaction so we see a
	// consistent view — no concurrent write can slip in between the lock and
	// this read.
	var balance int64
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(CASE WHEN direction = 'credit' THEN amount ELSE -amount END), 0)
		 FROM ledger_entries WHERE account_id = $1`,
		accountID,
	).Scan(&balance); err != nil {
		return err
	}
	if balance < requiredAmount {
		return exceptions.ErrInsufficientFunds
	}

	// Write the ledger transaction header.
	if _, err := tx.Exec(ctx,
		`INSERT INTO ledger_transactions (id, type, reference_id, created_at)
		 VALUES ($1, $2, $3, $4)`,
		ledgerTx.ID, string(ledgerTx.Type), ledgerTx.ReferenceID, ledgerTx.CreatedAt,
	); err != nil {
		return err
	}

	// Write all entries in a single batch.
	batch := &pgxBatch{}
	for _, e := range entries {
		batch.Queue(
			`INSERT INTO ledger_entries (id, transaction_id, account_id, amount, direction, currency, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			e.ID, e.TransactionID, e.AccountID, e.Amount,
			string(e.Direction), string(e.Currency), e.CreatedAt,
		)
	}
	br := tx.SendBatch(ctx, batch.Batch())
	for range entries {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return err
		}
	}
	br.Close()

	return tx.Commit(ctx)
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

func (r *LedgerRepo) GetByReference(ctx context.Context, referenceID uuid.UUID) ([]*models.PayoutLedgerTransaction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT lt.id, lt.type, lt.created_at,
		        le.id, le.amount, le.direction, le.currency, le.created_at,
		        a.type
		 FROM ledger_transactions lt
		 JOIN ledger_entries le ON le.transaction_id = lt.id
		 JOIN accounts a ON a.id = le.account_id
		 WHERE lt.reference_id = $1
		 ORDER BY lt.created_at ASC, le.id ASC`,
		referenceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	index := map[uuid.UUID]*models.PayoutLedgerTransaction{}
	var order []uuid.UUID

	for rows.Next() {
		var (
			txID, entryID     uuid.UUID
			txType, direction, currency, accountType string
			txCreatedAt, entryCreatedAt              time.Time
			amount                                   int64
		)
		if err := rows.Scan(
			&txID, &txType, &txCreatedAt,
			&entryID, &amount, &direction, &currency, &entryCreatedAt,
			&accountType,
		); err != nil {
			return nil, err
		}

		if _, exists := index[txID]; !exists {
			index[txID] = &models.PayoutLedgerTransaction{
				ID:        txID,
				Type:      models.TransactionType(txType),
				CreatedAt: txCreatedAt,
			}
			order = append(order, txID)
		}
		index[txID].Entries = append(index[txID].Entries, &models.LedgerEntryWithAccount{
			ID:          entryID,
			Amount:      amount,
			Direction:   models.Direction(direction),
			Currency:    models.Currency(currency),
			AccountType: models.AccountType(accountType),
			CreatedAt:   entryCreatedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]*models.PayoutLedgerTransaction, 0, len(order))
	for _, id := range order {
		result = append(result, index[id])
	}
	return result, nil
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
