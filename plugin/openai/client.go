package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
)

// Client is a simple HTTP client for OpenAI API.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a chat completion request.
type ChatRequest struct {
	Model               string    `json:"model"`
	Messages            []Message `json:"messages"`
	MaxCompletionTokens int       `json:"max_completion_tokens,omitempty"`
}

// ChatResponse represents a chat completion response.
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewClient creates a new OpenAI client using environment variables.
func NewClient() (*Client, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY environment variable is required")
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-5-nano-2025-08-07"
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Chat sends a chat completion request and returns the response content.
func (c *Client) Chat(messages []Message, maxTokens int) (string, error) {
	req := ChatRequest{
		Model:               c.model,
		Messages:            messages,
		MaxCompletionTokens: maxTokens,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal request")
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", errors.Wrap(err, "failed to create request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response")
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("OpenAI API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal response: %s", string(respBody))
	}

	if len(chatResp.Choices) == 0 {
		return "", errors.Errorf("no choices in response: %s", string(respBody))
	}

	content := chatResp.Choices[0].Message.Content
	if content == "" {
		return "", errors.Errorf("empty content in response. Finish reason: %s, Full response: %s",
			chatResp.Choices[0].FinishReason, string(respBody))
	}

	return content, nil
}
