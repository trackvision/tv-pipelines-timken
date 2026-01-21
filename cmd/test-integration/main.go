package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/trackvision/tv-pipelines-template/tasks"

	_ "github.com/go-sql-driver/mysql"
	"github.com/trackvision/tv-shared-go/logger"
	"go.uber.org/zap"
)

func main() {
	logger.Trace()

	// Example flags - customize for your use case
	gtin := flag.String("gtin", "01234567890123", "GTIN to look up")
	directusURL := flag.String("directus-url", "", "Directus API URL")
	query := flag.String("query", "test", "Query parameter for Directus")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	logger.Info("Starting integration test")

	// Example 1: Fetch from Directus API
	if *directusURL != "" {
		apiKey := os.Getenv("DIRECTUS_API_KEY")
		if apiKey == "" {
			logger.Warn("DIRECTUS_API_KEY not set, skipping Directus test")
		} else {
			logger.Info("Testing Directus fetch...", zap.String("url", *directusURL))
			data, err := tasks.FetchDirectusData(ctx, nil, *directusURL, apiKey, *query)
			if err != nil {
				logger.Error("Failed to fetch Directus data", zap.Error(err))
			} else {
				logger.Info("Directus data fetched",
					zap.Int("items", len(data.Items)),
					zap.String("query", data.Query),
				)
			}
		}
	}

	// Example 2: Fetch from TiDB
	dbDSN := os.Getenv("DATABASE_DSN")
	if dbDSN == "" {
		logger.Info("DATABASE_DSN not set, skipping TiDB test")
		logger.Info("Set DATABASE_DSN=user:pass@tcp(host:port)/database to test")
	} else {
		logger.Info("Testing TiDB fetch...", zap.String("gtin", *gtin))

		db, err := sqlx.Connect("mysql", dbDSN)
		if err != nil {
			logger.Error("Failed to connect to database", zap.Error(err))
		} else {
			defer func() {
				if cerr := db.Close(); cerr != nil {
					logger.Error("Failed to close database", zap.Error(cerr))
				}
			}()

			product, err := tasks.FetchProductByGTIN(ctx, db, *gtin)
			if err != nil {
				logger.Error("Failed to fetch product", zap.Error(err))
			} else {
				fmt.Println("\n========== PRODUCT ==========")
				fmt.Printf("ID: %s\n", product.ID)
				fmt.Printf("GTIN: %s\n", product.GTIN)
				fmt.Printf("Name: %s\n", product.Name)
				fmt.Printf("Brand: %s\n", product.Brand)
				fmt.Printf("Status: %s\n", product.Status)
				fmt.Println("==============================")
			}
		}
	}

	logger.Info("Integration test complete")
}
