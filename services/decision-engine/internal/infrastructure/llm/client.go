package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// Generic request/response wrappers
type Request struct {
	Data interface{} `json:"data"`
}

type Response struct {
	Data interface{} `json:"data"`
}

// Client represents the LLM microservice client
type Client struct {
	cfg     Config
	http    *http.Client
	breaker *gobreaker.CircuitBreaker
	logger  logger.Logger
}

// NewClient creates a new LLM client with circuit breaker
func NewClient(cfg Config, logger logger.Logger) *Client {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "llm-http",
		MaxRequests: cfg.CBMaxRequests,
		Interval:    cfg.CBInterval,
		Timeout:     cfg.CBTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cfg.CBMaxFailures
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.Warn("LLM circuit breaker state changed",
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

// Intent sends a POST request to /llm/intent
func (c *Client) Intent(ctx context.Context, data interface{}) (interface{}, error) {
	return c.post(ctx, "/llm/intent", data)
}

// Transition sends a POST request to /llm/transition
func (c *Client) Transition(ctx context.Context, data interface{}) (interface{}, error) {
	return c.post(ctx, "/llm/transition", data)
}

// Orchestrate sends a POST request to /llm/decide
func (c *Client) Decide(ctx context.Context, data interface{}) (interface{}, error) {
	return c.post(ctx, "/llm/decide", data)
}

// Response sends a POST request to /llm/response
func (c *Client) Response(ctx context.Context, data interface{}) (interface{}, error) {
	return c.post(ctx, "/llm/generate_response", data)
}

// Summary sends a POST request to /llm/summary
func (c *Client) Summary (ctx context.Context, data interface{}) (interface{}, error) {
	return c.post(ctx, "/llm/summary", data)
}

// post is a generic method that sends POST requests through the circuit breaker
func (c *Client) post(ctx context.Context, endpoint string, data interface{}) (interface{}, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	result, err := c.breaker.Execute(func() (interface{}, error) {
		return c.doPost(ctx, endpoint, body)
	})
	if err != nil {
		return nil, fmt.Errorf("llm client: %w", err)
	}

	return result, nil
}

// doPost performs the actual HTTP POST request
func (c *Client) doPost(ctx context.Context, endpoint string, body []byte) (interface{}, error) {
	url := c.cfg.BaseURL + endpoint

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
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
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Data, nil
}
