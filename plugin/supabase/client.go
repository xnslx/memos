package supabase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
)

// Client is a simple HTTP client for Supabase REST API.
type Client struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

// MemoEmbedding represents a row from the memo_embeddings table.
type MemoEmbedding struct {
	ID        int64     `json:"id"`
	MemoName  string    `json:"memo_name"`
	Content   string    `json:"content"`
	Embedding []float64 `json:"embedding"`
	CreatedAt time.Time `json:"created_at"`
}

// NewClient creates a new Supabase client using environment variables.
// Required env vars: SUPABASE_URL, SUPABASE_SERVICE_KEY
func NewClient() (*Client, error) {
	baseURL := os.Getenv("SUPABASE_URL")
	if baseURL == "" {
		return nil, errors.New("SUPABASE_URL environment variable is required")
	}

	serviceKey := os.Getenv("SUPABASE_SERVICE_KEY")
	if serviceKey == "" {
		return nil, errors.New("SUPABASE_SERVICE_KEY environment variable is required")
	}

	return &Client{
		baseURL:    baseURL,
		serviceKey: serviceKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// NewClientWithConfig creates a new Supabase client with explicit configuration.
func NewClientWithConfig(baseURL, serviceKey string) (*Client, error) {
	if baseURL == "" {
		return nil, errors.New("baseURL is required")
	}
	if serviceKey == "" {
		return nil, errors.New("serviceKey is required")
	}

	return &Client{
		baseURL:    baseURL,
		serviceKey: serviceKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// GetAllEmbeddings fetches all memo embeddings from Supabase.
func (c *Client) GetAllEmbeddings() ([]MemoEmbedding, error) {
	url := fmt.Sprintf("%s/rest/v1/memo_embeddings?select=id,memo_name,content,embedding,created_at", c.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch embeddings")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("supabase API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var embeddings []MemoEmbedding
	if err := json.NewDecoder(resp.Body).Decode(&embeddings); err != nil {
		return nil, errors.Wrap(err, "failed to decode embeddings response")
	}

	return embeddings, nil
}

// GetEmbeddingsByMemoNames fetches embeddings for specific memo names.
func (c *Client) GetEmbeddingsByMemoNames(memoNames []string) ([]MemoEmbedding, error) {
	if len(memoNames) == 0 {
		return []MemoEmbedding{}, nil
	}

	// Build the filter query
	// Supabase uses PostgREST syntax: memo_name=in.(value1,value2,...)
	namesCSV := ""
	for i, name := range memoNames {
		if i > 0 {
			namesCSV += ","
		}
		namesCSV += fmt.Sprintf("\"%s\"", name)
	}

	url := fmt.Sprintf("%s/rest/v1/memo_embeddings?select=id,memo_name,content,embedding,created_at&memo_name=in.(%s)",
		c.baseURL, namesCSV)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch embeddings by memo names")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("supabase API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var embeddings []MemoEmbedding
	if err := json.NewDecoder(resp.Body).Decode(&embeddings); err != nil {
		return nil, errors.Wrap(err, "failed to decode embeddings response")
	}

	return embeddings, nil
}

// setHeaders sets the required headers for Supabase API requests.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.serviceKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
}
