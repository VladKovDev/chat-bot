package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	if err := c.postJSON(ctx, "/api/v1/sessions", req, &respBody); err != nil {
		return dto.SessionResponse{}, err
	}

	return respBody, nil
}

// SendMessage sends a message to decision engine and returns response
func (c *Client) SendMessage(ctx context.Context, text string, sessionID string, clientID string, eventID string) (dto.DecisionEngineResponse, error) {
	req := dto.DecisionEngineRequest{
		Text:      text,
		SessionID: sessionID,
		EventID:   eventID,
		Type:      "user_message",
		Channel:   WebsiteChannel,
		ClientID:  clientID,
	}

	var respBody dto.DecisionEngineResponse
	if err := c.postJSON(ctx, "/api/v1/messages", req, &respBody); err != nil {
		return dto.DecisionEngineResponse{}, err
	}

	return respBody, nil
}

func (c *Client) RequestHandoff(ctx context.Context, sessionID string) (dto.OperatorQueueActionResponse, error) {
	var respBody dto.OperatorQueueActionResponse
	if err := c.postJSON(ctx, fmt.Sprintf("/api/v1/operator/queue/%s/request", sessionID), map[string]string{}, &respBody); err != nil {
		return dto.OperatorQueueActionResponse{}, err
	}
	return respBody, nil
}

func (c *Client) CloseHandoff(ctx context.Context, sessionID string) (dto.OperatorQueueActionResponse, error) {
	var respBody dto.OperatorQueueActionResponse
	if err := c.postJSON(ctx, fmt.Sprintf("/api/v1/operator/queue/%s/close", sessionID), map[string]string{}, &respBody); err != nil {
		return dto.OperatorQueueActionResponse{}, err
	}
	return respBody, nil
}

func (c *Client) GetSessionMessages(ctx context.Context, sessionID string) (dto.SessionMessagesResponse, error) {
	var respBody dto.SessionMessagesResponse
	if err := c.getJSON(ctx, fmt.Sprintf("/api/v1/sessions/%s/messages", sessionID), &respBody); err != nil {
		return dto.SessionMessagesResponse{}, err
	}
	return respBody, nil
}

func (c *Client) GetOperatorQueue(ctx context.Context, status string) (dto.OperatorQueueResponse, error) {
	path := "/api/v1/operator/queue"
	if status != "" {
		path = fmt.Sprintf("%s?status=%s", path, url.QueryEscape(status))
	}

	var respBody dto.OperatorQueueResponse
	if err := c.getJSON(ctx, path, &respBody); err != nil {
		return dto.OperatorQueueResponse{}, err
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var publicError dto.ErrorEnvelope
		if err := json.NewDecoder(resp.Body).Decode(&publicError); err != nil {
			return fmt.Errorf("decision engine returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("%s: %s (request_id=%s)", publicError.Error.Code, publicError.Error.Message, publicError.Error.RequestID)
	}

	if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
		c.logger.Error("failed to decode response",
			logger.Err(err),
			logger.Int("status", resp.StatusCode),
		)
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *Client) getJSON(ctx context.Context, path string, respBody interface{}) error {
	url := fmt.Sprintf("%s%s", c.baseURL, path)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("failed to create request",
			logger.Err(err),
			logger.String("url", url),
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var publicError dto.ErrorEnvelope
		if err := json.NewDecoder(resp.Body).Decode(&publicError); err != nil {
			return fmt.Errorf("decision engine returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("%s: %s (request_id=%s)", publicError.Error.Code, publicError.Error.Message, publicError.Error.RequestID)
	}

	if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
		c.logger.Error("failed to decode response",
			logger.Err(err),
			logger.Int("status", resp.StatusCode),
		)
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}
