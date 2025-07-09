# Architecture Documentation

## Overview

The Shopify Size Chart Extractor is built with a modular, adapter-based architecture that allows for easy extension to new stores while maintaining clean separation of concerns.

## Design Principles

1. **Separation of Concerns**: Each store has its own adapter and extractor
2. **DRY (Don't Repeat Yourself)**: Common functionality is shared through base classes
3. **Single Responsibility**: Each component has a single, well-defined purpose
4. **Extensibility**: Easy to add new stores without modifying existing code
5. **Performance**: Optimized for speed while being respectful to target servers

## Architecture Components

### 1. Adapter Layer (`adapters/`)

The adapter layer provides store-specific implementations for web scraping. Each adapter implements the same interface but handles the unique characteristics of each store.

#### Base Adapter (`adapters/base.go`)

**Purpose**: Provides common functionality shared across all store adapters.

**Key Features**:
- HTTP client management with timeouts and retries
- HTML parsing utilities using goquery
- Browser automation for JavaScript-heavy sites
- URL validation and normalization
- Common text extraction methods

**Design Decisions**:
- Uses `goquery` for HTML parsing (similar to jQuery syntax)
- Implements headless browser support via Chrome DevTools Protocol
- Provides fallback mechanisms for failed requests
- Includes rate limiting to be respectful to target servers

#### Store-Specific Adapters

Each store adapter extends the base adapter and implements store-specific logic:

**Westside Adapter** (`adapters/westside.go`):
- Uses headless browser due to dynamic content loading
- Extracts size charts from `.sizeguide` containers
- Handles dual-unit measurements (inches and centimeters)
- Processes multiple collections to find products

**LittleBoxIndia Adapter** (`adapters/littleboxindia.go`):
- Uses standard HTTP requests (faster than browser)
- Extracts from `.ks-table` containers
- Handles JSON-based measurement data
- Supports both inches and centimeters

**Suqah Adapter** (`adapters/suqah.go`):
- Uses standard HTTP requests
- Multiple selector fallbacks for size chart detection
- Custom size inference logic
- Handles various table formats

### 2. Extractor Layer (`extractor/`)

The extractor layer orchestrates the extraction process and provides high-level interfaces.

#### Individual Store Extractors

Each store has its own extractor that:
- Manages the extraction workflow
- Handles product discovery
- Coordinates between adapters
- Provides JSON output formatting

**Key Methods**:
- `ExtractAll()`: Main extraction method
- `ExtractToJSON()`: Saves results to file
- `Close()`: Cleanup resources

### 3. API Layer (`cmd/api/`)

**Purpose**: Provides HTTP API for programmatic access.

**Endpoints**:
- `GET /health`: Health check endpoint
- `POST /extract`: Main extraction endpoint

**Design Decisions**:
- Uses standard `net/http` package
- JSON request/response format
- Supports multiple stores in single request
- Includes proper error handling and status codes

### 4. CLI Layer (`cmd/main.go`)

**Purpose**: Provides command-line interface for direct usage.

**Features**:
- Support for individual store extraction
- Output file specification
- Help and usage information

## Data Flow

### 1. Product Discovery Flow

```
1. Get products page URL
2. Extract collection URLs
3. For each collection:
   a. Fetch collection page
   b. Extract product URLs
   c. Validate and normalize URLs
4. Remove duplicates
5. Return unique product URLs
```

### 2. Size Chart Extraction Flow

```
1. For each product URL:
   a. Fetch product page (once)
   b. Extract product title
   c. Extract size chart data
   d. Parse and structure data
   e. Create dual-unit charts (inches/cm)
2. Return structured results
```

### 3. API Request Flow

```
1. Receive JSON request with store list
2. For each store:
   a. Create appropriate adapter
   b. Run extraction process
   c. Collect results
3. Combine all results
4. Return JSON response
```

## Key Design Patterns

### 1. Adapter Pattern

Each store implements the same interface but handles store-specific details:

```go
type StoreAdapter interface {
    GetStoreName() string
    GetProductURLs(ctx types.Context) ([]string, error)
    ExtractSizeChart(ctx types.Context, productURL string) (*types.SizeChart, error)
    GetProductTitle(ctx types.Context, productURL string) (string, error)
}
```

### 2. Template Method Pattern

The base adapter provides a template for common operations while allowing subclasses to override specific steps.

### 3. Strategy Pattern

Different extraction strategies are used based on the store:
- Headless browser for dynamic content (Westside)
- Standard HTTP for static content (LittleBoxIndia, Suqah)

## Performance Optimizations

### 1. Page Content Caching

- Fetch product page once and reuse for title and size chart extraction
- Avoid duplicate HTTP requests within the same extraction

### 2. Collection Processing Limits

- Process limited number of collections to avoid overwhelming servers
- Configurable limits for testing vs production

### 3. Efficient Selectors

- Use specific CSS selectors for faster extraction
- Fallback to broader selectors when specific ones fail

### 4. Rate Limiting

- Built-in delays between requests
- Respectful to target server resources

## Error Handling Strategy

### 1. Graceful Degradation

- Continue processing other products if one fails
- Log warnings instead of failing completely
- Return partial results when possible

### 2. Retry Logic

- Automatic retries for transient failures
- Exponential backoff for repeated failures
- Maximum retry limits to prevent infinite loops

### 3. Validation

- URL validation before processing
- Data validation after extraction
- Fallback mechanisms for missing data

## Configuration Management

### 1. Environment Variables

- `API_PORT`: Server port configuration
- `USE_HEADLESS_BROWSER`: Browser automation toggle
- `HTTP_TIMEOUT`: Request timeout settings

### 2. Store-Specific Configuration

Each adapter can have its own configuration:
- Selector preferences
- Processing limits
- Timeout settings

## Testing Strategy

### 1. Unit Tests

- Individual adapter methods
- Utility functions
- Data parsing logic

### 2. Integration Tests

- End-to-end extraction workflows
- API endpoint testing
- CLI command testing

### 3. Mock Testing

- Mock HTTP responses for consistent testing
- Mock browser automation for faster tests

## Security Considerations

### 1. Input Validation

- Validate store names before processing
- Sanitize URLs to prevent injection attacks
- Validate JSON input in API endpoints

### 2. Rate Limiting

- Prevent abuse through request throttling
- Respect target server resources

### 3. Error Information

- Don't expose internal errors to clients
- Log detailed errors for debugging
- Return generic error messages to users

## Future Enhancements

### 1. Concurrent Processing

- Process multiple products simultaneously
- Use goroutines for parallel extraction
- Implement worker pools for controlled concurrency

### 2. Caching Layer

- Cache extracted data to avoid re-scraping
- Implement TTL-based cache invalidation
- Support for persistent storage

### 3. Monitoring and Metrics

- Request/response timing metrics
- Success/failure rate tracking
- Performance monitoring

### 4. Plugin Architecture

- Dynamic loading of store adapters
- Configuration-driven adapter selection
- Hot-reloading of adapter code

## Assumptions and Limitations

### 1. Website Structure Assumptions

- Size charts are typically in HTML tables
- Product URLs follow predictable patterns
- Collections are accessible via standard URLs

### 2. Performance Limitations

- Single-threaded processing (can be enhanced)
- No persistent caching
- Limited to HTTP/HTTPS protocols

### 3. Reliability Considerations

- Dependent on target website availability
- Subject to website structure changes
- May be blocked by anti-bot measures

## Code Quality Standards

### 1. Documentation

- All public methods have godoc comments
- Complex logic includes inline comments
- Architecture decisions are documented

### 2. Error Handling

- All errors are properly handled and logged
- Meaningful error messages for debugging
- Graceful degradation when possible

### 3. Testing

- High test coverage for critical paths
- Integration tests for end-to-end workflows
- Mock tests for external dependencies 