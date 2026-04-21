package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/infrastructure/repositories"
)

// PayoutProcessor is the background job that transitions payouts through their lifecycle.
// It atomically claims pending payouts using FOR UPDATE SKIP LOCKED so multiple
// instances won't double-process the same payout.
type PayoutProcessor struct {
	payoutUC   in.PayoutUseCase
	payoutRepo *repositories.PayoutRepo
	logger     *slog.Logger
	interval   time.Duration
}

func NewPayoutProcessor(payoutUC in.PayoutUseCase, payoutRepo *repositories.PayoutRepo, logger *slog.Logger) *PayoutProcessor {
	return &PayoutProcessor{
		payoutUC:   payoutUC,
		payoutRepo: payoutRepo,
		logger:     logger,
		interval:   5 * time.Second,
	}
}

// Start runs the processor loop. Called from the FX OnStart lifecycle hook.
func (p *PayoutProcessor) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	p.logger.Info("payout processor started")

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
	for {
		payout, err := p.payoutRepo.ClaimPending(ctx)
		if err != nil {
			p.logger.Error("claim pending payout", "error", err)
			return
		}
		if payout == nil {
			return // no more pending
		}

		go p.processOne(payout.ID)
	}
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
