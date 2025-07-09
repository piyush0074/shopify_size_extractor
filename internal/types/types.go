package types

import "time"

// SizeChart represents a product size chart
type SizeChart struct {
	Headers []string            `json:"headers"`
	Rows    []map[string]string `json:"rows"`
}

// Product represents a product with its size chart
type Product struct {
	ProductTitle string       `json:"product_title"`
	ProductURL   string       `json:"product_url"`
	SizeCharts   []*SizeChart `json:"size_chart,omitempty"`
}

// StoreResult represents the extraction result for a single store
type StoreResult struct {
	StoreName string    `json:"store_name"`
	Products  []Product `json:"products"`
	Error     string    `json:"error,omitempty"`
}

// ExtractionResult represents the complete extraction result
type ExtractionResult struct {
	Stores []StoreResult `json:"stores"`
}

// Config holds the configuration for the extractor
type Config struct {
	RequestDelay           time.Duration
	MaxRetries            int
	Timeout               time.Duration
	MaxConcurrentRequests int
	UseHeadlessBrowser    bool
	UserAgent             string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		RequestDelay:           1 * time.Second,
		MaxRetries:            3,
		Timeout:               30 * time.Second,
		MaxConcurrentRequests: 5,
		UseHeadlessBrowser:    true,
		UserAgent:             "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

// StoreAdapter defines the interface for store-specific extraction logic
type StoreAdapter interface {
	// GetStoreName returns the name of the store
	GetStoreName() string
	
	// GetProductURLs returns a list of product URLs for the store
	GetProductURLs(ctx Context) ([]string, error)
	
	// ExtractSizeChart extracts the size chart from a product page
	ExtractSizeChart(ctx Context, productURL string) (*SizeChart, error)
	
	// GetProductTitle extracts the product title from a product page
	GetProductTitle(ctx Context, productURL string) (string, error)
}

// Context provides context for extraction operations
type Context struct {
	Config *Config
	Logger Logger
}

// Logger defines the logging interface
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
} 