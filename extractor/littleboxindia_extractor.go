package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"shopify-extractor/adapters"
	"shopify-extractor/internal/types"
)

// LittleBoxIndiaExtractor handles extraction for LittleBoxIndia store only
type LittleBoxIndiaExtractor struct {
	adapter *adapters.LittleBoxIndiaAdapter
	logger  types.Logger
}

// NewLittleBoxIndiaExtractor creates a new LittleBoxIndia extractor
func NewLittleBoxIndiaExtractor(config *types.Config, logger types.Logger) *LittleBoxIndiaExtractor {
	return &LittleBoxIndiaExtractor{
		adapter: adapters.NewLittleBoxIndiaAdapter(config, logger),
		logger:  logger,
	}
}

// ExtractAll extracts all size charts from LittleBoxIndia
func (l *LittleBoxIndiaExtractor) ExtractAll(ctx context.Context) ([]types.Product, error) {
	startTime := time.Now()
	l.logger.Infof("Starting LittleBoxIndia extraction at %v", startTime.Format("15:04:05.000"))

	// Step 1: Get all product URLs
	l.logger.Info("Step 1: Discovering product URLs...")
	storeCtx := types.Context{
		Config: l.adapter.Config(),
		Logger: l.logger,
	}
	productURLs, err := l.adapter.GetProductURLs(storeCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get product URLs: %w", err)
	}

	l.logger.Infof("Found %d product URLs", len(productURLs))

	// Step 2: Extract size charts from each product
	l.logger.Info("Step 2: Extracting size charts...")
	var results []types.Product
	processedCount := 0

	for i, productURL := range productURLs {
		productStartTime := time.Now()
		l.logger.Debugf("Processing product %d/%d: %s", i+1, len(productURLs), productURL)

		// Use optimized method that fetches page once and extracts both title and size charts
		title, sizeCharts, err := l.adapter.ExtractProductTitleAndSizeCharts(storeCtx, productURL)
		if err != nil {
			l.logger.Warnf("Failed to extract data for %s: %v", productURL, err)
			continue
		}

		if len(sizeCharts) > 0 {
			result := types.Product{
				ProductTitle: title,
				ProductURL:   productURL,
				SizeCharts:   sizeCharts,
			}
			results = append(results, result)
			processedCount++
		}

		productTime := time.Since(productStartTime)
		l.logger.Debugf("Product %s processed in %v", productURL, productTime)
		if i >= 5 {
			break // limit exceed
		}
	}

	totalTime := time.Since(startTime)
	l.logger.Infof("LittleBoxIndia extraction completed in %v", totalTime)
	l.logger.Infof("Successfully processed %d/%d products", processedCount, len(productURLs))

	return results, nil
}

// ExtractToJSON extracts all size charts and saves to JSON file
func (l *LittleBoxIndiaExtractor) ExtractToJSON(ctx context.Context, filename string) error {
	results, err := l.ExtractAll(ctx)
	if err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results to JSON: %w", err)
	}

	if err := writeToFile(filename, jsonData); err != nil {
		return fmt.Errorf("failed to write results to file: %w", err)
	}

	l.logger.Infof("Results saved to %s", filename)
	return nil
}

// Close cleans up resources
func (l *LittleBoxIndiaExtractor) Close() {
	if l.adapter != nil {
		l.adapter.Close()
	}
}
