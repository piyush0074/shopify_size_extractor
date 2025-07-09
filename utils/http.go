package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"shopify-extractor/internal/types"
)

// HTTPClient provides HTTP functionality with rate limiting and retries
type HTTPClient struct {
	client  *http.Client
	config  *types.Config
	logger  types.Logger
	limiter *time.Ticker
}

// NewHTTPClient creates a new HTTP client with the given configuration
func NewHTTPClient(config *types.Config, logger types.Logger) *HTTPClient {
	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &HTTPClient{
		client:  client,
		config:  config,
		logger:  logger,
		limiter: time.NewTicker(config.RequestDelay),
	}
}

// Get performs a GET request with rate limiting and retries
func (h *HTTPClient) Get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	
	for attempt := 0; attempt <= h.config.MaxRetries; attempt++ {
		// Wait for rate limiter
		select {
		case <-h.limiter.C:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		// Create request
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("User-Agent", h.config.UserAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		req.Header.Set("Accept-Encoding", "gzip, deflate")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Upgrade-Insecure-Requests", "1")

		// Make request
		h.logger.Debugf("Making request to %s (attempt %d/%d)", url, attempt+1, h.config.MaxRetries+1)
		
		resp, err := h.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			h.logger.Warnf("Request failed (attempt %d): %v", attempt+1, err)
			continue
		}

		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			h.logger.Warnf("Unexpected status code %d (attempt %d)", resp.StatusCode, attempt+1)
			continue
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			h.logger.Warnf("Failed to read response body (attempt %d): %v", attempt+1, err)
			continue
		}

		h.logger.Debugf("Successfully retrieved %d bytes from %s", len(body), url)
		return body, nil
	}

	return nil, fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// Close cleans up resources
func (h *HTTPClient) Close() {
	if h.limiter != nil {
		h.limiter.Stop()
	}
} 