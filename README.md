# Shopify Size Chart Extractor

A Go-based web scraping tool that extracts size charts from Shopify stores. The tool supports multiple stores including Westside, LittleBoxIndia, and Suqah, providing structured JSON output with size measurements in both inches and centimeters.

## Features

- **Multi-store Support**: Extracts size charts from Westside, LittleBoxIndia, and Suqah
- **Dual Unit Output**: Provides measurements in both inches and centimeters
- **REST API**: HTTP API for programmatic access
- **CLI Interface**: Command-line interface for direct usage
- **Modular Architecture**: Separate adapters for each store
- **Performance Optimized**: Caches page content and minimizes HTTP requests
- **Structured Output**: Clean JSON format with headers and measurement rows

## Prerequisites

- **Go 1.19+**: Required for building and running the application
- **Chrome/Chromium**: Required for headless browser automation (for Westside)
- **Internet Connection**: Required for web scraping

## Installation

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd shopify_extractor
   ```

2. **Install Go dependencies**:
   ```bash
   go mod download
   ```

3. **Verify installation**:
   ```bash
   go version
   go mod verify
   ```

## Configuration

### Environment Variables

Create a `.env` file in the project root (optional):

```env
# API Configuration
API_PORT=8080

# Browser Configuration (for Westside)
USE_HEADLESS_BROWSER=true
BROWSER_TIMEOUT=30s

# HTTP Configuration
HTTP_TIMEOUT=30s
USER_AGENT=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36
```

### Store-Specific Configuration

Each store adapter can be configured independently:

- **Westside**: Always uses headless browser for dynamic content
- **LittleBoxIndia**: Uses standard HTTP requests
- **Suqah**: Uses standard HTTP requests

## Usage

### 1. REST API Server

Start the API server:

```bash
# Using Makefile
make run-api

# Or directly
go run cmd/api/main.go
```

The server will start on port 8080 (or the port specified in `API_PORT` environment variable).

#### API Endpoints

**Health Check**:
```bash
curl http://localhost:8080/health
```

**Extract Size Charts**:
```bash
curl -X POST http://localhost:8080/extract \
  -H "Content-Type: application/json" \
  -d '{"stores": ["westside.com", "littleboxindia.com", "suqah.com"]}'
```

**Single Store Extraction**:
```bash
curl -X POST http://localhost:8080/extract \
  -H "Content-Type: application/json" \
  -d '{"stores": ["westside.com"]}'
```

### 2. Command Line Interface

**Extract from all stores**:
```bash
go run cmd/main.go
```

**Extract from specific store**:
```bash
# Westside only
go run cmd/main.go westside

# LittleBoxIndia only  
go run cmd/main.go littleboxindia

# Suqah only
go run cmd/main.go suqah
```

**Save to specific file**:
```bash
go run cmd/main.go westside results_westside.json
```

### 3. Individual Store Extractors

**Westside**:
```bash
go run extractor/westside_extractor.go
```

**LittleBoxIndia**:
```bash
go run extractor/littleboxindia_extractor.go
```

**Suqah**:
```bash
go run extractor/suqah_extractor.go
```

## Output Format

The tool outputs structured JSON with the following format:

```json
{
  "success": true,
  "data": {
    "stores": [
      {
        "store_name": "westside.com",
        "products": [
          {
            "product_title": "Wardrobe Off-White Stripe Printed Top",
            "product_url": "https://www.westside.com/products/...",
            "size_chart": [
              {
                "headers": ["Size", "Bust (in)", "Waist (in)", "Hip (in)"],
                "rows": [
                  {
                    "Size": "XXS",
                    "Bust (in)": "31",
                    "Waist (in)": "24",
                    "Hip (in)": "34"
                  }
                ]
              },
              {
                "headers": ["Size", "Bust (cm)", "Waist (cm)", "Hip (cm)"],
                "rows": [
                  {
                    "Size": "XXS",
                    "Bust (cm)": "78",
                    "Waist (cm)": "62",
                    "Hip (cm)": "86"
                  }
                ]
              }
            ]
          }
        ]
      }
    ]
  }
}
```

## Project Structure

```
shopify_extractor/
├── adapters/                 # Store-specific adapters
│   ├── base.go              # Base adapter with common functionality
│   ├── westside.go          # Westside store adapter
│   ├── littleboxindia.go    # LittleBoxIndia store adapter
│   └── suqah.go             # Suqah store adapter
├── cmd/                     # Command-line interfaces
│   ├── main.go              # Main CLI application
│   └── api/                 # API server
│       └── main.go          # API server entry point
├── extractor/               # Store-specific extractors
│   ├── westside_extractor.go
│   ├── littleboxindia_extractor.go
│   └── suqah_extractor.go
├── internal/                # Internal packages
│   └── types/               # Type definitions
│       └── types.go
├── utils/                   # Utility functions
│   ├── browser.go           # Browser automation utilities
│   └── http.go              # HTTP utilities
├── docs/                    # Documentation
│   ├── ARCHITECTURE.md      # Technical architecture details
│   ├── DEVELOPMENT.md       # Development guide
│   └── COST_ANALYSIS.md     # Scaling and cost analysis
├── go.mod                   # Go module file
├── go.sum                   # Go module checksums
├── Makefile                 # Build and run commands
└── README.md                # This file
```

## Development

### Building

```bash
# Build all binaries
make build

