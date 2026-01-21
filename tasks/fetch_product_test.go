package tasks

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestFetchProductByGTIN_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")

	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"id", "gtin", "name", "description", "brand", "status", "date_created", "date_updated"}).
		AddRow("prod-123", "01234567890123", "Test Product", "A test product", "TestBrand", "published", createdAt, nil)

	mock.ExpectQuery("SELECT .+ FROM product WHERE gtin = ?").
		WithArgs("01234567890123").
		WillReturnRows(rows)

	product, err := FetchProductByGTIN(context.Background(), sqlxDB, "01234567890123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if product.ID != "prod-123" {
		t.Errorf("expected ID 'prod-123', got '%s'", product.ID)
	}
	if product.GTIN != "01234567890123" {
		t.Errorf("expected GTIN '01234567890123', got '%s'", product.GTIN)
	}
	if product.Name != "Test Product" {
		t.Errorf("expected Name 'Test Product', got '%s'", product.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestFetchProductByGTIN_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")

	rows := sqlmock.NewRows([]string{"id", "gtin", "name", "description", "brand", "status", "date_created", "date_updated"})

	mock.ExpectQuery("SELECT .+ FROM product WHERE gtin = ?").
		WithArgs("99999999999999").
		WillReturnRows(rows)

	_, err = FetchProductByGTIN(context.Background(), sqlxDB, "99999999999999")
	if err == nil {
		t.Fatal("expected error for not found product")
	}

	if err.Error() != "product not found for GTIN 99999999999999" {
		t.Errorf("unexpected error message: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestFetchProductsByGTINs_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")

	createdAt1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createdAt2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"id", "gtin", "name", "description", "brand", "status", "date_created", "date_updated"}).
		AddRow("prod-1", "01234567890123", "Product 1", "Desc 1", "Brand1", "published", createdAt1, nil).
		AddRow("prod-2", "01234567890124", "Product 2", "Desc 2", "Brand2", "published", createdAt2, nil)

	mock.ExpectQuery("SELECT .+ FROM product WHERE gtin IN").
		WithArgs("01234567890123", "01234567890124").
		WillReturnRows(rows)

	products, err := FetchProductsByGTINs(context.Background(), sqlxDB, []string{"01234567890123", "01234567890124"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(products) != 2 {
		t.Errorf("expected 2 products, got %d", len(products))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestFetchProductsByGTINs_EmptyList(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")

	products, err := FetchProductsByGTINs(context.Background(), sqlxDB, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if products != nil {
		t.Errorf("expected nil for empty input, got %v", products)
	}
}
