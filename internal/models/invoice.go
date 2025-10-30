package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Invoice represents the extracted data from a receipt/invoice
type Invoice struct {
	// Basic information
	Vendor string          `json:"vendor"`           // Merchant/store name
	Date   time.Time       `json:"date"`             // Invoice date
	Total  decimal.Decimal `json:"total"`            // Total amount
	Tax    decimal.Decimal `json:"tax,omitempty"`    // Tax amount if available

	// Line items
	Items []InvoiceItem `json:"items,omitempty"` // Individual line items

	// Categories (optional)
	Categories []string `json:"categories,omitempty"` // Suggested categories

	// Raw data
	RawText string `json:"rawText,omitempty"` // Complete OCR text

	// Metadata
	Confidence  float64 `json:"confidence"`  // Overall confidence score (0-1)
	ProcessedAt time.Time `json:"processedAt"` // When it was processed
}

// InvoiceItem represents a line item in an invoice
type InvoiceItem struct {
	Name   string          `json:"name"`             // Item name/description
	Amount decimal.Decimal `json:"amount"`           // Item price
	IsTaxed bool           `json:"isTaxed"`          // Whether tax applies to this item
	Quantity int           `json:"quantity,omitempty"` // Quantity (if detected)
}

// ProcessRequest represents the input for invoice processing
type ProcessRequest struct {
	// Image data (base64 encoded or raw bytes will be sent as multipart)
	ImageData []byte `json:"-"`

	// Configuration (optional)
	UseVisionModel bool   `json:"useVisionModel"` // Use vision AI directly (skip OCR)
	AIProvider     string `json:"aiProvider"`     // "openai", "gemini", "ollama"
	Model          string `json:"model"`          // Specific model name
	Language       string `json:"language"`       // OCR language (default: "eng")
}

// ProcessResponse represents the output of invoice processing
type ProcessResponse struct {
	Success bool     `json:"success"`
	Invoice *Invoice `json:"invoice,omitempty"`
	Error   string   `json:"error,omitempty"`

	// Processing metadata
	OCRDuration float64 `json:"ocrDuration,omitempty"` // OCR time in seconds
	AIDuration  float64 `json:"aiDuration,omitempty"`  // AI extraction time in seconds
	TotalDuration float64 `json:"totalDuration"`       // Total processing time
}

// Config represents the service configuration
type Config struct {
	// Server config
	Port int    `yaml:"port"`
	Host string `yaml:"host"`

	// OCR config
	OCR OCRConfig `yaml:"ocr"`

	// AI config
	AI AIConfig `yaml:"ai"`

	// Categories (for better extraction)
	Categories []string `yaml:"categories"`
}

// OCRConfig represents OCR-specific configuration
type OCRConfig struct {
	Engine   string `yaml:"engine"` // "tesseract" or "easyocr"
	Language string `yaml:"language"` // OCR language (default: "eng")
}

// AIConfig represents AI provider configuration
type AIConfig struct {
	// OpenAI
	OpenAI OpenAIConfig `yaml:"openai"`

	// Gemini
	Gemini GeminiConfig `yaml:"gemini"`

	// Ollama (local)
	Ollama OllamaConfig `yaml:"ollama"`

	// Default provider
	DefaultProvider string `yaml:"default_provider"` // "openai", "gemini", "ollama"
}

// OpenAIConfig for OpenAI/Azure OpenAI
type OpenAIConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url,omitempty"` // For custom endpoints
	Model   string `yaml:"model"`              // Default: "gpt-4"
}

// GeminiConfig for Google Gemini
type GeminiConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"` // Default: "gemini-pro"
}

// OllamaConfig for local Ollama
type OllamaConfig struct {
	BaseURL string `yaml:"base_url"` // Default: "http://localhost:11434"
	Model   string `yaml:"model"`    // e.g., "mistral", "llama2"
}
