package di

import (
	"github.com/kite/internal/infrastructure/adapters"
)

// FXRateAdapter wraps the adapter so it can be provided via DI.
type FXRateAdapter = adapters.FXRateAdapter

func NewFXRateAdapter() *FXRateAdapter {
	return adapters.NewFXRateAdapter()
}
