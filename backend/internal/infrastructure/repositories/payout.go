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

type PayoutRepo struct {
	db *pgxpool.Pool
}

func NewPayoutRepo(db *pgxpool.Pool) *PayoutRepo {
	return &PayoutRepo{db: db}
}

func (r *PayoutRepo) Create(ctx context.Context, p *models.Payout) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO payouts
		 (id, user_id, source_currency, amount, status,
		  recipient_account_number, recipient_bank_code, recipient_account_name,
		  compliance_flagged, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		p.ID, p.UserID, string(p.SourceCurrency), p.Amount, string(p.Status),
		p.RecipientAccountNumber, p.RecipientBankCode, p.RecipientAccountName,
		p.ComplianceFlagged, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *PayoutRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Payout, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, user_id, source_currency, amount, status,
		        recipient_account_number, recipient_bank_code, recipient_account_name,
		        compliance_flagged, failure_reason, reversed_at, created_at, updated_at
		 FROM payouts WHERE id = $1`,
		id,
	)
	return scanPayout(row)
}

// ClaimPending atomically moves one pending payout to processing.
// Returns nil if no pending payout is available.
func (r *PayoutRepo) ClaimPending(ctx context.Context) (*models.Payout, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE payouts SET status = 'processing', updated_at = NOW()
		 WHERE id = (
		     SELECT id FROM payouts WHERE status = 'pending'
		     ORDER BY created_at ASC
		     LIMIT 1
		     FOR UPDATE SKIP LOCKED
		 )
		 RETURNING id, user_id, source_currency, amount, status,
		           recipient_account_number, recipient_bank_code, recipient_account_name,
		           compliance_flagged, failure_reason, reversed_at, created_at, updated_at`,
	)
	p, err := scanPayout(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *PayoutRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status models.PayoutStatus, failureReason *string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE payouts SET status = $2, failure_reason = $3, updated_at = NOW() WHERE id = $1`,
		id, string(status), failureReason,
	)
	return err
}

func (r *PayoutRepo) MarkReversed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE payouts SET reversed_at = NOW(), updated_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}

func (r *PayoutRepo) MarkComplianceReview(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE payouts SET status = 'review', compliance_flagged = TRUE, updated_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}

func scanPayout(row pgx.Row) (*models.Payout, error) {
	var p models.Payout
	var currency, status string
	err := row.Scan(
		&p.ID, &p.UserID, &currency, &p.Amount, &status,
		&p.RecipientAccountNumber, &p.RecipientBankCode, &p.RecipientAccountName,
		&p.ComplianceFlagged, &p.FailureReason, &p.ReversedAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, exceptions.ErrPayoutNotFound
		}
		return nil, err
	}
	p.SourceCurrency = models.Currency(currency)
	p.Status = models.PayoutStatus(status)
	return &p, nil
}
