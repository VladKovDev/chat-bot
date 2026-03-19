package lemmatizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/sony/gobreaker"
)

type Config struct {
	BaseURL       string
	Timeout       time.Duration
	CBMaxRequests uint32
	CBInterval    time.Duration
	CBMaxFailures uint32
	CBTimeout     time.Duration
}

type request struct {
	Tokens []string `json:"tokens"`
}

type response struct {
	Lemmas []string `json:"lemmas"`
}

type Client struct {
	cfg     Config
	http    *http.Client
	breaker *gobreaker.CircuitBreaker
	logger  logger.Logger
}

func New(cfg Config, logger logger.Logger) *Client {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "lemmatizer-http",
		MaxRequests: cfg.CBMaxRequests,
		Interval:    cfg.CBInterval,
		Timeout:     cfg.CBTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cfg.CBMaxFailures
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.Warn("lemmatizer circuit breaker state changed",
				logger.String("from", from.String()),
				logger.String("to", to.String()),
			)
		},
	})

	return &Client{
		cfg:     cfg,
		http:    &http.Client{Timeout: cfg.Timeout},
		breaker: cb,
		logger:  logger,
	}
}

func (c *Client) Lemmatize(ctx context.Context, tokens []string) ([]string, error) {
	if len(tokens) == 0 {
		return tokens, nil
	}

	body, err := json.Marshal(request{Tokens: tokens})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	result, err := c.breaker.Execute(func() (interface{}, error) {
		return c.post(ctx, body, len(tokens))
	})
	if err != nil {
		return nil, fmt.Errorf("lemmatizer: %w", err)
	}

	return result.([]string), nil
}

func (c *Client) post(ctx context.Context, body []byte, expectedLen int) ([]string, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.cfg.BaseURL+"/lemmatize",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var parsed response
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(parsed.Lemmas) != expectedLen {
		return nil, fmt.Errorf("lemma count mismatch: got %d, expected %d",
			len(parsed.Lemmas), expectedLen)
	}

	return parsed.Lemmas, nil
}