# Build specific binary
make build-api
make build-cli
```

### Testing

```bash
# Run all tests
go test ./...

# Run specific test
go test ./adapters -v
go test ./extractor -v
```

### Running in Development

```bash
# Start API server in background
make run-api &

# Test API
curl http://localhost:8080/health

# Stop server
pkill -f "go run cmd/api/main.go"
```

## Troubleshooting

### Common Issues

1. **Port already in use**:
   ```bash
   # Kill process on port 8080
   lsof -ti:8080 | xargs kill -9
   ```

2. **Chrome not found** (for Westside):
   ```bash
   # Install Chrome on macOS
   brew install --cask google-chrome
   
   # Install Chrome on Ubuntu
   sudo apt-get install google-chrome-stable
   ```

3. **Permission denied**:
   ```bash
   # Make scripts executable
   chmod +x test_api.sh
   ```

4. **Slow extraction**:
   - Check internet connection
   - Verify target websites are accessible
   - Consider reducing the number of collections processed

### Debug Mode

Enable debug logging by setting the log level:

```bash
export LOG_LEVEL=debug
go run cmd/api/main.go
```

## Performance Considerations

- **Collection Limits**: The tool processes a limited number of collections by default to avoid overwhelming target servers
- **Rate Limiting**: Built-in delays between requests to be respectful to target websites
- **Caching**: Page content is cached to minimize duplicate requests
- **Parallel Processing**: Future versions may support concurrent extraction

## Scaling and Cost Analysis

For detailed information about scaling the service and cost optimization strategies, see:

- **[Cost Analysis](docs/COST_ANALYSIS.md)**: Comprehensive cost projections and optimization strategies
- **[Architecture Guide](docs/ARCHITECTURE.md)**: Technical details for scaling decisions
- **[Development Guide](docs/DEVELOPMENT.md)**: How to implement scaling improvements

### Quick Cost Overview
- **1,000 stores**: ~$300-600/month (optimized)
- **10,000 stores**: ~$3,000-6,000/month (optimized)
- **100,000 stores**: ~$30,000-60,000/month (optimized)

*Note: Costs include proxy services, compute resources, storage, and infrastructure. See the full cost analysis for detailed breakdowns.*

## Legal and Ethical Considerations

- **Respect robots.txt**: The tool respects website robots.txt files
- **Rate Limiting**: Built-in delays prevent overwhelming target servers
- **Terms of Service**: Ensure compliance with target website terms of service
- **Data Usage**: Use extracted data responsibly and in accordance with applicable laws

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions:
1. Check the troubleshooting section
2. Review existing issues
3. Create a new issue with detailed information

## Changelog

### v1.0.0
- Initial release
- Support for Westside, LittleBoxIndia, and Suqah
- REST API and CLI interfaces
- Dual unit output (inches and centimeters)
- Modular adapter architecture