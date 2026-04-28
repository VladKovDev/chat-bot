package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/VladKovDev/web-adapter/internal/config"
	"github.com/VladKovDev/web-adapter/internal/dto"
	"github.com/VladKovDev/web-adapter/pkg/logger"
)

// Client represents a decision engine HTTP client
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     logger.Logger
}

// NewClient creates a new decision engine client
func NewClient(cfg config.DecisionEngine, log logger.Logger) *Client {
	return &Client{
		baseURL: cfg.URL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: log,
	}
}

// SendMessage sends a message to decision engine and returns response
func (c *Client) SendMessage(ctx context.Context, text string, chatID int64) (dto.DecisionEngineResponse, error) {
	req := dto.DecisionEngineRequest{
		Text:   text,
		ChatID: chatID,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		c.logger.Error("failed to marshal request",
			logger.Err(err),
			logger.String("text", text),
			logger.Int64("chat_id", chatID),
		)
		return dto.DecisionEngineResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/decide", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		c.logger.Error("failed to create request",
			logger.Err(err),
			logger.String("url", url),
		)
		return dto.DecisionEngineResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	c.logger.Info("sending request to decision engine",
		logger.String("url", url),
		logger.Int64("chat_id", chatID),
	)

	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.Error("failed to send request",
			logger.Err(err),
			logger.String("url", url),
			logger.String("duration", time.Since(start).String()),
		)
		return dto.DecisionEngineResponse{
				Success: false,
				Error:   "Failed to connect to decision engine",
			},
			fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)
	c.logger.Info("received response from decision engine",
		logger.Int("status", resp.StatusCode),
		logger.String("duration", duration.String()),
	)

	var respBody dto.DecisionEngineResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		c.logger.Error("failed to decode response",
			logger.Err(err),
			logger.Int("status", resp.StatusCode),
		)
		return dto.DecisionEngineResponse{
				Success: false,
				Error:   "Failed to decode response",
			},
			fmt.Errorf("failed to decode response: %w", err)
	}

	return respBody, nil
}