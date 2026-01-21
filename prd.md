# Timken COC Pipeline

Generate Certificate of Conformance PDFs for shipments and email them to stakeholders.

---

## Endpoint

`POST /run/coc` with body `{"sscc": "..."}`

---

## Steps

### 1. Generate PDF

Call the COC viewer webpage and save as A4 PDF:
```
{COC_VIEWER_BASE_URL}?sscc={sscc}
```
Current URL: `https://timken-coc-viewer.netlify.app/html/sscc-coc/?sscc={sscc}`

If error, throw monitoring exception and halt.

### 2. Fetch COC Data

Call the COC data API:
```
{TIMKEN_COC_API_URL}?sscc={sscc}
```
Current URL: `https://timkendev.trackvision.ai/flows/trigger/705d83de-7f24-4c84-be1c-39ce49cf1677?sscc={sscc}`

If error or no rows returned, throw monitoring exception and halt.

### 3. Create Certification Record

Create a record in Directus `certification` collection with these mappings:

| Field | Value |
|-------|-------|
| `certification_type` | Hard code to `"Conformance"` |
| `certification_identification` | `coc_document_id` from first row |
| `sscc` | `sscc` from first row |
| `delivery_note` | Last path segment of `delivery_note_uri` (e.g., `https://desadv.sap.timken.com/bt/ASN123` → `ASN123`) |
| `customer_po` | Last path segment of `purchase_order_uri` (e.g., `https://po.sap.timken.com/bt/PO123` → `PO123`) |
| `initial_certification_date` | `coc_document_date` from first row |
| `covered_serials` | All `serial` values from all rows, joined with newlines |
| `covered_products` | `[{"product_id": "{product_id from first row}"}]` |
| `event_id` | `shipping_event_id` from first row |

Remember the certification ID.

### 4. Upload PDF

Upload the PDF to Directus and set it as `primary_attachment` on the certification record.

### 5. Send Email (conditional)

Check `send_coc_emails` in first row of API response. If set to `1`:

1. Collect email addresses from `ship_to_notification_emails` and `sold_to_notification_emails` (both arrays)
2. If NO addresses or ANY invalid address, throw monitoring exception and halt
3. Send email to all addresses with PDF attached:

**Subject:** Timken Certificate of Conformance

**Body:**
```
Please find the attached certificate of conformance for your Timken products.

Kind regards,
Timken support team.
```

---

## Response

```json
{
  "success": true,
  "certification_id": "uuid",
  "file_id": "uuid",
  "email_sent": true
}
```
