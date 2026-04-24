package payouts

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kite/internal/domain/exceptions"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/domain/ports/out"
	"github.com/kite/internal/domain/services"
)

type UseCase struct {
	payouts    out.PayoutRepository
	accounts   out.AccountRepository
	ledger     out.LedgerRepository
	audit      out.AuditLogRepository
	compliance int64 // NGN threshold in kobo (minor units)
	guard      *services.TransactionGuard
}

func NewUseCase(
	payouts out.PayoutRepository,
	accounts out.AccountRepository,
	ledger out.LedgerRepository,
	complianceNGNThreshold int64,
	guard *services.TransactionGuard,
	audit out.AuditLogRepository,
) *UseCase {
	return &UseCase{
		payouts:    payouts,
		accounts:   accounts,
		ledger:     ledger,
		audit:      audit,
		compliance: complianceNGNThreshold,
		guard:      guard,
	}
}

func (uc *UseCase) InitiatePayout(ctx context.Context, cmd in.PayoutCommand) (result *in.PayoutResult, err error) {
	logID := uuid.New()
	auditNow := time.Now().UTC()
	_ = uc.audit.Create(ctx, &models.AuditLog{
		ID: logID, UserID: cmd.UserID, Operation: "payout",
		Status: "pending", RequestID: services.RequestIDFromCtx(ctx),
		CreatedAt: auditNow, UpdatedAt: auditNow,
	})
	defer func() {
		status, errCode := "success", (*string)(nil)
		if err != nil {
			status = "failure"
			if de, ok := err.(*exceptions.DomainError); ok {
				errCode = &de.Code
			}
		}
		if auditErr := uc.audit.Update(ctx, logID, status, errCode); auditErr != nil {
			slog.Error("audit update failed", "op", "payout", "error", auditErr)
		}
	}()

	if !cmd.SourceCurrency.Valid() {
		return nil, exceptions.ErrInvalidCurrency
	}
	if cmd.Amount <= 0 {
		return nil, exceptions.ErrInvalidCurrency.WithDetails(map[string]interface{}{
			"field": "amount", "reason": "must be greater than 0",
		})
	}
	if cmd.RecipientAccountNumber == "" || cmd.RecipientBankCode == "" || cmd.RecipientAccountName == "" {
		return nil, exceptions.ErrInternal.WithDetails(map[string]interface{}{
			"reason": "recipient details are required",
		})
	}

	// Duplicate guard: reject identical payout within 30 seconds.
	if err := uc.guard.CheckAndLock(services.PayoutFingerprint(
		cmd.UserID.String(), string(cmd.SourceCurrency), cmd.Amount,
		cmd.RecipientAccountNumber, cmd.RecipientBankCode,
	)...); err != nil {
		return nil, err
	}

	// Check balance before holding.
	userWallet, err := uc.accounts.GetByUserAndCurrency(ctx, cmd.UserID, cmd.SourceCurrency, models.AccountTypeUserWallet)
	if err != nil {
		return nil, err
	}
	balance, err := uc.ledger.GetBalanceForAccount(ctx, userWallet.ID)
	if err != nil {
		return nil, fmt.Errorf("check balance: %w", err)
	}
	if balance < cmd.Amount {
		return nil, exceptions.ErrInsufficientFunds
	}

	now := time.Now().UTC()
	payout := &models.Payout{
		ID:                     uuid.New(),
		UserID:                 cmd.UserID,
		SourceCurrency:         cmd.SourceCurrency,
		Amount:                 cmd.Amount,
		Status:                 models.PayoutStatusPending,
		RecipientAccountNumber: cmd.RecipientAccountNumber,
		RecipientBankCode:      cmd.RecipientBankCode,
		RecipientAccountName:   cmd.RecipientAccountName,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	// Compliance rule: NGN payouts above threshold go to review.
	if cmd.SourceCurrency == models.NGN && cmd.Amount > uc.compliance {
		payout.Status = models.PayoutStatusReview
		payout.ComplianceFlagged = true
	}

	ledgerTx, entries, err := uc.buildHoldEntries(ctx, payout.ID, userWallet.ID, cmd.SourceCurrency, cmd.Amount, now)
	if err != nil {
		return nil, err
	}

	if err := uc.payouts.CreateWithHold(ctx, payout, ledgerTx, entries); err != nil {
		return nil, fmt.Errorf("create payout with hold: %w", err)
	}

	return &in.PayoutResult{Payout: payout}, nil
}

func (uc *UseCase) GetPayout(ctx context.Context, userID, payoutID uuid.UUID) (*in.PayoutDetail, error) {
	p, err := uc.payouts.GetByID(ctx, payoutID)
	if err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, exceptions.ErrPayoutNotFound // don't leak other users' payouts
	}
	ledger, err := uc.ledger.GetByReference(ctx, payoutID)
	if err != nil {
		return nil, fmt.Errorf("get ledger for payout: %w", err)
	}
	return &in.PayoutDetail{Payout: p, Ledger: ledger}, nil
}

// ProcessPayout is called by the background job to simulate payout execution.
// The job uses ClaimPending which atomically sets status to processing.
func (uc *UseCase) ProcessPayout(ctx context.Context, payoutID uuid.UUID) error {
	payout, err := uc.payouts.GetByID(ctx, payoutID)
	if err != nil {
		return err
	}

	// Simulate: 80% success, 20% failure (deterministic for testing via ForceFailure).
	success := simulateSuccess(payout.ID)

	if success {
		if err := uc.settleSuccess(ctx, payout); err != nil {
			return err
		}
		return uc.payouts.UpdateStatus(ctx, payoutID, models.PayoutStatusSuccessful, nil)
	}

	reason := "payment failed at provider"
	if err := uc.payouts.UpdateStatus(ctx, payoutID, models.PayoutStatusFailed, &reason); err != nil {
		return err
	}
	return uc.ReversePayout(ctx, payoutID)
}

