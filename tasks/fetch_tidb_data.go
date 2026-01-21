package tasks

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/trackvision/tv-pipelines-template/types"
	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

// FetchProductByGTIN retrieves a product from TiDB using its GTIN
// This is a template example - customize the query and type for your use case
func FetchProductByGTIN(ctx context.Context, db *sqlx.DB, gtin string) (*types.Product, error) {
	logger.Info("Fetching product by GTIN", zap.String("gtin", gtin))

	var product types.Product
	query := `
		SELECT
			id,
			gtin,
			name,
			description,
			brand,
			status,
			date_created,
			date_updated
		FROM product
		WHERE gtin = ?
	`

	err := db.GetContext(ctx, &product, query, gtin)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product not found for GTIN %s", gtin)
		}
		return nil, fmt.Errorf("querying product: %w", err)
	}

	logger.Info("Product found",
		zap.String("gtin", gtin),
		zap.String("id", product.ID),
		zap.String("name", product.Name),
	)

	return &product, nil
}

// FetchProductsByGTINs retrieves multiple products by their GTINs
// This is a template example - customize the query and type for your use case
func FetchProductsByGTINs(ctx context.Context, db *sqlx.DB, gtins []string) ([]types.Product, error) {
	if len(gtins) == 0 {
		return nil, nil
	}

	logger.Info("Fetching products by GTINs", zap.Int("count", len(gtins)))

	query, args, err := sqlx.In(`
		SELECT
			id,
			gtin,
			name,
			description,
			brand,
			status,
			date_created,
			date_updated
		FROM product
		WHERE gtin IN (?)
	`, gtins)
	if err != nil {
		return nil, fmt.Errorf("building IN query: %w", err)
	}

	// Rebind for MySQL/TiDB
	query = db.Rebind(query)

	var products []types.Product
	err = db.SelectContext(ctx, &products, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying products: %w", err)
	}

	logger.Info("Products found", zap.Int("count", len(products)))

	return products, nil
}
