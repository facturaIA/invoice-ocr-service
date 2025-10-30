package ocr

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/gographics/imagick.v3/imagick"
)

// Preprocessor handles image preprocessing for optimal OCR results
type Preprocessor struct {
	scaleForEasyOCR bool
}

// NewPreprocessor creates a new image preprocessor
func NewPreprocessor(scaleForEasyOCR bool) *Preprocessor {
	return &Preprocessor{
		scaleForEasyOCR: scaleForEasyOCR,
	}
}

// PreprocessImage applies ImageMagick operations to optimize image for OCR
// Based on Receipt Wrangler's prepareImage() function
func (p *Preprocessor) PreprocessImage(imagePath string) ([]byte, error) {
	// Initialize ImageMagick
	imagick.Initialize()
	defer imagick.Terminate()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	// Read image
	err := mw.ReadImage(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	// Step 1: Trim borders/whitespace
	err = mw.TrimImage(0)
	if err != nil {
		return nil, fmt.Errorf("trim failed: %w", err)
	}

	// Step 2: Convert to bilevel (pure black and white)
	// This improves OCR accuracy by removing gray areas
	err = mw.SetImageType(imagick.IMAGE_TYPE_BILEVEL)
	if err != nil {
		return nil, fmt.Errorf("bilevel conversion failed: %w", err)
	}

	// Step 3: Apply blur to reduce noise
	// Radius: 0 (auto), Sigma: 1.5
	err = mw.BlurImage(0, 1.5)
	if err != nil {
		return nil, fmt.Errorf("blur failed: %w", err)
	}

	// Step 4: Sharpen edges
	// Radius: 0 (auto), Sigma: 1
	err = mw.SharpenImage(0, 1)
	if err != nil {
		return nil, fmt.Errorf("sharpen failed: %w", err)
	}

	// Step 5: Enhance image (improve contrast and detail)
	err = mw.EnhanceImage()
	if err != nil {
		return nil, fmt.Errorf("enhance failed: %w", err)
	}

	// Step 6: Reduce contrast
	// false = reduce (not increase)
	err = mw.ContrastImage(false)
	if err != nil {
		return nil, fmt.Errorf("contrast reduction failed: %w", err)
	}

	// Step 7: Deskew (straighten tilted images)
	// Threshold: 0.40 (40%)
	err = mw.DeskewImage(0.40)
	if err != nil {
		return nil, fmt.Errorf("deskew failed: %w", err)
	}

	// Step 8: Scale down for EasyOCR (optional)
	// EasyOCR performs better with smaller images
	if p.scaleForEasyOCR {
		width := mw.GetImageWidth()
		height := mw.GetImageHeight()
		err = mw.ScaleImage(width/2, height/2)
		if err != nil {
			return nil, fmt.Errorf("scale failed: %w", err)
		}
	}

	// Get processed image as bytes
	blob := mw.GetImageBlob()
	if len(blob) == 0 {
		return nil, fmt.Errorf("processed image is empty")
	}

	return blob, nil
}

// PreprocessImageFromBytes processes image from byte slice
func (p *Preprocessor) PreprocessImageFromBytes(imageData []byte) ([]byte, error) {
	// Write to temp file
	tempFile, err := os.CreateTemp("", "invoice-*.jpg")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write(imageData)
	if err != nil {
		tempFile.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tempFile.Close()

	// Process from file
	return p.PreprocessImage(tempFile.Name())
}

// SaveProcessedImage saves preprocessed image to file (for debugging)
func (p *Preprocessor) SaveProcessedImage(imageBytes []byte, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, io.NopCloser(io.Reader(os.Stdin)))
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	_, err = file.Write(imageBytes)
	return err
}
