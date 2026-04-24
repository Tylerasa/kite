package controllers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/infrastructure/rest/apiutil"
	"github.com/kite/internal/infrastructure/rest/middleware"
)

type PayoutController struct {
	uc     in.PayoutUseCase
	auth   in.AuthUseCase
	logger *slog.Logger
}

func NewPayoutController(uc in.PayoutUseCase, auth in.AuthUseCase, logger *slog.Logger) *PayoutController {
	return &PayoutController{uc: uc, auth: auth, logger: logger}
}

type payoutRequest struct {
	SourceCurrency         string `json:"source_currency" binding:"required,len=3"`
	Amount                 int64  `json:"amount" binding:"required,min=1"`
	RecipientAccountNumber string `json:"recipient_account_number" binding:"required,max=20"`
	RecipientBankCode      string `json:"recipient_bank_code" binding:"required,max=10"`
	RecipientAccountName   string `json:"recipient_account_name" binding:"required,max=200"`
	Pin                    string `json:"pin" binding:"required,min=4,max=6"`
}

type ledgerEntryResponse struct {
	Amount      int64  `json:"amount"`
	Direction   string `json:"direction"`
	Currency    string `json:"currency"`
	AccountType string `json:"account_type"`
}

type ledgerTxResponse struct {
	ID        string                `json:"id"`
	Type      string                `json:"type"`
	CreatedAt time.Time             `json:"created_at"`
	Entries   []ledgerEntryResponse `json:"entries"`
}

type payoutResponse struct {
	ID                     string             `json:"id"`
	SourceCurrency         string             `json:"source_currency"`
	Amount                 int64              `json:"amount"`
	Status                 string             `json:"status"`
	RecipientAccountNumber string             `json:"recipient_account_number"`
	RecipientBankCode      string             `json:"recipient_bank_code"`
	RecipientAccountName   string             `json:"recipient_account_name"`
	ComplianceFlagged      bool               `json:"compliance_flagged"`
	FailureReason          *string            `json:"failure_reason,omitempty"`
	ReversedAt             *time.Time         `json:"reversed_at,omitempty"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
	Ledger                 []ledgerTxResponse `json:"ledger"`
}

// Create godoc
// @Summary      Initiate a payout to a bank account
// @Description  Holds funds immediately. Settlement is async (poll GET /payouts/:id). NGN payouts over ₦500,000 are placed in compliance review (202).
// @Tags         payouts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      payoutRequest  true  "Payout details"
// @Success      201   {object}  payoutResponse         "Payout accepted"
// @Success      202   {object}  payoutResponse         "Compliance review — payout held"
// @Failure      400   {object}  apiutil.ErrorResponse
// @Failure      401   {object}  apiutil.ErrorResponse
// @Failure      422   {object}  apiutil.ErrorResponse  "insufficient_funds"
// @Router       /payouts [post]
func (ctrl *PayoutController) Create(c *gin.Context) {
	var req payoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code: "validation_error", Message: apiutil.BindingError(err),
		}})
		return
	}

	userID := middleware.GetUserID(c)

	if err := ctrl.auth.VerifyPin(c.Request.Context(), userID, req.Pin); err != nil {
		c.JSON(http.StatusForbidden, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code: "invalid_pin", Message: "Incorrect PIN.",
		}})
		return
	}

	result, err := ctrl.uc.InitiatePayout(c.Request.Context(), in.PayoutCommand{
		UserID:                 userID,
		SourceCurrency:         models.Currency(req.SourceCurrency),
		Amount:                 req.Amount,
		RecipientAccountNumber: req.RecipientAccountNumber,
		RecipientBankCode:      req.RecipientBankCode,
		RecipientAccountName:   req.RecipientAccountName,
	})
	if err != nil {
		apiutil.RespondError(c, ctrl.logger, err)
		return
	}

	status := http.StatusCreated
	if result.Payout.ComplianceFlagged {
		status = http.StatusAccepted // 202 for compliance review
	}

	c.JSON(status, toPayoutResponse(&in.PayoutDetail{Payout: result.Payout}))
}

// Get godoc
// @Summary      Get payout status
// @Description  Poll this endpoint after initiating a payout. Status transitions: pending → processing → successful | failed | review.
// @Tags         payouts
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Payout UUID"
// @Success      200  {object}  payoutResponse
// @Failure      400  {object}  apiutil.ErrorResponse  "invalid UUID"
// @Failure      401  {object}  apiutil.ErrorResponse
// @Failure      404  {object}  apiutil.ErrorResponse  "payout not found"
// @Router       /payouts/{id} [get]
func (ctrl *PayoutController) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)

	payoutID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code: "invalid_payout_id", Message: "payout ID must be a valid UUID.",
		}})
		return
	}

	detail, err := ctrl.uc.GetPayout(c.Request.Context(), userID, payoutID)
	if err != nil {
		apiutil.RespondError(c, ctrl.logger, err)
		return
	}

	c.JSON(http.StatusOK, toPayoutResponse(detail))
}

func toPayoutResponse(d *in.PayoutDetail) payoutResponse {
	p := d.Payout
	resp := payoutResponse{
		ID:                     p.ID.String(),
		SourceCurrency:         string(p.SourceCurrency),
		Amount:                 p.Amount,
		Status:                 string(p.Status),
		RecipientAccountNumber: p.RecipientAccountNumber,
		RecipientBankCode:      p.RecipientBankCode,
		RecipientAccountName:   p.RecipientAccountName,
		ComplianceFlagged:      p.ComplianceFlagged,
		FailureReason:          p.FailureReason,
		ReversedAt:             p.ReversedAt,
		CreatedAt:              p.CreatedAt,
		UpdatedAt:              p.UpdatedAt,
		Ledger:                 make([]ledgerTxResponse, 0, len(d.Ledger)),
	}
	for _, tx := range d.Ledger {
		txResp := ledgerTxResponse{
			ID:        tx.ID.String(),
			Type:      string(tx.Type),
			CreatedAt: tx.CreatedAt,
			Entries:   make([]ledgerEntryResponse, 0, len(tx.Entries)),
		}
		for _, e := range tx.Entries {
			txResp.Entries = append(txResp.Entries, ledgerEntryResponse{
				Amount:      e.Amount,
				Direction:   string(e.Direction),
				Currency:    string(e.Currency),
				AccountType: string(e.AccountType),
			})
		}
		resp.Ledger = append(resp.Ledger, txResp)
	}
	return resp
}
