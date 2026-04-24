package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kite/internal/domain/models"
)

type AuditLogRepo struct {
	db *pgxpool.Pool
}

func NewAuditLogRepo(db *pgxpool.Pool) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

func (r *AuditLogRepo) Create(ctx context.Context, entry *models.AuditLog) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO audit_log
		 (id, user_id, operation, reference_id, status, error_code, request_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
		entry.ID, entry.UserID, entry.Operation, entry.ReferenceID,
		entry.Status, entry.ErrorCode, entry.RequestID, entry.CreatedAt,
	)
	return err
}

func (r *AuditLogRepo) Update(ctx context.Context, id uuid.UUID, status string, errorCode *string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE audit_log SET status = $2, error_code = $3, updated_at = $4 WHERE id = $1`,
		id, status, errorCode, time.Now().UTC(),
	)
	return err
}
