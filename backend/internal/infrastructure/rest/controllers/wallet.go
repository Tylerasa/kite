package controllers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/infrastructure/rest/apiutil"
	"github.com/kite/internal/infrastructure/rest/middleware"
)

type WalletController struct {
	walletUC in.WalletUseCase
	txUC     in.TransactionUseCase
	logger   *slog.Logger
}

func NewWalletController(walletUC in.WalletUseCase, txUC in.TransactionUseCase, logger *slog.Logger) *WalletController {
	return &WalletController{walletUC: walletUC, txUC: txUC, logger: logger}
}

type balanceItem struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
	Display  string `json:"display"` // human-readable e.g. "100.00"
}

type balancesResponse struct {
	Balances []balanceItem `json:"balances"`
}

type transactionsResponse struct {
	Items      any `json:"items"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	TotalPages int `json:"total_pages"`
}

type transactionEntryResponse struct {
	ID          string    `json:"id"`
	Amount      int64     `json:"amount"`
	Direction   string    `json:"direction"`
	Currency    string    `json:"currency"`
	AccountType string    `json:"account_type"`
	CreatedAt   time.Time `json:"created_at"`
}

type transactionDetailResponse struct {
	ID          string                     `json:"id"`
	Type        string                     `json:"type"`
	ReferenceID string                     `json:"reference_id"`
	CreatedAt   time.Time                  `json:"created_at"`
	Entries     []transactionEntryResponse `json:"entries"`
}

// GetBalances godoc
// @Summary      List wallet balances for all currencies
// @Tags         wallet
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  balancesResponse
// @Failure      401  {object}  apiutil.ErrorResponse
// @Router       /wallets/balances [get]
func (ctrl *WalletController) GetBalances(c *gin.Context) {
	userID := middleware.GetUserID(c)

	balances, err := ctrl.walletUC.GetBalances(c.Request.Context(), userID)
	if err != nil {
		apiutil.RespondError(c, ctrl.logger, err)
		return
	}

	items := make([]balanceItem, 0, len(balances))
	for _, b := range balances {
		items = append(items, balanceItem{
			Currency: string(b.Currency),
			Amount:   b.Amount,
			Display:  formatMinorUnits(b.Amount, string(b.Currency)),
		})
	}

	c.JSON(http.StatusOK, gin.H{"balances": items})
}

// GetTransactions godoc
// @Summary      Paginated transaction history
// @Tags         wallet
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int  false  "Page number (default 1)"
// @Param        limit  query     int  false  "Items per page (default 20, max 100)"
// @Success      200    {object}  transactionsResponse
// @Failure      401    {object}  apiutil.ErrorResponse
// @Router       /wallets/transactions [get]
func (ctrl *WalletController) GetTransactions(c *gin.Context) {
	userID := middleware.GetUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	result, err := ctrl.txUC.GetHistory(c.Request.Context(), in.HistoryQuery{
		UserID: userID,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		apiutil.RespondError(c, ctrl.logger, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       result.Items,
		"total":       result.Total,
		"page":        result.Page,
		"total_pages": result.TotalPages,
	})
}

// GetTransaction godoc
// @Summary      Get a transaction with ledger entries
// @Tags         wallet
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Transaction UUID"
// @Success      200  {object}  transactionDetailResponse
// @Failure      400  {object}  apiutil.ErrorResponse
// @Failure      401  {object}  apiutil.ErrorResponse
// @Failure      404  {object}  apiutil.ErrorResponse
// @Router       /wallets/transactions/{id} [get]
func (ctrl *WalletController) GetTransaction(c *gin.Context) {
	transactionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code: "invalid_transaction_id", Message: "transaction ID must be a valid UUID.",
		}})
		return
	}

	detail, err := ctrl.txUC.GetByID(c.Request.Context(), middleware.GetUserID(c), transactionID)
	if err != nil {
		apiutil.RespondError(c, ctrl.logger, err)
		return
	}

	c.JSON(http.StatusOK, toTransactionDetailResponse(detail.Transaction, detail.Entries))
}

func toTransactionDetailResponse(tx *models.LedgerTransaction, entries []*models.LedgerEntryWithAccount) transactionDetailResponse {
	resp := transactionDetailResponse{
		ID:          tx.ID.String(),
		Type:        string(tx.Type),
		ReferenceID: tx.ReferenceID.String(),
		CreatedAt:   tx.CreatedAt,
		Entries:     make([]transactionEntryResponse, 0, len(entries)),
	}
	for _, entry := range entries {
		resp.Entries = append(resp.Entries, transactionEntryResponse{
			ID:          entry.ID.String(),
			Amount:      entry.Amount,
			Direction:   string(entry.Direction),
			Currency:    string(entry.Currency),
			AccountType: string(entry.AccountType),
			CreatedAt:   entry.CreatedAt,
		})
	}
	return resp
}

// formatMinorUnits formats an int64 minor unit amount as a decimal string.
// All currencies supported here use 2 decimal places.
func formatMinorUnits(amount int64, _ string) string {
	whole := amount / 100
	frac := amount % 100
	if frac < 0 {
		frac = -frac
	}
	return strconv.FormatInt(whole, 10) + "." + zeroPad(frac)
}

func zeroPad(n int64) string {
	if n < 10 {
		return "0" + strconv.FormatInt(n, 10)
	}
	return strconv.FormatInt(n, 10)
}
