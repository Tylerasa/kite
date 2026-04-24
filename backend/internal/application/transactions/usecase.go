package transactions

import (
	"context"
	"math"

	"github.com/google/uuid"
	"github.com/kite/internal/domain/exceptions"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/domain/ports/out"
)

type UseCase struct {
	txRepo out.TransactionRepository
}

func NewUseCase(txRepo out.TransactionRepository) *UseCase {
	return &UseCase{txRepo: txRepo}
}

func (uc *UseCase) GetHistory(ctx context.Context, q in.HistoryQuery) (*in.HistoryResult, error) {
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	offset := (q.Page - 1) * q.Limit

	records, total, err := uc.txRepo.GetHistoryForUser(ctx, q.UserID, q.Limit, offset)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(q.Limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return &in.HistoryResult{
		Items:      records,
		Total:      total,
		Page:       q.Page,
		TotalPages: totalPages,
	}, nil
}

func (uc *UseCase) GetByID(ctx context.Context, userID, transactionID uuid.UUID) (*in.TransactionDetail, error) {
	tx, entries, err := uc.txRepo.GetByIDForUser(ctx, userID, transactionID)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, exceptions.ErrTransactionNotFound
	}
	return &in.TransactionDetail{Transaction: tx, Entries: entries}, nil
}
