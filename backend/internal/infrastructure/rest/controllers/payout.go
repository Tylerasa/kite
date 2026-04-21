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
	logger *slog.Logger
}

func NewPayoutController(uc in.PayoutUseCase, logger *slog.Logger) *PayoutController {
	return &PayoutController{uc: uc, logger: logger}
}

type payoutRequest struct {
	SourceCurrency         string `json:"source_currency" binding:"required"`
	Amount                 int64  `json:"amount" binding:"required,min=1"`
	RecipientAccountNumber string `json:"recipient_account_number" binding:"required"`
	RecipientBankCode      string `json:"recipient_bank_code" binding:"required"`
	RecipientAccountName   string `json:"recipient_account_name" binding:"required"`
}

type payoutResponse struct {
	ID                     string       `json:"id"`
	SourceCurrency         string       `json:"source_currency"`
	Amount                 int64        `json:"amount"`
	Status                 string       `json:"status"`
	RecipientAccountNumber string       `json:"recipient_account_number"`
	RecipientBankCode      string       `json:"recipient_bank_code"`
	RecipientAccountName   string       `json:"recipient_account_name"`
	ComplianceFlagged      bool         `json:"compliance_flagged"`
	FailureReason          *string      `json:"failure_reason,omitempty"`
	ReversedAt             *time.Time   `json:"reversed_at,omitempty"`
	CreatedAt              time.Time    `json:"created_at"`
	UpdatedAt              time.Time    `json:"updated_at"`
}

func (ctrl *PayoutController) Create(c *gin.Context) {
	var req payoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code: "validation_error", Message: err.Error(),
		}})
		return
	}

	userID := middleware.GetUserID(c)

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

	c.JSON(status, toPayoutResponse(result.Payout))
}

func (ctrl *PayoutController) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)

	payoutID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code: "invalid_payout_id", Message: "payout ID must be a valid UUID.",
		}})
		return
	}

	payout, err := ctrl.uc.GetPayout(c.Request.Context(), userID, payoutID)
	if err != nil {
		apiutil.RespondError(c, ctrl.logger, err)
		return
	}

	c.JSON(http.StatusOK, toPayoutResponse(payout))
}

func toPayoutResponse(p *models.Payout) payoutResponse {
	return payoutResponse{
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
	}
}
