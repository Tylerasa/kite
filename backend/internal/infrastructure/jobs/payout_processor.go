package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/domain/ports/out"
	"golang.org/x/sync/semaphore"
)

// PayoutProcessor is the background job that transitions payouts through their lifecycle.
// It atomically claims pending payouts using FOR UPDATE SKIP LOCKED so multiple
// instances won't double-process the same payout. Concurrent processing is bounded
// by maxConcurrency (semaphore pattern from Cinnamon's SimpleBatchProcessor).
type PayoutProcessor struct {
	payoutUC       in.PayoutUseCase
	payoutRepo     out.PayoutRepository
	logger         *slog.Logger
	interval       time.Duration
	maxConcurrency int
}

func NewPayoutProcessor(payoutUC in.PayoutUseCase, payoutRepo out.PayoutRepository, logger *slog.Logger, maxConcurrency int) *PayoutProcessor {
	return &PayoutProcessor{
		payoutUC:       payoutUC,
		payoutRepo:     payoutRepo,
		logger:         logger,
		interval:       5 * time.Second,
		maxConcurrency: maxConcurrency,
	}
}

// Start runs the processor loop. Called from the FX OnStart lifecycle hook.
func (p *PayoutProcessor) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	p.logger.Info("payout processor started", "max_concurrency", p.maxConcurrency)

	for {
		select {
		case <-ticker.C:
			p.processBatch(ctx)
		case <-ctx.Done():
			p.logger.Info("payout processor stopped")
			return
		}
	}
}

func (p *PayoutProcessor) processBatch(ctx context.Context) {
	// Collect all currently claimable pending payouts.
	var batch []*models.Payout
	for {
		payout, err := p.payoutRepo.ClaimPending(ctx)
		if err != nil {
			p.logger.Error("claim pending payout", "error", err)
			break
		}
		if payout == nil {
			break // queue empty
		}
		batch = append(batch, payout)
	}

	if len(batch) == 0 {
		return
	}

	p.logger.Info("processing payout batch", "count", len(batch))

	// Process with bounded concurrency via semaphore.
	sem := semaphore.NewWeighted(int64(p.maxConcurrency))
	for _, payout := range batch {
		if err := sem.Acquire(ctx, 1); err != nil {
			return // context cancelled
		}
		go func(id uuid.UUID) {
			defer sem.Release(1)
			p.processOne(id)
		}(payout.ID)
	}

	// Wait for all goroutines in this batch to finish.
	_ = sem.Acquire(ctx, int64(p.maxConcurrency))
}

func (p *PayoutProcessor) processOne(payoutID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Simulate network latency (2–5 seconds based on payout ID byte).
	delay := time.Duration(2+int(payoutID[15])%4) * time.Second
	time.Sleep(delay)

	if err := p.payoutUC.ProcessPayout(ctx, payoutID); err != nil {
		p.logger.Error("process payout", "payout_id", payoutID, "error", err)
	} else {
		p.logger.Info("payout processed", "payout_id", payoutID)
	}
}
