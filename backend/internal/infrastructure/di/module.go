package di

import (
	"log/slog"
	"os"
	"time"

	slogmulti "github.com/samber/slog-multi"
	"go.uber.org/fx"
	"gopkg.in/lumberjack.v2"

	"github.com/kite/internal/application/auth"
	"github.com/kite/internal/application/conversions"
	"github.com/kite/internal/application/deposits"
	"github.com/kite/internal/application/inquiry"
	"github.com/kite/internal/application/payouts"
	"github.com/kite/internal/application/transactions"
	"github.com/kite/internal/application/wallet"
	portin "github.com/kite/internal/domain/ports/in"
	portout "github.com/kite/internal/domain/ports/out"
	"github.com/kite/internal/domain/services"
	"github.com/kite/internal/infrastructure/bootstrap"
	"github.com/kite/internal/infrastructure/jobs"
	"github.com/kite/internal/infrastructure/repositories"
	"github.com/kite/internal/infrastructure/rest"
	"github.com/kite/internal/infrastructure/rest/controllers"
	"github.com/kite/internal/infrastructure/store"
)

var Module = fx.Options(
	// Config
	fx.Provide(bootstrap.NewConfig),
	fx.Provide(bootstrap.NewDBPool),

	// Logger — production fans to rotating file + stdout; dev uses text handler
	fx.Provide(func(cfg *bootstrap.Config) *slog.Logger {
		if cfg.AppEnv == "production" {
			_ = os.MkdirAll("log", 0o755)
			rotator := &lumberjack.Logger{
				Filename:   cfg.LogFile,
				MaxSize:    100, // MB
				MaxBackups: 3,
				MaxAge:     28, // days
				Compress:   true,
			}
			return slog.New(slogmulti.Fanout(
				slog.NewJSONHandler(rotator, nil),
				slog.NewJSONHandler(os.Stdout, nil),
			))
		}
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}),

	// Repositories — each provided as its domain interface so no lambdas are needed downstream.
	fx.Provide(fx.Annotate(repositories.NewUserRepo,         fx.As(new(portout.UserRepository)))),
	fx.Provide(fx.Annotate(repositories.NewAccountRepo,      fx.As(new(portout.AccountRepository)))),
	fx.Provide(fx.Annotate(repositories.NewLedgerRepo,       fx.As(new(portout.LedgerRepository)))),
	fx.Provide(fx.Annotate(repositories.NewDepositRepo,      fx.As(new(portout.DepositRepository)))),
	fx.Provide(fx.Annotate(repositories.NewFXRateCacheRepo,  fx.As(new(portout.FXRateCacheRepository)))),
	fx.Provide(fx.Annotate(repositories.NewQuoteRepo,        fx.As(new(portout.QuoteRepository)))),
	fx.Provide(fx.Annotate(repositories.NewConversionRepo,   fx.As(new(portout.ConversionRepository)))),
	fx.Provide(fx.Annotate(repositories.NewPayoutRepo,       fx.As(new(portout.PayoutRepository)))),
	fx.Provide(fx.Annotate(repositories.NewTransactionRepo,  fx.As(new(portout.TransactionRepository)))),
	fx.Provide(fx.Annotate(repositories.NewAuditLogRepo,     fx.As(new(portout.AuditLogRepository)))),

	// FX rate adapter + cache
	fx.Provide(NewFXRateAdapter),

	// Transaction guard — singleton shared by all request handlers
	fx.Provide(services.NewTransactionGuard),

	// Institution store + dummy recipient provider (in-memory, no DB)
	fx.Provide(fx.Annotate(store.NewInstitutionStore, fx.As(new(portout.InstitutionStore)))),
	fx.Provide(func() portout.RecipientInquiryProvider { return &store.DummyRecipientProvider{} }),

	// Use cases — annotated directly where all parameters are already interfaces in the container.
	// Lambdas kept only where config scalar fields must be extracted.
	fx.Provide(func(cfg *bootstrap.Config, users portout.UserRepository, accounts portout.AccountRepository) portin.AuthUseCase {
		return auth.NewUseCase(users, accounts, cfg.JWTSecret)
	}),
	fx.Provide(fx.Annotate(wallet.NewUseCase,       fx.As(new(portin.WalletUseCase)))),
	fx.Provide(fx.Annotate(deposits.NewUseCase,     fx.As(new(portin.DepositUseCase)))),
	fx.Provide(func(
		cfg   *bootstrap.Config,
		q     portout.QuoteRepository,
		c     portout.ConversionRepository,
		a     portout.AccountRepository,
		l     portout.LedgerRepository,
		cache portout.FXRateCacheRepository,
		fx_   *FXRateAdapter,
		al    portout.AuditLogRepository,
	) portin.ConversionUseCase {
		return conversions.NewUseCase(q, c, a, l, fx_, cache, al, conversions.Config{SpreadPct: cfg.FXSpreadPct, CacheTTL: time.Duration(cfg.FXCacheTTLMinutes) * time.Minute})
	}),
	fx.Provide(func(
		p     portout.PayoutRepository,
		a     portout.AccountRepository,
		l     portout.LedgerRepository,
		cfg   *bootstrap.Config,
		guard *services.TransactionGuard,
		al    portout.AuditLogRepository,
	) portin.PayoutUseCase {
		return payouts.NewUseCase(p, a, l, cfg.ComplianceNGNThreshold, guard, al)
	}),
	fx.Provide(fx.Annotate(transactions.NewUseCase, fx.As(new(portin.TransactionUseCase)))),
	fx.Provide(fx.Annotate(inquiry.NewUseCase,      fx.As(new(portin.InstitutionUseCase)))),

	// Background jobs
	fx.Provide(func(uc portin.PayoutUseCase, repo portout.PayoutRepository, logger *slog.Logger, cfg *bootstrap.Config) *jobs.PayoutProcessor {
		return jobs.NewPayoutProcessor(uc, repo, logger, cfg.PayoutMaxConcurrency)
	}),

	// Controllers
	fx.Provide(controllers.NewAuthController),
	fx.Provide(controllers.NewWalletController),
	fx.Provide(controllers.NewDepositController),
	fx.Provide(controllers.NewConversionController),
	fx.Provide(controllers.NewPayoutController),
	fx.Provide(controllers.NewInstitutionController),

	// Router
	fx.Provide(func(cfg *bootstrap.Config) rest.RouterConfig {
		return rest.RouterConfig{JWTSecret: cfg.JWTSecret, Env: cfg.AppEnv, AllowedOrigin: cfg.AllowedOrigin}
	}),
	fx.Provide(rest.NewRouter),
)
