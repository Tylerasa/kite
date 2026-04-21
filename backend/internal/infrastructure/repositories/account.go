package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kite/internal/domain/exceptions"
	"github.com/kite/internal/domain/models"
)

type AccountRepo struct {
	db *pgxpool.Pool
}

func NewAccountRepo(db *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{db: db}
}

func (r *AccountRepo) Create(ctx context.Context, account *models.Account) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO accounts (id, user_id, currency, type, name, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		account.ID, account.UserID, string(account.Currency),
		string(account.Type), account.Name, account.CreatedAt,
	)
	return err
}

func (r *AccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Account, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, user_id, currency, type, name, created_at FROM accounts WHERE id = $1`,
		id,
	)
	return scanAccount(row)
}

func (r *AccountRepo) GetByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency models.Currency, accountType models.AccountType) (*models.Account, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, user_id, currency, type, name, created_at
		 FROM accounts WHERE user_id = $1 AND currency = $2 AND type = $3`,
		userID, string(currency), string(accountType),
	)
	return scanAccount(row)
}

func (r *AccountRepo) GetByTypeAndCurrency(ctx context.Context, accountType models.AccountType, currency models.Currency) (*models.Account, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, user_id, currency, type, name, created_at
		 FROM accounts WHERE user_id IS NULL AND type = $1 AND currency = $2`,
		string(accountType), string(currency),
	)
	return scanAccount(row)
}

func (r *AccountRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.Account, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, currency, type, name, created_at
		 FROM accounts WHERE user_id = $1 ORDER BY currency`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*models.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func scanAccount(row pgx.Row) (*models.Account, error) {
	var a models.Account
	var currency, accountType string
	err := row.Scan(&a.ID, &a.UserID, &currency, &accountType, &a.Name, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, exceptions.ErrAccountNotFound
		}
		return nil, err
	}
	a.Currency = models.Currency(currency)
	a.Type = models.AccountType(accountType)
	return &a, nil
}
