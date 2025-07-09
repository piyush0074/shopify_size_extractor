package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"shopify-extractor/internal/types"
)

func TestNewHTTPClient(t *testing.T) {
	config := types.DefaultConfig()
	logger := logrus.New()
	
	client := NewHTTPClient(config, logger)
	
	assert.NotNil(t, client)
	assert.Equal(t, config, client.config)
	assert.Equal(t, logger, client.logger)
	assert.NotNil(t, client.client)
	assert.NotNil(t, client.limiter)
	
	client.Close()
}

func TestHTTPClient_Get_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()
	
	config := types.DefaultConfig()
	config.RequestDelay = 10 * time.Millisecond // Faster for testing
	logger := logrus.New()
	client := NewHTTPClient(config, logger)
	defer client.Close()
	
	ctx := context.Background()
	body, err := client.Get(ctx, server.URL)
	
	require.NoError(t, err)
	assert.Equal(t, "test response", string(body))
}

func TestHTTPClient_Get_NotFound(t *testing.T) {
	// Create test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	
	config := types.DefaultConfig()
	config.RequestDelay = 10 * time.Millisecond
	config.MaxRetries = 1 // Reduce retries for faster test
	logger := logrus.New()
	client := NewHTTPClient(config, logger)
	defer client.Close()
	
	ctx := context.Background()
	_, err := client.Get(ctx, server.URL)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 404")
}

func TestHTTPClient_Get_ContextCancelled(t *testing.T) {
	config := types.DefaultConfig()
	config.RequestDelay = 100 * time.Millisecond
	logger := logrus.New()
	client := NewHTTPClient(config, logger)
	defer client.Close()
	
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	_, err := client.Get(ctx, "http://example.com")
	
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestHTTPClient_Close(t *testing.T) {
	config := types.DefaultConfig()
	logger := logrus.New()
	client := NewHTTPClient(config, logger)
	
	// Should not panic
	client.Close()
} 