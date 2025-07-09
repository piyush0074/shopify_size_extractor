# Development Guide

## Getting Started

### Prerequisites

1. **Go 1.19+**: Install from [golang.org](https://golang.org/dl/)
2. **Git**: For version control
3. **Chrome/Chromium**: For headless browser automation
4. **Code Editor**: VS Code, GoLand, or Vim with Go support

### Development Environment Setup

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd shopify_extractor
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   go mod tidy
   ```

3. **Verify setup**:
   ```bash
   go test ./...
   go run cmd/main.go --help
   ```

## Project Structure

### Key Directories

- `adapters/`: Store-specific web scraping adapters
- `extractor/`: High-level extraction orchestration
- `cmd/`: Application entry points (CLI and API)
- `internal/`: Internal packages and types
- `utils/`: Utility functions and helpers
- `docs/`: Documentation files

### Code Organization

```
shopify_extractor/
├── adapters/                 # Store adapters
│   ├── base.go              # Base adapter with common functionality
│   ├── westside.go          # Westside store implementation
│   ├── littleboxindia.go    # LittleBoxIndia store implementation
│   └── suqah.go             # Suqah store implementation
├── extractor/               # Extraction orchestration
│   ├── westside_extractor.go
│   ├── littleboxindia_extractor.go
│   └── suqah_extractor.go
├── cmd/                     # Application entry points
│   ├── main.go              # CLI application
│   └── api/                 # API server
│       └── main.go
├── internal/                # Internal packages
│   └── types/               # Type definitions
│       └── types.go
├── utils/                   # Utilities
│   ├── browser.go           # Browser automation
│   └── http.go              # HTTP utilities
└── docs/                    # Documentation
    ├── ARCHITECTURE.md
    └── DEVELOPMENT.md
```

## Adding a New Store

### Step 1: Create the Adapter

Create a new file `adapters/newstore.go`:

```go
package adapters

import (
    "context"
    "fmt"
    "shopify-extractor/internal/types"
    "github.com/PuerkitoBio/goquery"
)

// NewStoreAdapter handles extraction for newstore.com
type NewStoreAdapter struct {
    *BaseAdapter
}

// NewNewStoreAdapter creates a new NewStore adapter
func NewNewStoreAdapter(config *types.Config, logger types.Logger) *NewStoreAdapter {
    return &NewStoreAdapter{
        BaseAdapter: NewBaseAdapter(config, logger),
    }
}

// GetStoreName returns the store name
func (n *NewStoreAdapter) GetStoreName() string {
    return "newstore.com"
}

// GetProductURLs returns a list of product URLs for NewStore
func (n *NewStoreAdapter) GetProductURLs(ctx types.Context) ([]string, error) {
    // Implement product discovery logic
    // 1. Get products page
    // 2. Extract collection URLs
    // 3. Extract product URLs from collections
    // 4. Return unique product URLs
}

// GetProductTitle extracts the product title
func (n *NewStoreAdapter) GetProductTitle(ctx types.Context, productURL string) (string, error) {
    // Implement title extraction logic
    // Use appropriate CSS selectors for the store
}

// ExtractSizeChart extracts size chart data
func (n *NewStoreAdapter) ExtractSizeChart(ctx types.Context, productURL string) (*types.SizeChart, error) {
    // Implement size chart extraction logic
    // 1. Fetch product page
    // 2. Find size chart table
    // 3. Parse headers and rows
    // 4. Return structured data
}
```

### Step 2: Create the Extractor

Create a new file `extractor/newstore_extractor.go`:

```go
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

// NewStoreExtractor handles extraction for NewStore
type NewStoreExtractor struct {
    adapter *adapters.NewStoreAdapter
    logger  types.Logger
}

// NewNewStoreExtractor creates a new NewStore extractor
func NewNewStoreExtractor(config *types.Config, logger types.Logger) *NewStoreExtractor {
    return &NewStoreExtractor{
        adapter: adapters.NewNewStoreAdapter(config, logger),
        logger:  logger,
    }
}

// ExtractAll extracts all size charts from NewStore
func (n *NewStoreExtractor) ExtractAll(ctx context.Context) ([]types.Product, error) {
    // Implement extraction workflow
    // 1. Get product URLs
    // 2. Extract size charts for each product
    // 3. Return structured results
}

// ExtractToJSON extracts and saves to JSON file
func (n *NewStoreExtractor) ExtractToJSON(ctx context.Context, filename string) error {
    results, err := n.ExtractAll(ctx)
    if err != nil {
        return err
    }

    jsonData, err := json.MarshalIndent(results, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal results to JSON: %w", err)
    }

    return os.WriteFile(filename, jsonData, 0644)
}

// Close cleans up resources
func (n *NewStoreExtractor) Close() {
    if n.adapter != nil {
        n.adapter.Close()
    }
}
```

### Step 3: Update CLI and API

#### Update CLI (`cmd/main.go`):

```go
// Add to the switch statement
case "newstore":
    extractor := extractor.NewNewStoreExtractor(config, logger)
    defer extractor.Close()
    
    if len(args) > 1 {
        filename = args[1]
    } else {
        filename = "results_newstore.json"
    }
    
    err = extractor.ExtractToJSON(ctx, filename)
```

#### Update API (`cmd/api/main.go`):

```go
// Add to the store mapping
storeExtractors := map[string]func(*types.Config, types.Logger) types.StoreExtractor{
    "westside.com":      func(c *types.Config, l types.Logger) types.StoreExtractor { return extractor.NewWestsideExtractor(c, l) },
    "littleboxindia.com": func(c *types.Config, l types.Logger) types.StoreExtractor { return extractor.NewLittleBoxIndiaExtractor(c, l) },
    "suqah.com":         func(c *types.Config, l types.Logger) types.StoreExtractor { return extractor.NewSuqahExtractor(c, l) },
    "newstore.com":      func(c *types.Config, l types.Logger) types.StoreExtractor { return extractor.NewNewStoreExtractor(c, l) },
}
```

### Step 4: Add Tests

Create `adapters/newstore_test.go`:

```go
package adapters

import (
    "testing"
    "shopify-extractor/internal/types"
)

func TestNewStoreAdapter_GetStoreName(t *testing.T) {
    adapter := NewNewStoreAdapter(&types.Config{}, &MockLogger{})
    expected := "newstore.com"
    if got := adapter.GetStoreName(); got != expected {
        t.Errorf("GetStoreName() = %v, want %v", got, expected)
    }
}

func TestNewStoreAdapter_GetProductURLs(t *testing.T) {
    // Add tests for product URL extraction
}

func TestNewStoreAdapter_ExtractSizeChart(t *testing.T) {
    // Add tests for size chart extraction
}
```

## Development Workflow

### 1. Feature Development

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/add-new-store
   ```

2. **Make your changes**:
   - Implement the new store adapter
   - Add tests
   - Update documentation

3. **Test your changes**:
   ```bash
   go test ./...
   go run cmd/main.go newstore
   ```

4. **Commit your changes**:
   ```bash
   git add .
   git commit -m "Add support for newstore.com"
   ```

### 2. Testing

#### Unit Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./adapters -v

# Run with coverage
go test -cover ./...
```

#### Integration Tests

```bash
# Test CLI
go run cmd/main.go westside

# Test API
go run cmd/api/main.go &
curl -X POST http://localhost:8080/extract -H "Content-Type: application/json" -d '{"stores": ["westside.com"]}'
```

### 3. Code Quality

#### Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

#### Formatting

```bash
# Format code
go fmt ./...

# Organize imports
goimports -w .
```

## Common Patterns

### 1. Error Handling

```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("failed to extract size chart: %w", err)
}

