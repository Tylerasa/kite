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

// FXRateAdapter fetches live FX rates from Frankfurter API (free, no key required).
// Falls back to open.er-api.com for currencies not supported by Frankfurter (NGN, KES).
type FXRateAdapter struct {
	client *http.Client
}

func NewFXRateAdapter() *FXRateAdapter {
	return &FXRateAdapter{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// frankfurterCurrencies are the currencies supported by the Frankfurter API.
var frankfurterCurrencies = map[models.Currency]bool{
	models.USD: true,
	models.GBP: true,
	models.EUR: true,
}

func (a *FXRateAdapter) GetRate(ctx context.Context, from, to models.Currency) (decimal.Decimal, error) {
	// If either currency is not on Frankfurter, use open.er-api.com
	if !frankfurterCurrencies[from] || !frankfurterCurrencies[to] {
		return a.getOpenERAPIRate(ctx, from, to)
	}
	return a.getFrankfurterRate(ctx, from, to)
}

func (a *FXRateAdapter) getFrankfurterRate(ctx context.Context, from, to models.Currency) (decimal.Decimal, error) {
	url := fmt.Sprintf("https://api.frankfurter.app/latest?from=%s&to=%s", from, to)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return decimal.Zero, fmt.Errorf("build request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return decimal.Zero, fmt.Errorf("fetch frankfurter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("frankfurter returned %d", resp.StatusCode)
	}

	var result struct {
		Rates map[string]decimal.Decimal `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return decimal.Zero, fmt.Errorf("decode response: %w", err)
	}

	rate, ok := result.Rates[string(to)]
	if !ok {
		return decimal.Zero, fmt.Errorf("rate for %s not found in response", to)
	}

	return rate, nil
}

func (a *FXRateAdapter) getOpenERAPIRate(ctx context.Context, from, to models.Currency) (decimal.Decimal, error) {
	// open.er-api.com — free tier, no API key needed.
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
