package types

import "time"

// Product represents a product record in TiDB
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

// COCItem represents a single item from the COC API response
type COCItem struct {
	SSCC                     string   `json:"sscc"`
	Serial                   string   `json:"serial"`
	ProductID                string   `json:"product_id"`
	COCDocumentID            string   `json:"coc_document_id"`
	COCDocumentDate          string   `json:"coc_document_date"`
	DeliveryNoteURI          string   `json:"delivery_note_uri"`
	PurchaseOrderURI         string   `json:"purchase_order_uri"`
	ShippingEventID          string   `json:"shipping_event_id"`
	SendCOCEmails            int      `json:"send_coc_emails"`
	ShipToNotificationEmails []string `json:"ship_to_notification_emails"`
	SoldToNotificationEmails []string `json:"sold_to_notification_emails"`
}

// COCData contains the collection of items from COC API
type COCData struct {
	Items []COCItem `json:"items"`
	SSCC  string    `json:"sscc"`
}

// PDFData contains the generated PDF bytes and metadata
type PDFData struct {
	PDFBytes    []byte
	PDFFilename string
	SSCC        string
}

// CoveredProduct represents a product covered by the certification
type CoveredProduct struct {
	ProductID string `json:"product_id"`
}

// CertificationRecord represents a Directus certification record
type CertificationRecord struct {
	CertificationType           string           `json:"certification_type"`
	CertificationIdentification string           `json:"certification_identification"`
	SSCC                        string           `json:"sscc"`
	DeliveryNote                string           `json:"delivery_note"`
	CustomerPO                  string           `json:"customer_po"`
	InitialCertificationDate    string           `json:"initial_certification_date"`
	CoveredSerials              string           `json:"covered_serials"`
	CoveredProducts             []CoveredProduct `json:"covered_products"`
	EventID                     string           `json:"event_id"`
}

// PreparedData contains all data prepared for Directus operations
type PreparedData struct {
	Certification  CertificationRecord
	PDFBytes       []byte
	PDFFilename    string
	SendEmail      bool
	EmailAddresses []string
	SSCC           string
}

// CertificationResult contains the result after creating certification in Directus
type CertificationResult struct {
	PreparedData
	CertificationID string
}

// UploadResult contains the result after uploading PDF to Directus
type UploadResult struct {
	CertificationResult
	FileID string
}

// PipelineResult contains the final pipeline result
type PipelineResult struct {
	UploadResult
	EmailSent    bool
	EmailSkipped string
}
