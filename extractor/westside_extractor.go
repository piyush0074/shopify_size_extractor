package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"shopify-extractor/adapters"
	"shopify-extractor/internal/types"
)

// WestsideExtractor handles extraction for Westside store only
type WestsideExtractor struct {
	adapter *adapters.WestsideAdapter
	logger  types.Logger
}

// NewWestsideExtractor creates a new Westside extractor
func NewWestsideExtractor(config *types.Config, logger types.Logger) *WestsideExtractor {
	return &WestsideExtractor{
		adapter: adapters.NewWestsideAdapter(config, logger),
		logger:  logger,
	}
}

// ExtractAll extracts all size charts from Westside
func (w *WestsideExtractor) ExtractAll(ctx context.Context) ([]types.Product, error) {
	startTime := time.Now()
	w.logger.Infof("Starting Westside extraction at %v", startTime.Format("15:04:05.000"))

	// Step 1: Get all product URLs
	w.logger.Info("Step 1: Discovering product URLs...")
	storeCtx := types.Context{
		Config: w.adapter.Config(),
		Logger: w.logger,
	}
	productURLs, err := w.adapter.GetProductURLs(storeCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get product URLs: %w", err)
	}

	w.logger.Infof("Found %d product URLs", len(productURLs))

	// Step 2: Extract size charts from each product
	w.logger.Info("Step 2: Extracting size charts...")
	var results []types.Product
	processedCount := 0

	for i, productURL := range productURLs {
		productStartTime := time.Now()
		w.logger.Debugf("Processing product %d/%d: %s", i+1, len(productURLs), productURL)

		// Only fetch the product page once and extract both title and size charts
		title, sizeCharts, err := w.adapter.ExtractAllSizeCharts(storeCtx, productURL)
		if err != nil {
			w.logger.Warnf("Failed to extract size charts for %s: %v", productURL, err)
			continue
		}

		if len(sizeCharts) > 0 {
			// Use the extracted title, fallback to "Unknown Product" if empty
			if title == "" {
				title = "Unknown Product"
			}
			result := types.Product{
				ProductTitle: title,
				ProductURL:   productURL,
				SizeCharts:   sizeCharts,
			}
			results = append(results, result)
			w.logger.Debugf("Extracted %d size charts for %s", len(sizeCharts), productURL)
			processedCount++
		}

		productTime := time.Since(productStartTime)
		w.logger.Debugf("Product %s processed in %v", productURL, productTime)

		// if i >= 5 {
		// 	break // limit exceed
		// }

	}

	totalTime := time.Since(startTime)
	w.logger.Infof("Westside extraction completed in %v", totalTime)
	w.logger.Infof("Successfully processed %d/%d products", processedCount, len(productURLs))

	return results, nil
}

// ExtractToJSON extracts all size charts and saves to JSON file
func (w *WestsideExtractor) ExtractToJSON(ctx context.Context, filename string) error {
	results, err := w.ExtractAll(ctx)
	if err != nil {
		return err
	}

	// Save to JSON file
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results to JSON: %w", err)
	}

	if err := writeToFile(filename, jsonData); err != nil {
		return fmt.Errorf("failed to write results to file: %w", err)
	}

	w.logger.Infof("Results saved to %s", filename)
	return nil
}

// Close cleans up resources
func (w *WestsideExtractor) Close() {
	if w.adapter != nil {
		w.adapter.Close()
	}
}

// writeToFile writes data to a file
func writeToFile(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0644)
}
