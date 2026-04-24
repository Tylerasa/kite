package rest

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kite/internal/infrastructure/rest/controllers"
	"github.com/kite/internal/infrastructure/rest/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(
	cfg RouterConfig,
	authCtrl *controllers.AuthController,
	walletCtrl *controllers.WalletController,
	depositCtrl *controllers.DepositController,
	conversionCtrl *controllers.ConversionController,
	payoutCtrl *controllers.PayoutController,
	institutionCtrl *controllers.InstitutionController,
	logger *slog.Logger,
) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 10 requests per minute per IP on auth endpoints.
	authLimiter := middleware.NewIPRateLimiter(10, time.Minute)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders(cfg.AllowedOrigin))
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))

	// Swagger UI — available at /swagger/index.html
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Public routes — rate-limited to prevent brute-force on login/signup
	authGroup := r.Group("/auth")
	authGroup.Use(authLimiter.RateLimit())
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
		protected.GET("/wallets/transactions/:id", walletCtrl.GetTransaction)

		protected.POST("/deposits", depositCtrl.Create)

		protected.POST("/conversions/quote", conversionCtrl.CreateQuote)
		protected.POST("/conversions/execute", conversionCtrl.Execute)

		protected.GET("/institutions", institutionCtrl.List)
		protected.POST("/payouts/inquiry", institutionCtrl.Inquiry)
		protected.POST("/payouts", payoutCtrl.Create)
		protected.GET("/payouts/:id", payoutCtrl.Get)
	}

	return r
}

type RouterConfig struct {
	JWTSecret     string
	Env           string
	AllowedOrigin string
}
