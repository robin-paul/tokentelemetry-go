package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// GetPricingOverrides retrieves all user-defined custom pricing overrides.
func (d *DB) GetPricingOverrides(ctx context.Context) ([]models.PricingOverride, error) {
	query := `
	SELECT model_pattern, input_cost_per_m, output_cost_per_m, cache_read_cost_per_m, cache_write_cost_per_m, source, updated_at
	FROM pricing_overrides
	ORDER BY updated_at DESC;
	`
	rows, err := d.readerDB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pricing overrides: %w", err)
	}
	defer rows.Close()

	var overrides []models.PricingOverride
	for rows.Next() {
		var o models.PricingOverride
		if err := rows.Scan(
			&o.ModelPattern, &o.InputCostPerM, &o.OutputCostPerM,
			&o.CacheReadCostPerM, &o.CacheWriteCostPerM,
			&o.Source, &o.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pricing override: %w", err)
		}
		overrides = append(overrides, o)
	}
	return overrides, rows.Err()
}

// GetPricingOverride retrieves a single pricing override by exact model pattern.
func (d *DB) GetPricingOverride(ctx context.Context, modelPattern string) (*models.PricingOverride, error) {
	query := `
	SELECT model_pattern, input_cost_per_m, output_cost_per_m, cache_read_cost_per_m, cache_write_cost_per_m, source, updated_at
	FROM pricing_overrides
	WHERE model_pattern = ?;
	`
	var o models.PricingOverride
	err := d.readerDB.QueryRowContext(ctx, query, modelPattern).Scan(
		&o.ModelPattern, &o.InputCostPerM, &o.OutputCostPerM,
		&o.CacheReadCostPerM, &o.CacheWriteCostPerM,
		&o.Source, &o.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing override: %w", err)
	}
	return &o, nil
}

// UpsertPricingOverride inserts or updates a model rate override.
func (d *DB) UpsertPricingOverride(ctx context.Context, o *models.PricingOverride) error {
	query := `
	INSERT INTO pricing_overrides (
		model_pattern, input_cost_per_m, output_cost_per_m,
		cache_read_cost_per_m, cache_write_cost_per_m,
		source, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(model_pattern) DO UPDATE SET
		input_cost_per_m = excluded.input_cost_per_m,
		output_cost_per_m = excluded.output_cost_per_m,
		cache_read_cost_per_m = excluded.cache_read_cost_per_m,
		cache_write_cost_per_m = excluded.cache_write_cost_per_m,
		source = excluded.source,
		updated_at = excluded.updated_at;
	`
	now := time.Now().UTC()
	if o.UpdatedAt.IsZero() {
		o.UpdatedAt = now
	}
	if o.Source == "" {
		o.Source = "user_override"
	}
	return d.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			o.ModelPattern, o.InputCostPerM, o.OutputCostPerM,
			o.CacheReadCostPerM, o.CacheWriteCostPerM,
			o.Source, o.UpdatedAt,
		)
		return err
	})
}

// DeletePricingOverride removes a custom pricing override.
func (d *DB) DeletePricingOverride(ctx context.Context, modelPattern string) error {
	return d.WithTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `DELETE FROM pricing_overrides WHERE model_pattern = ?;`, modelPattern)
		if err != nil {
			return fmt.Errorf("failed to delete override: %w", err)
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return ErrNotFound
		}
		return nil
	})
}
