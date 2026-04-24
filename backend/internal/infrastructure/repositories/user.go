package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kite/internal/domain/exceptions"
	"github.com/kite/internal/domain/models"
	"golang.org/x/crypto/bcrypt"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, name, email, password_hash, pin_hash, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID, user.Name, user.Email, user.PasswordHash, user.PinHash, user.CreatedAt,
	)
	return err
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, name, email, password_hash, pin_hash, created_at FROM users WHERE email = $1`,
		email,
	)
	return scanUser(row)
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, name, email, password_hash, pin_hash, created_at FROM users WHERE id = $1`,
		id,
	)
	return scanUser(row)
}

func (r *UserRepo) VerifyPin(ctx context.Context, userID uuid.UUID, pin string) error {
	var pinHash string
	err := r.db.QueryRow(ctx, `SELECT pin_hash FROM users WHERE id = $1`, userID).Scan(&pinHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return exceptions.ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(pinHash), []byte(pin)); err != nil {
		return exceptions.ErrInvalidCredentials
	}
	return nil
}

func scanUser(row pgx.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.PinHash, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, exceptions.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}
