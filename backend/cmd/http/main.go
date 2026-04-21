package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/kite/internal/infrastructure/bootstrap"
	"github.com/kite/internal/infrastructure/di"
	"github.com/kite/internal/infrastructure/jobs"
	"github.com/kite/internal/infrastructure/repositories"
	inports "github.com/kite/internal/domain/ports/in"
)

func main() {
	fx.New(
		di.Module,

		// Payout background processor
		fx.Provide(jobs.NewPayoutProcessor),

		// Server lifecycle
		fx.Invoke(registerLifecycle),
	).Run()
}

func registerLifecycle(
	lc fx.Lifecycle,
	r *gin.Engine,
	cfg *bootstrap.Config,
	processor *jobs.PayoutProcessor,
	payoutRepo *repositories.PayoutRepo,
	payoutUC inports.PayoutUseCase,
	logger *slog.Logger,
) {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: r,
	}

	processorCtx, cancelProcessor := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("starting kite server", "port", cfg.Port, "env", cfg.AppEnv)

			// Start the payout background job.
			go processor.Start(processorCtx)

			// Start the HTTP server.
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("server error", "error", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			cancelProcessor()
			return server.Shutdown(ctx)
		},
	})
}
