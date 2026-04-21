package out

import (
	"context"

	"github.com/kite/internal/domain/models"
	"github.com/shopspring/decimal"
)

// FXRateProvider fetches live exchange rates.
type FXRateProvider interface {
	GetRate(ctx context.Context, from, to models.Currency) (decimal.Decimal, error)
}
