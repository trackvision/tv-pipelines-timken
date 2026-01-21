package types

import "time"

// Product represents a product record in TiDB
// This is a template example - customize for your use case
type Product struct {
	ID          string     `db:"id" json:"id"`
	GTIN        string     `db:"gtin" json:"gtin"`
	Name        string     `db:"name" json:"name"`
	Description string     `db:"description" json:"description"`
	Brand       string     `db:"brand" json:"brand"`
	Status      string     `db:"status" json:"status"`
	DateCreated time.Time  `db:"date_created" json:"date_created"`
	DateUpdated *time.Time `db:"date_updated" json:"date_updated,omitempty"`
}

// PipelineResult is a generic result type for pipelines
// Customize this for your specific pipeline output
type PipelineResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}
