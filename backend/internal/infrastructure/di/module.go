package di

import (
	"log/slog"
	"os"
	"time"

	"go.uber.org/fx"

	"github.com/kite/internal/application/auth"
	"github.com/kite/internal/application/conversions"
	"github.com/kite/internal/application/deposits"
	"github.com/kite/internal/application/payouts"
	"github.com/kite/internal/application/transactions"
	"github.com/kite/internal/application/wallet"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/infrastructure/bootstrap"
	"github.com/kite/internal/infrastructure/repositories"
	"github.com/kite/internal/infrastructure/rest"
	"github.com/kite/internal/infrastructure/rest/controllers"
)

var Module = fx.Options(
	// Config
	fx.Provide(bootstrap.NewConfig),
	fx.Provide(bootstrap.NewDBPool),

	// Logger
	fx.Provide(func(cfg *bootstrap.Config) *slog.Logger {
		if cfg.AppEnv == "production" {
			return slog.New(slog.NewJSONHandler(os.Stdout, nil))
		}
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}),

	// Repositories
	fx.Provide(repositories.NewUserRepo),
	fx.Provide(repositories.NewAccountRepo),
	fx.Provide(repositories.NewLedgerRepo),
	fx.Provide(repositories.NewDepositRepo),
	fx.Provide(repositories.NewFXRateCacheRepo),
	fx.Provide(repositories.NewQuoteRepo),
	fx.Provide(repositories.NewConversionRepo),
	fx.Provide(repositories.NewPayoutRepo),
	fx.Provide(repositories.NewTransactionRepo),

	// FX adapter
	fx.Provide(NewFXRateAdapter),

	// Use cases
	fx.Provide(func(cfg *bootstrap.Config, users *repositories.UserRepo, accounts *repositories.AccountRepo) in.AuthUseCase {
		return auth.NewUseCase(users, accounts, cfg.JWTSecret)
	}),
	fx.Provide(func(accounts *repositories.AccountRepo, ledger *repositories.LedgerRepo) in.WalletUseCase {
		return wallet.NewUseCase(accounts, ledger)
	}),
	fx.Provide(func(d *repositories.DepositRepo, a *repositories.AccountRepo, l *repositories.LedgerRepo) in.DepositUseCase {
		return deposits.NewUseCase(d, a, l)
	}),
	fx.Provide(func(
		cfg *bootstrap.Config,
		q *repositories.QuoteRepo,
		c *repositories.ConversionRepo,
		a *repositories.AccountRepo,
		l *repositories.LedgerRepo,
		cache *repositories.FXRateCacheRepo,
		fx_ *FXRateAdapter,
	) in.ConversionUseCase {
		return conversions.NewUseCase(q, c, a, l, fx_, cache, cfg.FXSpreadPct, time.Duration(cfg.FXCacheTTLMinutes)*time.Minute)
	}),
	fx.Provide(func(p *repositories.PayoutRepo, a *repositories.AccountRepo, l *repositories.LedgerRepo, cfg *bootstrap.Config) in.PayoutUseCase {
		return payouts.NewUseCase(p, a, l, cfg.ComplianceNGNThreshold)
	}),
	fx.Provide(func(txRepo *repositories.TransactionRepo) in.TransactionUseCase {
		return transactions.NewUseCase(txRepo)
	}),

	// Controllers
	fx.Provide(controllers.NewAuthController),
	fx.Provide(controllers.NewWalletController),
	fx.Provide(controllers.NewDepositController),
	fx.Provide(controllers.NewConversionController),
	fx.Provide(controllers.NewPayoutController),

	// Router
	fx.Provide(func(cfg *bootstrap.Config) rest.RouterConfig {
		return rest.RouterConfig{JWTSecret: cfg.JWTSecret, Env: cfg.AppEnv}
	}),
	fx.Provide(rest.NewRouter),
)