// Use custom error types for specific cases
type SizeChartNotFoundError struct {
    URL string
}

func (e SizeChartNotFoundError) Error() string {
    return fmt.Sprintf("size chart not found for URL: %s", e.URL)
}
```

### 2. Logging

```go
// Use structured logging
w.logger.Infof("Processing collection %d/%d: %s", i+1, total, collectionURL)
w.logger.Debugf("Found %d products in collection", len(products))
w.logger.Warnf("Failed to extract from %s: %v", productURL, err)
```

### 3. Configuration

```go
// Use environment variables for configuration
port := os.Getenv("API_PORT")
if port == "" {
    port = "8080"
}

// Use struct-based configuration
type Config struct {
    UseHeadlessBrowser bool
    HTTPTimeout        time.Duration
    MaxRetries         int
}
```

### 4. Testing

```go
// Use table-driven tests
func TestExtractSizeChart(t *testing.T) {
    tests := []struct {
        name    string
        url     string
        want    *types.SizeChart
        wantErr bool
    }{
        {
            name: "valid size chart",
            url:  "https://example.com/product",
            want: &types.SizeChart{
                Headers: []string{"Size", "Bust (in)"},
                Rows:    []map[string]string{{"Size": "S", "Bust (in)": "34"}},
            },
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Debugging

### 1. Enable Debug Logging

```bash
export LOG_LEVEL=debug
go run cmd/api/main.go
```

### 2. Use Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug with delve
dlv debug cmd/api/main.go
```

### 3. Profile Performance

```bash
# CPU profiling
go test -cpuprofile=cpu.prof ./...

# Memory profiling
go test -memprofile=mem.prof ./...

# Analyze profiles
go tool pprof cpu.prof
```

## Best Practices

### 1. Code Organization

- Keep functions small and focused
- Use meaningful variable and function names
- Group related functionality together
- Follow Go naming conventions

### 2. Error Handling

- Always check for errors
- Provide meaningful error messages
- Use error wrapping for context
- Don't ignore errors

### 3. Performance

- Minimize HTTP requests
- Use efficient selectors
- Cache page content when possible
- Implement rate limiting

### 4. Security

- Validate all inputs
- Sanitize URLs
- Use HTTPS when possible
- Don't expose sensitive information

### 5. Testing

- Write tests for all new functionality
- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Test error conditions

## Troubleshooting

### Common Issues

1. **Import errors**: Run `go mod tidy`
2. **Test failures**: Check test data and mocks
3. **Build errors**: Verify Go version and dependencies
4. **Runtime errors**: Check logs and debug output

### Getting Help

1. Check existing documentation
2. Review similar implementations
3. Search existing issues
4. Create a new issue with details

## Contributing Guidelines

### 1. Code Review Process

1. Create a pull request
2. Ensure all tests pass
3. Update documentation
4. Request review from maintainers

### 2. Commit Message Format

```
type(scope): description

[optional body]

[optional footer]
```

Examples:
- `feat(westside): add support for new size chart format`
- `fix(api): handle empty store list in request`
- `docs(readme): update installation instructions`

### 3. Pull Request Checklist

- [ ] Tests pass
- [ ] Code is formatted
- [ ] Documentation is updated
- [ ] No linting errors
- [ ] Follows project conventions 