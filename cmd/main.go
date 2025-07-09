package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"shopify-extractor/extractor"
	"shopify-extractor/internal/types"
)

func main() {
	// Load .env file if present
	_ = godotenv.Load()

	// Parse command line flags
	var (
		storeFlag      = flag.String("store", "", "Single store to extract (westside, littleboxindia, suqah)")
		storesFlag     = flag.String("stores", "", "Comma-separated list of store domains (for multi-store extraction)")
		outputFlag     = flag.String("output", "", "Output file path (default: stdout)")
		requestDelay   = flag.Duration("delay", 1*time.Second, "Delay between requests")
		maxRetries     = flag.Int("retries", 3, "Maximum retry attempts")
		timeout        = flag.Duration("timeout", 30*time.Second, "Request timeout")
		maxConcurrent  = flag.Int("concurrent", 5, "Maximum concurrent requests")
		useBrowser     = flag.Bool("browser", true, "Use headless browser for JavaScript-heavy sites")
		httpOnly       = flag.Bool("http-only", false, "Use HTTP requests only (disable headless browser)")
		verbose        = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	// Validate flags - either --store or --stores must be provided
	if *storeFlag == "" && *storesFlag == "" {
		log.Fatal("Either --store or --stores flag is required")
	}
	if *storeFlag != "" && *storesFlag != "" {
		log.Fatal("Cannot use both --store and --stores flags")
	}

	// Parse stores
	var stores []string
	if *storeFlag != "" {
		// Single store mode
		stores = []string{strings.TrimSpace(*storeFlag)}
	} else {
		// Multi-store mode
		stores = strings.Split(*storesFlag, ",")
		for i, store := range stores {
			stores[i] = strings.TrimSpace(store)
		}
	}

	// Setup logging
	logger := logrus.New()
	
	// Set timestamp format with milliseconds
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})
	
	// Set log level from LOG_LEVEL env if present
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		if level, err := logrus.ParseLevel(levelStr); err == nil {
			logger.SetLevel(level)
		}
	} else if *verbose {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// Create configuration
	config := &types.Config{
		RequestDelay:           *requestDelay,
		MaxRetries:            *maxRetries,
		Timeout:               *timeout,
		MaxConcurrentRequests: *maxConcurrent,
		UseHeadlessBrowser:    *useBrowser && !*httpOnly,
		UserAgent:             "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Extract size charts using individual store extractors
	startTime := time.Now()
	logger.Infof("Starting extraction for stores: %v", stores)
	
	var storeResults []types.StoreResult
	totalProducts := 0
	productsWithSizeCharts := 0

	for _, store := range stores {
		logger.Infof("Processing store: %s", store)
		
		var storeExtractor interface {
			ExtractAll(context.Context) ([]types.Product, error)
			Close()
		}
		
		// Create the appropriate extractor based on store name
		switch store {
		case "westside.com":
			storeExtractor = extractor.NewWestsideExtractor(config, logger)
		case "littleboxindia.com":
			storeExtractor = extractor.NewLittleBoxIndiaExtractor(config, logger)
		case "suqah.com":
			storeExtractor = extractor.NewSuqahExtractor(config, logger)
		default:
			logger.Warnf("Unknown store: %s, skipping", store)
			continue
		}
		
		defer storeExtractor.Close()
		
		// Extract from this store
		products, err := storeExtractor.ExtractAll(ctx)
		if err != nil {
			logger.Warnf("Failed to extract from %s: %v", store, err)
			continue
		}
		
		// Create store result with actual store name
		storeResult := types.StoreResult{
			StoreName: store,
			Products:  products,
		}
		storeResults = append(storeResults, storeResult)
		
		totalProducts += len(products)
		for _, product := range products {
			if len(product.SizeCharts) > 0 {
				productsWithSizeCharts++
			}
		}
	}
	
	extractionTime := time.Since(startTime)
	logger.Infof("Extraction completed in %v", extractionTime)

	// Create the final result structure with separate store results
	finalResults := types.ExtractionResult{
		Stores: storeResults,
	}

	// Marshal results to JSON
	jsonData, err := json.MarshalIndent(finalResults, "", "  ")
	if err != nil {
		logger.Fatalf("Failed to marshal results: %v", err)
	}

	// Output results
	if *outputFlag != "" {
		// Write to file
		err = os.WriteFile(*outputFlag, jsonData, 0644)
		if err != nil {
			logger.Fatalf("Failed to write output file: %v", err)
		}
		logger.Infof("Results written to: %s", *outputFlag)
	} else {
		// Write to stdout
		fmt.Println(string(jsonData))
	}

	// Print summary
	logger.Infof("Extraction completed successfully")
	logger.Infof("Total stores processed: %d", len(stores))
	logger.Infof("Total products found: %d", totalProducts)
	logger.Infof("Products with size charts: %d", productsWithSizeCharts)
} 