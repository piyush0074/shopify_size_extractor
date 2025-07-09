package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"shopify-extractor/adapters"
	"shopify-extractor/internal/types"
)

// SuqahExtractor handles extraction for Suqah store only
type SuqahExtractor struct {
	adapter *adapters.SuqahAdapter
	logger  types.Logger
}

// NewSuqahExtractor creates a new Suqah extractor
func NewSuqahExtractor(config *types.Config, logger types.Logger) *SuqahExtractor {
	return &SuqahExtractor{
		adapter: adapters.NewSuqahAdapter(config, logger),
		logger:  logger,
	}
}

// ExtractAll extracts all size charts from Suqah
func (s *SuqahExtractor) ExtractAll(ctx context.Context) ([]types.Product, error) {
	startTime := time.Now()
	s.logger.Infof("Starting Suqah extraction at %v", startTime.Format("15:04:05.000"))

	s.logger.Info("Step 1: Discovering product URLs...")
	storeCtx := types.Context{
		Config: s.adapter.Config(),
		Logger: s.logger,
	}
	productURLs, err := s.adapter.GetProductURLs(storeCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get product URLs: %w", err)
	}

	s.logger.Infof("Found %d product URLs", len(productURLs))

	s.logger.Info("Step 2: Extracting size charts...")
	var results []types.Product
	processedCount := 0

	for i, productURL := range productURLs {
		productStartTime := time.Now()
		s.logger.Debugf("Processing product %d/%d: %s", i+1, len(productURLs), productURL)

		// Use optimized method that fetches page once and extracts both title and size charts
		title, sizeCharts, err := s.adapter.ExtractProductData(storeCtx, productURL)
		if err != nil {
			s.logger.Warnf("Failed to extract data for %s: %v", productURL, err)
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
		s.logger.Debugf("Product %s processed in %v", productURL, productTime)
		// if i >= 5 {
		// 	break // limit exceed
		// }

	}

	totalTime := time.Since(startTime)
	s.logger.Infof("Suqah extraction completed in %v", totalTime)
	s.logger.Infof("Successfully processed %d/%d products", processedCount, len(productURLs))

	return results, nil
}

// ExtractToJSON extracts all size charts and saves to JSON file
func (s *SuqahExtractor) ExtractToJSON(ctx context.Context, filename string) error {
	results, err := s.ExtractAll(ctx)
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

	s.logger.Infof("Results saved to %s", filename)
	return nil
}

// Close cleans up resources
func (s *SuqahExtractor) Close() {
	if s.adapter != nil {
		s.adapter.Close()
	}
}
