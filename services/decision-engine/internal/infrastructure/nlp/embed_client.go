package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type EmbedderConfig struct {
	BaseURL           string        `mapstructure:"base_url"`
	Timeout           time.Duration `mapstructure:"timeout"`
	ExpectedDimension int           `mapstructure:"expected_dimension"`
}

type EmbedderClient struct {
	baseURL           string
	expectedDimension int
	httpClient        *http.Client
}

func NewEmbedderClient(cfg EmbedderConfig) (*EmbedderClient, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("nlp embedder base_url is required")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &EmbedderClient{
		baseURL:           baseURL,
		expectedDimension: cfg.ExpectedDimension,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *EmbedderClient) Embed(ctx context.Context, text string) ([]float64, error) {
	body, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call embed endpoint: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read embed response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed endpoint returned status %d", resp.StatusCode)
	}

	var payload struct {
		Embedding []float64 `json:"embedding"`
		Model     string    `json:"model"`
		Dimension int       `json:"dimension"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}
	if len(payload.Embedding) == 0 {
		return nil, fmt.Errorf("embed response contains empty embedding")
	}
	if c.expectedDimension > 0 && len(payload.Embedding) != c.expectedDimension {
		return nil, fmt.Errorf("embed dimension = %d, want %d", len(payload.Embedding), c.expectedDimension)
	}
	return payload.Embedding, nil
}
