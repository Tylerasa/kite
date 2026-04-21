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

type ConversionController struct {
	uc     in.ConversionUseCase
	logger *slog.Logger
}

func NewConversionController(uc in.ConversionUseCase, logger *slog.Logger) *ConversionController {
	return &ConversionController{uc: uc, logger: logger}
}

type quoteRequest struct {
	FromCurrency string `json:"from_currency" binding:"required"`
	ToCurrency   string `json:"to_currency" binding:"required"`
	AmountIn     int64  `json:"amount_in" binding:"required,min=1"`
}

type quoteResponse struct {
	ID           string    `json:"id"`
	FromCurrency string    `json:"from_currency"`
	ToCurrency   string    `json:"to_currency"`
	MarketRate   string    `json:"market_rate"`
	QuotedRate   string    `json:"quoted_rate"`
	AmountIn     int64     `json:"amount_in"`
	AmountOut    int64     `json:"amount_out"`
	Fee          int64     `json:"fee"`
	ExpiresAt    time.Time `json:"expires_at"`
	SecondsLeft  int       `json:"seconds_left"`
}

type executeRequest struct {
	QuoteID string `json:"quote_id" binding:"required"`
}

type conversionResponse struct {
	ID           string    `json:"id"`
	FromCurrency string    `json:"from_currency"`
	ToCurrency   string    `json:"to_currency"`
	AmountIn     int64     `json:"amount_in"`
	AmountOut    int64     `json:"amount_out"`
	QuotedRate   string    `json:"quoted_rate"`
	Fee          int64     `json:"fee"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

func (ctrl *ConversionController) CreateQuote(c *gin.Context) {
	var req quoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code: "validation_error", Message: err.Error(),
		}})
		return
	}

	userID := middleware.GetUserID(c)

	result, err := ctrl.uc.CreateQuote(c.Request.Context(), in.QuoteCommand{
		UserID:       userID,
		FromCurrency: models.Currency(req.FromCurrency),
		ToCurrency:   models.Currency(req.ToCurrency),
		AmountIn:     req.AmountIn,
	})
	if err != nil {
		apiutil.RespondError(c, ctrl.logger, err)
		return
	}

	q := result.Quote
	secondsLeft := int(time.Until(q.ExpiresAt).Seconds())
	if secondsLeft < 0 {
		secondsLeft = 0
	}

	c.JSON(http.StatusOK, quoteResponse{
		ID:           q.ID.String(),
		FromCurrency: string(q.FromCurrency),
		ToCurrency:   string(q.ToCurrency),
		MarketRate:   q.MarketRate,
		QuotedRate:   q.QuotedRate,
		AmountIn:     q.AmountIn,
		AmountOut:    q.AmountOut,
		Fee:          q.Fee,
		ExpiresAt:    q.ExpiresAt,
		SecondsLeft:  secondsLeft,
	})
}

func (ctrl *ConversionController) Execute(c *gin.Context) {
	var req executeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code: "validation_error", Message: err.Error(),
		}})
		return
	}

	quoteID, err := uuid.Parse(req.QuoteID)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code:    "invalid_quote_id",
			Message: "quote_id must be a valid UUID.",
		}})
		return
	}

	userID := middleware.GetUserID(c)

	result, err := ctrl.uc.ExecuteConversion(c.Request.Context(), in.ExecuteCommand{
		UserID:  userID,
		QuoteID: quoteID,
	})
	if err != nil {
		apiutil.RespondError(c, ctrl.logger, err)
		return
	}

	conv := result.Conversion
	c.JSON(http.StatusOK, conversionResponse{
		ID:           conv.ID.String(),
		FromCurrency: string(conv.FromCurrency),
		ToCurrency:   string(conv.ToCurrency),
		AmountIn:     conv.AmountIn,
		AmountOut:    conv.AmountOut,
		QuotedRate:   conv.QuotedRate,
		Fee:          conv.Fee,
		Status:       conv.Status,
		CreatedAt:    conv.CreatedAt,
	})
}
