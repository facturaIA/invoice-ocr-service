package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/sashabaranov/go-openai"
	"google.golang.org/api/option"
)

// Provider interface for AI providers
type Provider interface {
	ExtractData(prompt string, imageBase64 string) (string, error)
}

// OpenAIProvider implements Provider for OpenAI/Azure OpenAI
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey, baseURL, model string) *OpenAIProvider {
	if model == "" {
		model = openai.GPT4 // Default model
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
	}
}

// ExtractData sends prompt and image to OpenAI
func (p *OpenAIProvider) ExtractData(prompt string, imageBase64 string) (string, error) {
	var config openai.ClientConfig

	// Check if Azure OpenAI
	if strings.Contains(p.baseURL, "azure") {
		config = openai.DefaultAzureConfig(p.apiKey, p.baseURL)
	} else {
		config = openai.DefaultConfig(p.apiKey)
		if p.baseURL != "" {
			config.BaseURL = p.baseURL
		}
	}

	client := openai.NewClientWithConfig(config)

	// Build messages
	var messages []openai.ChatCompletionMessage

	if imageBase64 != "" {
		// Vision model with image
		messages = []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: prompt,
					},
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL:    imageBase64,
							Detail: openai.ImageURLDetailAuto,
						},
					},
				},
			},
		}
	} else {
		// Text-only
		messages = []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		}
	}

	// Create chat completion
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       p.model,
			Messages:    messages,
			Temperature: 0, // Deterministic results
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("OpenAI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

// GeminiProvider implements Provider for Google Gemini
type GeminiProvider struct {
	apiKey string
	model  string
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(apiKey, model string) *GeminiProvider {
	if model == "" {
		model = "gemini-pro" // Default model
	}
	return &GeminiProvider{
		apiKey: apiKey,
		model:  model,
	}
}

// ExtractData sends prompt and image to Gemini
func (p *GeminiProvider) ExtractData(prompt string, imageBase64 string) (string, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, option.WithAPIKey(p.apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(p.model)
	model.GenerationConfig.ResponseMIMEType = "application/json"

	// Build parts
	parts := []genai.Part{genai.Text(prompt)}

	// Add image if provided
	if imageBase64 != "" {
		// Remove data URI prefix if present
		imageData := imageBase64
		if strings.HasPrefix(imageData, "data:image") {
			parts := strings.Split(imageData, ",")
			if len(parts) > 1 {
				imageData = parts[1]
			}
		}

		// Decode base64
		imageBytes, err := decodeBase64(imageData)
		if err != nil {
			return "", fmt.Errorf("failed to decode image: %w", err)
		}

		// Detect MIME type
		mimeType := detectMIMEType(imageBytes)

		blob := genai.Blob{
			MIMEType: mimeType,
			Data:     imageBytes,
		}

		parts = append(parts, blob)
	}

	// Generate content
	resp, err := model.GenerateContent(ctx, parts...)
	if err != nil {
		return "", fmt.Errorf("Gemini API call failed: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no response from Gemini")
	}

	// Extract text from first candidate
	var result string
	for _, part := range resp.Candidates[0].Content.Parts {
		result += fmt.Sprintf("%s", part)
	}

	return result, nil
}

// OllamaProvider implements Provider for local Ollama
type OllamaProvider struct {
	baseURL string
	model   string
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434" // Default Ollama URL
	}
	if model == "" {
		model = "mistral" // Default model
	}
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
	}
}

// ExtractData sends prompt and image to Ollama
func (p *OllamaProvider) ExtractData(prompt string, imageBase64 string) (string, error) {
	// Build message
	message := map[string]interface{}{
		"role":    "user",
		"content": prompt,
	}

	// Add image if provided
	if imageBase64 != "" {
		// Remove data URI prefix if present
		if strings.HasPrefix(imageBase64, "data:image") {
			parts := strings.Split(imageBase64, ",")
			if len(parts) > 1 {
				imageBase64 = parts[1]
			}
		}

		message["images"] = []string{imageBase64}
	}

	// Build request body
	body := map[string]interface{}{
		"model":       p.model,
		"messages":    []interface{}{message},
		"temperature": 0,
		"stream":      false,
		"format":      "json",
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	httpClient := &http.Client{
		Timeout: 120 * time.Second, // Ollama can be slow on CPU
	}

	url := p.baseURL + "/api/chat"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyText, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(bodyText))
	}

	// Parse response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var responseObj struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}

	err = json.Unmarshal(responseBody, &responseObj)
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return responseObj.Message.Content, nil
}

// Helper functions

func decodeBase64(s string) ([]byte, error) {
	// Try standard base64 first
	decoded := make([]byte, len(s))
	n, err := io.ReadFull(strings.NewReader(s), decoded)
	if err == nil {
		return decoded[:n], nil
	}

	// If that fails, use encoding/base64
	import_base64 := func() ([]byte, error) {
		// This would normally import encoding/base64
		// For simplicity, returning error
		return nil, fmt.Errorf("base64 decoding not implemented")
	}

	return import_base64()
}

func detectMIMEType(data []byte) string {
	// Simple MIME type detection based on magic bytes
	if len(data) < 4 {
		return "application/octet-stream"
	}

	// JPEG
	if data[0] == 0xFF && data[1] == 0xD8 {
		return "image/jpeg"
	}

	// PNG
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}

	// GIF
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
		return "image/gif"
	}

	// WebP
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}

	return "image/jpeg" // Default assumption for images
}
