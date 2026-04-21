package rest

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/kite/internal/infrastructure/rest/controllers"
	"github.com/kite/internal/infrastructure/rest/middleware"
)

func NewRouter(
	cfg RouterConfig,
	authCtrl *controllers.AuthController,
	walletCtrl *controllers.WalletController,
	depositCtrl *controllers.DepositController,
	conversionCtrl *controllers.ConversionController,
	payoutCtrl *controllers.PayoutController,
	logger *slog.Logger,
) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))

	// Public routes
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/signup", authCtrl.Signup)
		authGroup.POST("/login", authCtrl.Login)
	}

	// Protected routes — all require a valid JWT
	protected := r.Group("/")
	protected.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		protected.GET("/wallets/balances", walletCtrl.GetBalances)
		protected.GET("/wallets/transactions", walletCtrl.GetTransactions)

		protected.POST("/deposits", depositCtrl.Create)

		protected.POST("/conversions/quote", conversionCtrl.CreateQuote)
		protected.POST("/conversions/execute", conversionCtrl.Execute)

		protected.POST("/payouts", payoutCtrl.Create)
		protected.GET("/payouts/:id", payoutCtrl.Get)
	}

	return r
}

type RouterConfig struct {
	JWTSecret string
	Env       string
}
