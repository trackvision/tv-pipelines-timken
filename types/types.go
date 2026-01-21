package types

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

// COCData represents the full response from the COC API
type COCData struct {
	Items []COCItem `json:"data"`
}

// CoveredProduct represents a product in the covered_products array
type CoveredProduct struct {
	ProductID string `json:"product_id"`
}

// CertificationRecord represents the Directus certification record
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

// DirectusResponse wraps a Directus API response
type DirectusResponse[T any] struct {
	Data T `json:"data"`
}

// DirectusFileResponse represents a file upload response from Directus
type DirectusFileResponse struct {
	ID string `json:"id"`
}

// PipelineRequest represents the incoming HTTP request
type PipelineRequest struct {
	SSCC string `json:"sscc"`
}

// PipelineResult holds the outcome of a pipeline execution
type PipelineResult struct {
	Success         bool
	CertificationID string
	FileID          string
	EmailSent       bool
	Error           string
}

// PipelineResponse represents the HTTP response
type PipelineResponse struct {
	Success         bool   `json:"success"`
	CertificationID string `json:"certification_id,omitempty"`
	FileID          string `json:"file_id,omitempty"`
	EmailSent       bool   `json:"email_sent"`
	Error           string `json:"error,omitempty"`
}
