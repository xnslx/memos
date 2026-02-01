package embedding

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	huggingFaceAPIURL = "https://router.huggingface.co/hf-inference/models/sentence-transformers/all-MiniLM-L6-v2/pipeline/feature-extraction"
)

func getHuggingFaceToken() string {
	return os.Getenv("HF_TOKEN")
}

type EmbeddingService struct{}

func NewEmbeddingService() *EmbeddingService {
	return &EmbeddingService{}
}

type EmbeddingRequest struct {
	Inputs  string                 `json:"inputs"`
	Options map[string]interface{} `json:"options,omitempty"`
}

func (s *EmbeddingService) RegisterRoutes(e *echo.Echo) {
	e.POST("/api/embedding", s.generateEmbedding)
}

func (s *EmbeddingService) generateEmbedding(c echo.Context) error {
	// Parse request body
	var req EmbeddingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.Inputs == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "inputs field is required"})
	}

	// Add wait_for_model option
	if req.Options == nil {
		req.Options = make(map[string]interface{})
	}
	req.Options["wait_for_model"] = true

	// Marshal request for HuggingFace
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to marshal request"})
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Create request to HuggingFace
	hfReq, err := http.NewRequest("POST", huggingFaceAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create request"})
	}

	hfReq.Header.Set("Authorization", "Bearer "+getHuggingFaceToken())
	hfReq.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := client.Do(hfReq)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Failed to call HuggingFace API: " + err.Error()})
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read response"})
	}

	// If HuggingFace returned an error, pass it through
	if resp.StatusCode != http.StatusOK {
		return c.JSONBlob(resp.StatusCode, body)
	}

	// Return the embedding
	return c.JSONBlob(http.StatusOK, body)
}
