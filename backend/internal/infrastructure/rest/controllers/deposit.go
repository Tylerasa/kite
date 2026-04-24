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
	auth   in.AuthUseCase
	logger *slog.Logger
}

func NewDepositController(uc in.DepositUseCase, auth in.AuthUseCase, logger *slog.Logger) *DepositController {
	return &DepositController{uc: uc, auth: auth, logger: logger}
}

type depositRequest struct {
	Currency string `json:"currency" binding:"required,len=3"`
	Amount   int64  `json:"amount" binding:"required,min=1"`
	Pin      string `json:"pin" binding:"required,min=4,max=6"`
}

type depositResponse struct {
	ID             string `json:"id"`
	Currency       string `json:"currency"`
	Amount         int64  `json:"amount"`
	Status         string `json:"status"`
	IdempotencyKey string `json:"idempotency_key"`
	CreatedAt      string `json:"created_at"`
}

// Create godoc
// @Summary      Simulate a deposit into a currency wallet
// @Description  Idempotent — repeated calls with the same Idempotency-Key return the original deposit (HTTP 200) without moving money twice.
// @Tags         deposits
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Idempotency-Key  header    string          true  "Unique key per deposit attempt (e.g. crypto.randomUUID())"
// @Param        body             body      depositRequest  true  "Deposit details"
// @Success      201              {object}  depositResponse        "New deposit created"
// @Success      200              {object}  depositResponse        "Duplicate — existing deposit returned"
// @Failure      400              {object}  apiutil.ErrorResponse
// @Failure      401              {object}  apiutil.ErrorResponse
// @Failure      422              {object}  apiutil.ErrorResponse  "invalid currency"
// @Router       /deposits [post]
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
