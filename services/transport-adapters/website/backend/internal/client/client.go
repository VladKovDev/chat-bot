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

const WebsiteChannel = "website"

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

func (c *Client) StartSession(ctx context.Context, clientID string) (dto.SessionResponse, error) {
	req := dto.SessionRequest{
		Channel:  WebsiteChannel,
		ClientID: clientID,
	}

	var respBody dto.SessionResponse
	if err := c.postJSON(ctx, "/sessions", req, &respBody); err != nil {
		if respBody.Error != nil {
			return respBody, err
		}
		return dto.SessionResponse{
				Success: false,
				Error:   safePublicError("session_start_failed", "Не удалось начать сессию.", ""),
			},
			err
	}

	return respBody, nil
}

// SendMessage sends a message to decision engine and returns response
func (c *Client) SendMessage(ctx context.Context, text string, sessionID string, clientID string) (dto.DecisionEngineResponse, error) {
	req := dto.DecisionEngineRequest{
		Text:      text,
		SessionID: sessionID,
		Channel:   WebsiteChannel,
		ClientID:  clientID,
	}

	var respBody dto.DecisionEngineResponse
	if err := c.postJSON(ctx, "/decide", req, &respBody); err != nil {
		if respBody.Error != nil {
			return respBody, err
		}
		return dto.DecisionEngineResponse{
				Success: false,
				Error:   safePublicError("processing_failed", "Не удалось обработать сообщение. Попробуйте позже.", ""),
			},
			err
	}

	return respBody, nil
}

func (c *Client) postJSON(ctx context.Context, path string, req interface{}, respBody interface{}) error {
	reqBody, err := json.Marshal(req)
	if err != nil {
		c.logger.Error("failed to marshal request",
			logger.Err(err),
		)
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, path)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		c.logger.Error("failed to create request",
			logger.Err(err),
			logger.String("url", url),
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	c.logger.Info("sending request to decision engine",
		logger.String("url", url),
	)

	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.Error("failed to send request",
			logger.Err(err),
			logger.String("url", url),
			logger.String("duration", time.Since(start).String()),
		)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)
	c.logger.Info("received response from decision engine",
		logger.Int("status", resp.StatusCode),
		logger.String("duration", duration.String()),
	)

	if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
		c.logger.Error("failed to decode response",
			logger.Err(err),
			logger.Int("status", resp.StatusCode),
		)
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("decision engine returned status %d", resp.StatusCode)
	}

	return nil
}

func safePublicError(code, message, requestID string) *dto.PublicError {
	return &dto.PublicError{
		Code:      code,
		Message:   message,
		RequestID: requestID,
	}
}
