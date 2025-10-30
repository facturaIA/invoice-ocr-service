package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/facturaIA/invoice-ocr-service/internal/ai"
	"github.com/facturaIA/invoice-ocr-service/internal/models"
	"github.com/facturaIA/invoice-ocr-service/internal/ocr"
	"github.com/gorilla/mux"
)

const (
	MaxUploadSize = 10 * 1024 * 1024 // 10MB
	Version       = "1.0.0"
)

// Handler handles HTTP requests for invoice processing
type Handler struct {
	config *models.Config
}

// NewHandler creates a new API handler
func NewHandler(config *models.Config) *Handler {
	return &Handler{
		config: config,
	}
}

// SetupRoutes configures the HTTP routes
func (h *Handler) SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Main endpoint
	router.HandleFunc("/api/process-invoice", h.ProcessInvoice).Methods("POST")

	// Health check
	router.HandleFunc("/health", h.Health).Methods("GET")

	return router
}

// HealthResponse represents the health check response structure
type HealthResponse struct {
	Status      string            `json:"status"`
	Version     string            `json:"version"`
	Timestamp   string            `json:"timestamp"`
	Uptime      string            `json:"uptime"`
	Memory      MemoryStats       `json:"memory"`
	Tesseract   ServiceStatus     `json:"tesseract"`
	ImageMagick ServiceStatus     `json:"imageMagick"`
	AI          map[string]string `json:"ai"`
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	Allocated string `json:"allocated"`
	Total     string `json:"total"`
	System    string `json:"system"`
}

// ServiceStatus represents the status of a service dependency
type ServiceStatus struct {
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
	Error     string `json:"error,omitempty"`
}

var startTime = time.Now()

// Health endpoint - enhanced for Railway monitoring
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Memory statistics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check Tesseract
	tesseractStatus := h.checkTesseract()

	// Check ImageMagick
	imageMagickStatus := h.checkImageMagick()

	// Build response
	response := HealthResponse{
		Status:    "healthy",
		Version:   Version,
		Timestamp: time.Now().Format(time.RFC3339),
		Uptime:    time.Since(startTime).String(),
		Memory: MemoryStats{
			Allocated: fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024),
			Total:     fmt.Sprintf("%.2f MB", float64(m.TotalAlloc)/1024/1024),
			System:    fmt.Sprintf("%.2f MB", float64(m.Sys)/1024/1024),
		},
		Tesseract:   tesseractStatus,
		ImageMagick: imageMagickStatus,
		AI: map[string]string{
			"defaultProvider": h.config.AI.DefaultProvider,
			"ocrEngine":       h.config.OCR.Engine,
		},
	}

	// If critical dependencies are down, mark as unhealthy
	if !tesseractStatus.Available || !imageMagickStatus.Available {
		response.Status = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(response)
}

// checkTesseract verifies Tesseract OCR is available
func (h *Handler) checkTesseract() ServiceStatus {
	cmd := exec.Command("tesseract", "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return ServiceStatus{
			Available: false,
			Error:     "tesseract not found or not executable",
		}
	}

	// Parse version from output (first line usually contains version)
	version := "unknown"
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		version = strings.TrimSpace(lines[0])
	}

	return ServiceStatus{
		Available: true,
		Version:   version,
	}
}

// checkImageMagick verifies ImageMagick is available
func (h *Handler) checkImageMagick() ServiceStatus {
	cmd := exec.Command("convert", "-version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return ServiceStatus{
			Available: false,
			Error:     "imagemagick not found or not executable",
		}
	}

	// Parse version from output
	version := "unknown"
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		version = strings.TrimSpace(lines[0])
	}

	return ServiceStatus{
		Available: true,
		Version:   version,
	}
}

// ProcessInvoice handles invoice processing requests
func (h *Handler) ProcessInvoice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	startTime := time.Now()

	// Parse multipart form
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)
	err := r.ParseMultipartForm(MaxUploadSize)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "File too large or invalid form data")
		return
	}

	// Get file
	file, header, err := r.FormFile("file")
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	// Read file bytes
	imageData, err := io.ReadAll(file)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to read file")
		return
	}

	// Get optional parameters
	useVisionModel := r.FormValue("useVisionModel") == "true"
	aiProvider := r.FormValue("aiProvider")
	if aiProvider == "" {
		aiProvider = h.config.AI.DefaultProvider
	}

	model := r.FormValue("model")
	language := r.FormValue("language")
	if language == "" {
		language = h.config.OCR.Language
	}

	// Process invoice
	invoice, ocrDuration, aiDuration, err := h.processInvoice(
		imageData,
		useVisionModel,
		aiProvider,
		model,
		language,
	)

	totalDuration := time.Since(startTime).Seconds()

	if err != nil {
		response := models.ProcessResponse{
			Success:       false,
			Error:         err.Error(),
			TotalDuration: totalDuration,
		}
		w.WriteHeader(http.StatusOK) // Still return 200 with error in body
		json.NewEncoder(w).Encode(response)
		return
	}

	// Success response
	response := models.ProcessResponse{
		Success:       true,
		Invoice:       invoice,
		OCRDuration:   ocrDuration,
		AIDuration:    aiDuration,
		TotalDuration: totalDuration,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// processInvoice performs the actual processing
func (h *Handler) processInvoice(
	imageData []byte,
	useVisionModel bool,
	providerName string,
	modelName string,
	language string,
) (*models.Invoice, float64, float64, error) {
	var ocrText string
	var ocrDuration float64
	var imageBase64 string

	// Step 1: Preprocess image
	preprocessor := ocr.NewPreprocessor(h.config.OCR.Engine == "easyocr")
	processedImage, err := preprocessor.PreprocessImageFromBytes(imageData)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("image preprocessing failed: %w", err)
	}

	// Step 2: OCR or prepare image for vision model
	if useVisionModel {
		// Convert to base64 for vision models
		imageBase64 = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(processedImage)
	} else {
		// Perform OCR
		tesseract := ocr.NewTesseractOCR(language)
		text, duration, err := tesseract.ExtractText(processedImage)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("OCR failed: %w", err)
		}
		ocrText = text
		ocrDuration = duration
	}

	// Step 3: Create AI provider
	provider, err := h.createProvider(providerName, modelName)
	if err != nil {
		return nil, ocrDuration, 0, err
	}

	// Step 4: Extract data with AI
	extractor := ai.NewExtractor(provider, h.config.Categories)
	invoice, aiDuration, err := extractor.Extract(ocrText, imageBase64)
	if err != nil {
		return nil, ocrDuration, 0, fmt.Errorf("AI extraction failed: %w", err)
	}

	return invoice, ocrDuration, aiDuration, nil
}

// createProvider creates the appropriate AI provider
func (h *Handler) createProvider(providerName, modelName string) (ai.Provider, error) {
	switch providerName {
	case "openai":
		model := modelName
		if model == "" {
			model = h.config.AI.OpenAI.Model
		}
		return ai.NewOpenAIProvider(
			h.config.AI.OpenAI.APIKey,
			h.config.AI.OpenAI.BaseURL,
			model,
		), nil

	case "gemini":
		model := modelName
		if model == "" {
			model = h.config.AI.Gemini.Model
		}
		return ai.NewGeminiProvider(
			h.config.AI.Gemini.APIKey,
			model,
		), nil

	case "ollama":
		model := modelName
		if model == "" {
			model = h.config.AI.Ollama.Model
		}
		return ai.NewOllamaProvider(
			h.config.AI.Ollama.BaseURL,
			model,
		), nil

	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", providerName)
	}
}

// sendError sends an error response
func (h *Handler) sendError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
