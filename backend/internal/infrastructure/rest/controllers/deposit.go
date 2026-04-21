package controllers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/infrastructure/rest/apiutil"
	"github.com/kite/internal/infrastructure/rest/middleware"
)

type DepositController struct {
	uc     in.DepositUseCase
	logger *slog.Logger
}

func NewDepositController(uc in.DepositUseCase, logger *slog.Logger) *DepositController {
	return &DepositController{uc: uc, logger: logger}
}

type depositRequest struct {
	Currency string `json:"currency" binding:"required"`
	Amount   int64  `json:"amount" binding:"required,min=1"`
}

type depositResponse struct {
	ID             string `json:"id"`
	Currency       string `json:"currency"`
	Amount         int64  `json:"amount"`
	Status         string `json:"status"`
	IdempotencyKey string `json:"idempotency_key"`
	CreatedAt      string `json:"created_at"`
}

func (ctrl *DepositController) Create(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code:    "missing_idempotency_key",
			Message: "The Idempotency-Key header is required for deposits.",
		}})
		return
	}

	var req depositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code: "validation_error", Message: err.Error(),
		}})
		return
	}

	userID := middleware.GetUserID(c)

	result, err := ctrl.uc.Deposit(c.Request.Context(), in.DepositCommand{
		UserID:         userID,
		IdempotencyKey: idempotencyKey,
		Currency:       models.Currency(req.Currency),
		Amount:         req.Amount,
	})
	if err != nil {
		apiutil.RespondError(c, ctrl.logger, err)
		return
	}

	resp := depositResponse{
		ID:             result.Deposit.ID.String(),
		Currency:       string(result.Deposit.Currency),
		Amount:         result.Deposit.Amount,
		Status:         result.Deposit.Status,
		IdempotencyKey: result.Deposit.IdempotencyKey,
		CreatedAt:      result.Deposit.CreatedAt.String(),
	}

	status := http.StatusCreated
	if !result.IsNew {
		status = http.StatusOK // idempotent: return existing deposit
	}
	c.JSON(status, resp)
}
