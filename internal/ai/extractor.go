package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/facturaIA/invoice-ocr-service/internal/models"
	"github.com/shopspring/decimal"
)

// Extractor handles AI-based data extraction from OCR text or images
type Extractor struct {
	provider   Provider
	categories []string
}

// NewExtractor creates a new AI extractor
func NewExtractor(provider Provider, categories []string) *Extractor {
	return &Extractor{
		provider:   provider,
		categories: categories,
	}
}

// Extract processes OCR text or image and returns structured invoice data
func (e *Extractor) Extract(ocrText string, imageBase64 string) (*models.Invoice, float64, error) {
	startTime := time.Now()

	// Build prompt
	prompt := e.buildPrompt(ocrText)

	// Call AI provider
	response, err := e.provider.ExtractData(prompt, imageBase64)
	if err != nil {
		return nil, 0, fmt.Errorf("AI extraction failed: %w", err)
	}

	duration := time.Since(startTime).Seconds()

	// Parse JSON response
	invoice, err := e.parseResponse(response, ocrText)
	if err != nil {
		return nil, duration, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return invoice, duration, nil
}

// buildPrompt creates the AI prompt with template variable substitution
// Based on Receipt Wrangler's prompt template
func (e *Extractor) buildPrompt(ocrText string) string {
	categoriesStr := strings.Join(e.categories, ", ")
	currentYear := time.Now().Year()

	prompt := fmt.Sprintf(`Extract invoice/receipt data from the following text and return ONLY valid JSON.

Available categories: %s

Return JSON with this EXACT structure (no markdown, no code blocks):
{
  "vendor": "merchant/store name",
  "date": "YYYY-MM-DD",
  "total": 123.45,
  "tax": 12.34,
  "items": [
    {
      "name": "item name",
      "amount": 10.50,
      "isTaxed": true,
      "quantity": 1
    }
  ],
  "categories": ["category1", "category2"]
}

Rules:
- Use 'Unknown Vendor' if store name cannot be found
- Omit fields if not found with confidence
- Assume year is %d if not specified
- Total and amounts must be numbers (not strings)
- Select up to 2 categories from the provided list
- Extract individual items if visible in the receipt

Receipt text:
%s`, categoriesStr, currentYear, ocrText)

	return prompt
}

// parseResponse converts AI JSON response to Invoice struct
func (e *Extractor) parseResponse(response string, ocrText string) (*models.Invoice, error) {
	// Clean response (remove markdown code blocks if present)
	cleaned := strings.TrimSpace(response)
	cleaned = strings.ReplaceAll(cleaned, "```json", "")
	cleaned = strings.ReplaceAll(cleaned, "```", "")
	cleaned = strings.TrimSpace(cleaned)

	// Parse JSON
	var raw struct {
		Vendor     string          `json:"vendor"`
		Date       string          `json:"date"`
		Total      json.Number     `json:"total"`
		Tax        json.Number     `json:"tax"`
		Categories []string        `json:"categories"`
		Items      []struct {
			Name     string      `json:"name"`
			Amount   json.Number `json:"amount"`
			IsTaxed  bool        `json:"isTaxed"`
			Quantity int         `json:"quantity"`
		} `json:"items"`
	}

	err := json.Unmarshal([]byte(cleaned), &raw)
	if err != nil {
		return nil, fmt.Errorf("JSON parse error: %w\nResponse: %s", err, cleaned)
	}

	// Build invoice
	invoice := &models.Invoice{
		Vendor:      raw.Vendor,
		Categories:  raw.Categories,
		RawText:     ocrText,
		Confidence:  0.85, // Default confidence
		ProcessedAt: time.Now(),
	}

	// Parse date
	if raw.Date != "" {
		date, err := time.Parse("2006-01-02", raw.Date)
		if err != nil {
			// Try alternative formats
			date, err = time.Parse("02/01/2006", raw.Date)
			if err != nil {
				date, err = time.Parse("2006-01-02T15:04:05Z07:00", raw.Date)
			}
		}
		if err == nil {
			invoice.Date = date
		}
	}

	// Parse total
	if raw.Total != "" {
		total, err := decimal.NewFromString(string(raw.Total))
		if err == nil {
			invoice.Total = total
		}
	}

	// Parse tax
	if raw.Tax != "" {
		tax, err := decimal.NewFromString(string(raw.Tax))
		if err == nil {
			invoice.Tax = tax
		}
	}

	// Parse items
	invoice.Items = make([]models.InvoiceItem, len(raw.Items))
	for i, item := range raw.Items {
		amount, _ := decimal.NewFromString(string(item.Amount))
		invoice.Items[i] = models.InvoiceItem{
			Name:     item.Name,
			Amount:   amount,
			IsTaxed:  item.IsTaxed,
			Quantity: item.Quantity,
		}
	}

	return invoice, nil
}
