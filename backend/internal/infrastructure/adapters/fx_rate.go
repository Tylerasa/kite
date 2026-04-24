package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kite/internal/domain/models"
	"github.com/shopspring/decimal"
)

// FXRateAdapter fetches live FX rates from open.er-api.com (free, no key required).
type FXRateAdapter struct {
	client *http.Client
}

func NewFXRateAdapter() *FXRateAdapter {
	return &FXRateAdapter{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *FXRateAdapter) GetRate(ctx context.Context, from, to models.Currency) (decimal.Decimal, error) {
	url := fmt.Sprintf("https://open.er-api.com/v6/latest/%s", from)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return decimal.Zero, fmt.Errorf("build request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return decimal.Zero, fmt.Errorf("fetch open.er-api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("open.er-api returned %d", resp.StatusCode)
	}

	var result struct {
		Result string                     `json:"result"`
		Rates  map[string]decimal.Decimal `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return decimal.Zero, fmt.Errorf("decode response: %w", err)
	}

	if result.Result != "success" {
		return decimal.Zero, fmt.Errorf("open.er-api error: %s", result.Result)
	}

	rate, ok := result.Rates[strings.ToUpper(string(to))]
	if !ok {
		return decimal.Zero, fmt.Errorf("rate for %s not found in response", to)
	}

	return rate, nil
}
