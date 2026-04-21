package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kite/internal/domain/models"
)

type FXRateCacheRepo struct {
	db *pgxpool.Pool
}

func NewFXRateCacheRepo(db *pgxpool.Pool) *FXRateCacheRepo {
	return &FXRateCacheRepo{db: db}
}

func (r *FXRateCacheRepo) Get(ctx context.Context, base, target models.Currency) (*models.FXRateCache, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, base_currency, target_currency, rate, fetched_at, expires_at
		 FROM fx_rate_cache WHERE base_currency = $1 AND target_currency = $2 AND expires_at > NOW()`,
		string(base), string(target),
	)
	var entry models.FXRateCache
	var base2, target2, rate string
	err := row.Scan(&entry.ID, &base2, &target2, &rate, &entry.FetchedAt, &entry.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	entry.BaseCurrency = models.Currency(base2)
	entry.TargetCurrency = models.Currency(target2)
	entry.Rate = rate
	return &entry, nil
}

func (r *FXRateCacheRepo) Upsert(ctx context.Context, entry *models.FXRateCache) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO fx_rate_cache (id, base_currency, target_currency, rate, fetched_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (base_currency, target_currency)
		 DO UPDATE SET rate = EXCLUDED.rate, fetched_at = EXCLUDED.fetched_at, expires_at = EXCLUDED.expires_at`,
		entry.ID, string(entry.BaseCurrency), string(entry.TargetCurrency),
		entry.Rate, entry.FetchedAt, entry.ExpiresAt,
	)
	return err
}

// QuoteRepo manages FX quotes.
type QuoteRepo struct {
	db *pgxpool.Pool
}

func NewQuoteRepo(db *pgxpool.Pool) *QuoteRepo {
	return &QuoteRepo{db: db}
}

func (r *QuoteRepo) Create(ctx context.Context, quote *models.FXQuote) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO fx_quotes
		 (id, user_id, from_currency, to_currency, market_rate, quoted_rate, spread_pct,
		  amount_in, amount_out, fee, expires_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		quote.ID, quote.UserID, string(quote.FromCurrency), string(quote.ToCurrency),
		quote.MarketRate, quote.QuotedRate, quote.SpreadPct,
		quote.AmountIn, quote.AmountOut, quote.Fee, quote.ExpiresAt, quote.CreatedAt,
	)
	return err
}

func (r *QuoteRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.FXQuote, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, user_id, from_currency, to_currency, market_rate, quoted_rate, spread_pct,
		        amount_in, amount_out, fee, expires_at, executed_at, created_at
		 FROM fx_quotes WHERE id = $1`,
		id,
	)
	return scanQuote(row)
}

func (r *QuoteRepo) MarkExecuted(ctx context.Context, id uuid.UUID) (bool, error) {
	tag, err := r.db.Exec(ctx,
		`UPDATE fx_quotes SET executed_at = NOW()
		 WHERE id = $1 AND executed_at IS NULL`,
		id,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func scanQuote(row pgx.Row) (*models.FXQuote, error) {
	var q models.FXQuote
	var from, to, marketRate, quotedRate, spreadPct string
	err := row.Scan(
		&q.ID, &q.UserID, &from, &to, &marketRate, &quotedRate, &spreadPct,
		&q.AmountIn, &q.AmountOut, &q.Fee, &q.ExpiresAt, &q.ExecutedAt, &q.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	q.FromCurrency = models.Currency(from)
	q.ToCurrency = models.Currency(to)
	q.MarketRate = marketRate
	q.QuotedRate = quotedRate
	q.SpreadPct = spreadPct
	return &q, nil
}

// ConversionRepo persists executed conversions.
type ConversionRepo struct {
	db *pgxpool.Pool
}

func NewConversionRepo(db *pgxpool.Pool) *ConversionRepo {
	return &ConversionRepo{db: db}
}

func (r *ConversionRepo) Create(ctx context.Context, c *models.Conversion) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO conversions (id, user_id, quote_id, from_currency, to_currency,
		  amount_in, amount_out, quoted_rate, fee, status, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		c.ID, c.UserID, c.QuoteID, string(c.FromCurrency), string(c.ToCurrency),
		c.AmountIn, c.AmountOut, c.QuotedRate, c.Fee, c.Status, c.CreatedAt,
	)
	return err
}

func (r *ConversionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Conversion, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, user_id, quote_id, from_currency, to_currency,
		        amount_in, amount_out, quoted_rate, fee, status, created_at
		 FROM conversions WHERE id = $1`,
		id,
	)
	var c models.Conversion
	var from, to, rate string
	err := row.Scan(&c.ID, &c.UserID, &c.QuoteID, &from, &to,
		&c.AmountIn, &c.AmountOut, &rate, &c.Fee, &c.Status, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	c.FromCurrency = models.Currency(from)
	c.ToCurrency = models.Currency(to)
	c.QuotedRate = rate
	return &c, nil
}

// Ensure unused import is satisfied
var _ = time.Now
