package extractor

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"shopify-extractor/internal/types"
)

func TestNewExtractor(t *testing.T) {
	config := types.DefaultConfig()
	logger := logrus.New()
	
	extractor := NewExtractor(config, logger)
	
	assert.NotNil(t, extractor)
	assert.Equal(t, config, extractor.config)
	assert.Equal(t, logger, extractor.logger)
	assert.NotNil(t, extractor.adapters)
	
	// Check that Westside adapter is initialized
	_, exists := extractor.adapters["westside.com"]
	assert.True(t, exists)
}

func TestExtractSizeCharts_EmptyStores(t *testing.T) {
	config := types.DefaultConfig()
	logger := logrus.New()
	extractor := NewExtractor(config, logger)
	defer extractor.Close()
	
	ctx := context.Background()
	results, err := extractor.ExtractSizeCharts(ctx, []string{})
	
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results.Stores)
}

func TestExtractSizeCharts_UnsupportedStore(t *testing.T) {
	config := types.DefaultConfig()
	logger := logrus.New()
	extractor := NewExtractor(config, logger)
	defer extractor.Close()
	
	ctx := context.Background()
	results, err := extractor.ExtractSizeCharts(ctx, []string{"unsupported-store.com"})
	
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results.Stores, 1)
	assert.Equal(t, "unsupported-store.com", results.Stores[0].StoreName)
	assert.Contains(t, results.Stores[0].Error, "no adapter found")
}

func TestExtractStore_ValidStore(t *testing.T) {
	config := types.DefaultConfig()
	logger := logrus.New()
	extractor := NewExtractor(config, logger)
	defer extractor.Close()
	
	ctx := context.Background()
	result, err := extractor.extractStore(ctx, "westside.com")
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "westside.com", result.StoreName)
	assert.Empty(t, result.Error)
}

func TestExtractStore_InvalidStore(t *testing.T) {
	config := types.DefaultConfig()
	logger := logrus.New()
	extractor := NewExtractor(config, logger)
	defer extractor.Close()
	
	ctx := context.Background()
	result, err := extractor.extractStore(ctx, "invalid-store.com")
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "invalid-store.com", result.StoreName)
	assert.Contains(t, result.Error, "no adapter found")
} 