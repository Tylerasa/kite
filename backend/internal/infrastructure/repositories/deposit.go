package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kite/internal/domain/models"
)

type DepositRepo struct {
	db *pgxpool.Pool
}

func NewDepositRepo(db *pgxpool.Pool) *DepositRepo {
	return &DepositRepo{db: db}
}

// CreateIfNotExists inserts if the idempotency_key is new.
// Returns (new deposit, true, nil) on insert; (existing, false, nil) on conflict.
func (r *DepositRepo) CreateIfNotExists(ctx context.Context, deposit *models.Deposit) (*models.Deposit, bool, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO deposits (id, user_id, idempotency_key, currency, amount, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (user_id, idempotency_key) DO UPDATE SET idempotency_key = EXCLUDED.idempotency_key
		 RETURNING id, user_id, idempotency_key, currency, amount, status, created_at`,
		deposit.ID, deposit.UserID, deposit.IdempotencyKey,
		string(deposit.Currency), deposit.Amount, deposit.Status, deposit.CreatedAt,
	)

	var d models.Deposit
	var currency string
	err := row.Scan(&d.ID, &d.UserID, &d.IdempotencyKey, &currency, &d.Amount, &d.Status, &d.CreatedAt)
	if err != nil {
		return nil, false, err
	}
	d.Currency = models.Currency(currency)

	// If the returned ID matches what we tried to insert, it's new.
	isNew := d.ID == deposit.ID
	return &d, isNew, nil
}

func (r *DepositRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Deposit, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, user_id, idempotency_key, currency, amount, status, created_at
		 FROM deposits WHERE id = $1`,
		id,
	)
	var d models.Deposit
	var currency string
	err := row.Scan(&d.ID, &d.UserID, &d.IdempotencyKey, &currency, &d.Amount, &d.Status, &d.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	d.Currency = models.Currency(currency)
	return &d, nil
}