// ReversePayout writes reversal ledger entries to restore the user's balance.
func (uc *UseCase) ReversePayout(ctx context.Context, payoutID uuid.UUID) error {
	payout, err := uc.payouts.GetByID(ctx, payoutID)
	if err != nil {
		return err
	}

	userWallet, err := uc.accounts.GetByUserAndCurrency(ctx, payout.UserID, payout.SourceCurrency, models.AccountTypeUserWallet)
	if err != nil {
		return fmt.Errorf("get user wallet for reversal: %w", err)
	}

	payoutPending, err := uc.accounts.GetByTypeAndCurrency(ctx, models.AccountTypePayoutPending, payout.SourceCurrency)
	if err != nil {
		return fmt.Errorf("get payout_pending for reversal: %w", err)
	}

	now := time.Now().UTC()
	ledgerTxID := uuid.New()
	ledgerTx := &models.LedgerTransaction{
		ID:          ledgerTxID,
		Type:        models.TxTypeReversal,
		ReferenceID: payoutID,
		CreatedAt:   now,
	}
	if err := uc.ledger.CreateTransaction(ctx, ledgerTx); err != nil {
		return fmt.Errorf("create reversal ledger tx: %w", err)
	}

	// Reversal: mirror of the hold, reversed direction.
	entries := []*models.LedgerEntry{
		{
			ID: uuid.New(), TransactionID: ledgerTxID, AccountID: payoutPending.ID,
			Amount: payout.Amount, Direction: models.Debit,
			Currency: payout.SourceCurrency, CreatedAt: now,
		},
		{
			ID: uuid.New(), TransactionID: ledgerTxID, AccountID: userWallet.ID,
			Amount: payout.Amount, Direction: models.Credit,
			Currency: payout.SourceCurrency, CreatedAt: now,
		},
	}
	if err := uc.ledger.CreateEntries(ctx, entries); err != nil {
		return fmt.Errorf("write reversal entries: %w", err)
	}

	return uc.payouts.MarkReversed(ctx, payoutID)
}

func (uc *UseCase) buildHoldEntries(ctx context.Context, payoutID, userWalletID uuid.UUID, currency models.Currency, amount int64, now time.Time) (*models.LedgerTransaction, []*models.LedgerEntry, error) {
	payoutPending, err := uc.accounts.GetByTypeAndCurrency(ctx, models.AccountTypePayoutPending, currency)
	if err != nil {
		return nil, nil, fmt.Errorf("get payout_pending account: %w", err)
	}

	ledgerTxID := uuid.New()
	ledgerTx := &models.LedgerTransaction{
		ID:          ledgerTxID,
		Type:        models.TxTypePayout,
		ReferenceID: payoutID,
		CreatedAt:   now,
	}
	entries := []*models.LedgerEntry{
		{
			ID: uuid.New(), TransactionID: ledgerTxID, AccountID: userWalletID,
			Amount: amount, Direction: models.Debit, Currency: currency, CreatedAt: now,
		},
		{
			ID: uuid.New(), TransactionID: ledgerTxID, AccountID: payoutPending.ID,
			Amount: amount, Direction: models.Credit, Currency: currency, CreatedAt: now,
		},
	}
	return ledgerTx, entries, nil
}

func (uc *UseCase) settleSuccess(ctx context.Context, payout *models.Payout) error {
	payoutPending, err := uc.accounts.GetByTypeAndCurrency(ctx, models.AccountTypePayoutPending, payout.SourceCurrency)
	if err != nil {
		return fmt.Errorf("get payout_pending: %w", err)
	}
	systemCash, err := uc.accounts.GetByTypeAndCurrency(ctx, models.AccountTypeSystemCash, payout.SourceCurrency)
	if err != nil {
		return fmt.Errorf("get system cash: %w", err)
	}

	now := time.Now().UTC()
	ledgerTxID := uuid.New()
	ledgerTx := &models.LedgerTransaction{
		ID: ledgerTxID, Type: models.TxTypePayout, ReferenceID: payout.ID, CreatedAt: now,
	}
	if err := uc.ledger.CreateTransaction(ctx, ledgerTx); err != nil {
		return err
	}

	entries := []*models.LedgerEntry{
		{
			ID: uuid.New(), TransactionID: ledgerTxID, AccountID: payoutPending.ID,
			Amount: payout.Amount, Direction: models.Debit, Currency: payout.SourceCurrency, CreatedAt: now,
		},
		{
			ID: uuid.New(), TransactionID: ledgerTxID, AccountID: systemCash.ID,
			Amount: payout.Amount, Direction: models.Credit, Currency: payout.SourceCurrency, CreatedAt: now,
		},
	}
	return uc.ledger.CreateEntries(ctx, entries)
}

// simulateSuccess returns false deterministically for testing — in prod this is random.
// Tests can force failure by using a specific payout ID.
var forceFailIDs = map[uuid.UUID]bool{}

// ForceFailPayout registers a payout ID to always fail (for tests).
func ForceFailPayout(id uuid.UUID) {
	forceFailIDs[id] = true
}

func simulateSuccess(id uuid.UUID) bool {
	if forceFailIDs[id] {
		return false
	}
	// Use last byte of UUID for 80/20 split.
	return id[15]%5 != 0
}
