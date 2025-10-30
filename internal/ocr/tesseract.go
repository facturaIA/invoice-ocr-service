package ocr

import (
	"fmt"
	"time"

	"github.com/otiai10/gosseract/v2"
)

// TesseractOCR implements OCR using Tesseract engine
type TesseractOCR struct {
	language string
}

// NewTesseractOCR creates a new Tesseract OCR instance
func NewTesseractOCR(language string) *TesseractOCR {
	if language == "" {
		language = "eng" // Default to English
	}
	return &TesseractOCR{
		language: language,
	}
}

// ExtractText performs OCR on preprocessed image bytes
// Based on Receipt Wrangler's ReadImageWithTesseract function
func (t *TesseractOCR) ExtractText(imageBytes []byte) (string, float64, error) {
	startTime := time.Now()

	// Create Tesseract client
	client := gosseract.NewClient()
	defer client.Close()

	// Set language
	err := client.SetLanguage(t.language)
	if err != nil {
		return "", 0, fmt.Errorf("failed to set language: %w", err)
	}

	// Blacklist special characters that rarely appear in invoices
	// This improves accuracy by preventing OCR from hallucinating special chars
	blacklist := "!@#$%^&*()_+=-[]}{;:'\"\\|~`<>/?"
	err = client.SetVariable("tessedit_char_blacklist", blacklist)
	if err != nil {
		// Non-fatal error, continue
		fmt.Printf("Warning: failed to set character blacklist: %v\n", err)
	}

	// Set image from bytes
	err = client.SetImageFromBytes(imageBytes)
	if err != nil {
		return "", 0, fmt.Errorf("failed to set image: %w", err)
	}

	// Extract text
	text, err := client.Text()
	if err != nil {
		return "", 0, fmt.Errorf("OCR extraction failed: %w", err)
	}

	duration := time.Since(startTime).Seconds()

	// Calculate confidence (0-1 scale)
	confidence, err := t.calculateConfidence(client)
	if err != nil {
		// Non-fatal, use default confidence
		confidence = 0.8
	}

	return text, duration, nil
}

// calculateConfidence gets mean confidence from Tesseract
func (t *TesseractOCR) calculateConfidence(client *gosseract.Client) (float64, error) {
	// Get confidence (0-100 scale)
	conf, err := client.GetConfidence()
	if err != nil {
		return 0, err
	}

	// Convert to 0-1 scale
	return float64(conf) / 100.0, nil
}

// ExtractTextWithDetails returns text and detailed word information
func (t *TesseractOCR) ExtractTextWithDetails(imageBytes []byte) (string, []WordInfo, error) {
	client := gosseract.NewClient()
	defer client.Close()

	err := client.SetLanguage(t.language)
	if err != nil {
		return "", nil, fmt.Errorf("failed to set language: %w", err)
	}

	err = client.SetImageFromBytes(imageBytes)
	if err != nil {
		return "", nil, fmt.Errorf("failed to set image: %w", err)
	}

	// Get text
	text, err := client.Text()
	if err != nil {
		return "", nil, fmt.Errorf("OCR extraction failed: %w", err)
	}

	// Get bounding boxes (for advanced use cases)
	boxes, err := client.GetBoundingBoxes(gosseract.RIL_WORD)
	if err != nil {
		// Return text without boxes
		return text, nil, nil
	}

	// Convert to WordInfo
	words := make([]WordInfo, len(boxes))
	for i, box := range boxes {
		words[i] = WordInfo{
			Text:       box.Word,
			Confidence: float64(box.Confidence) / 100.0,
			Box: BoundingBox{
				X:      box.Box.Min.X,
				Y:      box.Box.Min.Y,
				Width:  box.Box.Max.X - box.Box.Min.X,
				Height: box.Box.Max.Y - box.Box.Min.Y,
			},
		}
	}

	return text, words, nil
}

// WordInfo contains detailed information about a detected word
type WordInfo struct {
	Text       string
	Confidence float64
	Box        BoundingBox
}

// BoundingBox represents the location of text in the image
type BoundingBox struct {
	X      int
	Y      int
	Width  int
	Height int
}
