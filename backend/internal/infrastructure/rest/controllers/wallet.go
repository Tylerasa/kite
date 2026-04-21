package controllers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
