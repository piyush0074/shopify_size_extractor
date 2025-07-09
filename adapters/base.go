package adapters

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"shopify-extractor/internal/types"
	"shopify-extractor/utils"

	"github.com/PuerkitoBio/goquery"
)

// BaseAdapter provides common functionality for store adapters.
// It implements the Template Method pattern, providing a foundation
// that store-specific adapters can extend and customize.
type BaseAdapter struct {
	config        *types.Config  // Configuration settings (timeouts, browser settings, etc.)
	logger        types.Logger   // Structured logging interface
	httpClient    *utils.HTTPClient    // HTTP client for standard requests
	browserClient *utils.BrowserClient // Headless browser client for dynamic content
}

// NewBaseAdapter creates a new base adapter with initialized HTTP and browser clients.
// This is the factory method that sets up the common infrastructure used by all store adapters.
func NewBaseAdapter(config *types.Config, logger types.Logger) *BaseAdapter {
	return &BaseAdapter{
		config:        config,
		logger:        logger,
		httpClient:    utils.NewHTTPClient(config, logger),
		browserClient: utils.NewBrowserClient(config, logger),
	}
}

// GetPageContent retrieves the HTML content of a page using either HTTP client or headless browser.
// The choice between HTTP and browser is determined by the UseHeadlessBrowser configuration.
// This method is used by all store adapters to fetch page content for parsing.
func (b *BaseAdapter) GetPageContent(ctx context.Context, url string) (string, error) {
	// Use headless browser for JavaScript-heavy sites (like Westside)
	if b.config.UseHeadlessBrowser {
		return b.browserClient.GetPageContent(ctx, url)
	}

	// Use standard HTTP client for static content (faster and more efficient)
	body, err := b.httpClient.Get(ctx, url)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// ParseHTML parses HTML content into a goquery document
func (b *BaseAdapter) ParseHTML(html string) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(strings.NewReader(html))
}

// ExtractTableData extracts table data from a goquery document using CSS selectors.
// This is a generic table parser that can handle various HTML table structures.
// It extracts both headers and data rows, returning a structured SizeChart object.
func (b *BaseAdapter) ExtractTableData(doc *goquery.Document, tableSelector string) (*types.SizeChart, error) {
	// Find the table using the provided CSS selector
	table := doc.Find(tableSelector)
	if table.Length() == 0 {
		return nil, fmt.Errorf("table not found with selector: %s", tableSelector)
	}

	// Extract headers from the first row or thead section
	// Supports multiple header formats: thead > tr > th, tr:first-child > th/td
	var headers []string
	table.Find("thead tr th, tr:first-child th, tr:first-child td").Each(func(i int, s *goquery.Selection) {
		headers = append(headers, strings.TrimSpace(s.Text()))
	})

	if len(headers) == 0 {
		return nil, fmt.Errorf("no headers found in table")
	}

	// Extract data rows from tbody or all rows except the first
	// Maps each cell to its corresponding header
	var rows []map[string]string
	table.Find("tbody tr, tr:not(:first-child)").Each(func(i int, s *goquery.Selection) {
		row := make(map[string]string)
		s.Find("td, th").Each(func(j int, cell *goquery.Selection) {
			if j < len(headers) {
				row[headers[j]] = strings.TrimSpace(cell.Text())
			}
		})

		// Only add rows that have actual data (not empty rows)
		if len(row) > 0 {
			rows = append(rows, row)
		}
	})

	if len(rows) == 0 {
		return nil, fmt.Errorf("no data rows found in table")
	}

	return &types.SizeChart{
		Headers: headers,
		Rows:    rows,
	}, nil
}

// ExtractText extracts text from an element using a CSS selector
func (b *BaseAdapter) ExtractText(doc *goquery.Document, selector string) (string, error) {
	element := doc.Find(selector)
	if element.Length() == 0 {
		return "", fmt.Errorf("element not found with selector: %s", selector)
	}

	return strings.TrimSpace(element.Text()), nil
}

// ExtractAttribute extracts an attribute value from an element
func (b *BaseAdapter) ExtractAttribute(doc *goquery.Document, selector string, attribute string) (string, error) {
	element := doc.Find(selector)
	if element.Length() == 0 {
		return "", fmt.Errorf("element not found with selector: %s", selector)
	}

	value, exists := element.Attr(attribute)
	if !exists {
		return "", fmt.Errorf("attribute %s not found on element %s", attribute, selector)
	}

	return value, nil
}

// Close cleans up resources
func (b *BaseAdapter) Close() {
	if b.httpClient != nil {
		b.httpClient.Close()
	}
}

// FilterSizeChart normalizes and filters size chart data to a standard format.
// This method handles the complexity of different stores using various header names
// and formats, converting them to a consistent output format with canonical headers.
//
// The method performs several key operations:
// 1. Maps various header names to canonical output headers
// 2. Filters out irrelevant columns (keeping only Size, Bust, Waist, Hip)
// 3. Normalizes data to ensure consistent structure
// 4. Filters out empty rows to maintain data quality
func (b *BaseAdapter) FilterSizeChart(sizeChart *types.SizeChart) *types.SizeChart {
	if sizeChart == nil {
		return nil
	}

	// Define the canonical output headers that all stores should produce
	// This ensures consistent JSON output across different stores
	outputHeaders := []string{"Size", "Bust (in)", "Waist (in)", "Hip (in)"}

	// Map various possible header names to canonical output headers
	// This handles the fact that different stores use different naming conventions
	// e.g., "BUST", "Bust Size", "Chest" all map to "Bust (in)"
	headerMap := map[string]string{
		"size":  "Size",
		"bust":  "Bust (in)",
		"waist": "Waist (in)",
		"hip":   "Hip (in)",
		"hips":  "Hip (in)", // Handle both singular and plural forms
	}

	// Create a mapping from input headers to canonical output headers
	// This allows us to know which input column corresponds to which output column
	inputToOutput := make(map[string]string) // input header -> output header
	for _, h := range sizeChart.Headers {
		lower := strings.ToLower(h)
		for key, canon := range headerMap {
			if strings.Contains(lower, key) {
				inputToOutput[h] = canon
				break
			}
		}
	}

	// Debug logging to help troubleshoot header mapping issues
	fmt.Printf("Processing headers: %v\n", sizeChart.Headers)
	fmt.Printf("Input to output mapping: %v\n", inputToOutput)

	// If no relevant headers found (Bust/Waist/Hip/Size), return nil
	// This prevents processing tables that aren't actually size charts
	if len(inputToOutput) == 0 {
		return nil
	}

	// Build filtered rows by mapping input data to canonical output format
	var filteredRows []map[string]string
	for _, row := range sizeChart.Rows {
		filteredRow := make(map[string]string)
		
		// For each canonical output header, find the corresponding input data
		for _, outHeader := range outputHeaders {
			found := false
			// Look through the input-to-output mapping to find the right data
			for inHeader, out := range inputToOutput {
				if out == outHeader {
					if val, ok := row[inHeader]; ok {
						filteredRow[outHeader] = val
						found = true
						break
					}
				}
			}
			// If no data found for this header, use empty string
			if !found {
				filteredRow[outHeader] = ""
			}
		}
		
		// Only add rows that have at least one measurement value
		// This filters out completely empty rows or rows with only size labels
		if filteredRow["Bust (in)"] != "" || filteredRow["Waist (in)"] != "" || filteredRow["Hip (in)"] != "" {
			filteredRows = append(filteredRows, filteredRow)
		}
	}

	return &types.SizeChart{
		Headers: outputHeaders,
		Rows:    filteredRows,
	}
}

// IsValidSizeChart checks if the extracted data looks like a valid size chart
// This is a shared utility that can be used by all adapters
func (b *BaseAdapter) IsValidSizeChart(sizeChart *types.SizeChart) bool {
	if sizeChart == nil || len(sizeChart.Headers) == 0 || len(sizeChart.Rows) == 0 {
		return false
	}

	// Check if headers contain size-related keywords
	sizeKeywords := []string{"size", "bust", "waist", "hip", "chest", "length", "width"}
	headerText := strings.ToLower(strings.Join(sizeChart.Headers, " "))

	for _, keyword := range sizeKeywords {
		if strings.Contains(headerText, keyword) {
			return true
		}
	}

	// Check if rows contain size-related data
	for _, row := range sizeChart.Rows {
		for _, value := range row {
			// Look for size indicators like S, M, L, XS, XL, or numbers
			if strings.Contains(strings.ToUpper(value), "S") ||
				strings.Contains(strings.ToUpper(value), "M") ||
				strings.Contains(strings.ToUpper(value), "L") ||
				strings.Contains(strings.ToUpper(value), "X") {
				return true
			}
		}
	}

	return false
}

// RemoveDuplicateURLs removes duplicate URLs from the slice
// This is a shared utility that can be used by all adapters
func (b *BaseAdapter) RemoveDuplicateURLs(urls []string) []string {
	seen := make(map[string]bool)
	var uniqueURLs []string

	for _, url := range urls {
		if !seen[url] {
			seen[url] = true
			uniqueURLs = append(uniqueURLs, url)
		}
	}

	return uniqueURLs
}

// ExtractCollectionURLs finds all collection URLs from the products page
// This is a shared utility that can be used by all adapters
func (b *BaseAdapter) ExtractCollectionURLs(doc *goquery.Document, baseURL string) ([]string, error) {
	var collectionURLs []string

	// Find all <a> tags that contain "collections" in their href
	doc.Find("a[href*='collections']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// Clean and normalize the URL
		href = strings.TrimSpace(href)
		if href == "" {
			return
		}

		// Convert relative URLs to absolute URLs
		if strings.HasPrefix(href, "/") {
			href = baseURL + href
		} else if !strings.HasPrefix(href, "http") {
			href = baseURL + "/" + href
		}

		// Validate URL
		if _, err := url.Parse(href); err == nil {
			collectionURLs = append(collectionURLs, href)
		}
	})

	if len(collectionURLs) == 0 {
		return nil, fmt.Errorf("no collection URLs found")
	}

	return collectionURLs, nil
}

// ExtractProductURLsFromCollection extracts product URLs from a collection page
// This is a shared utility that can be used by all adapters
func (b *BaseAdapter) ExtractProductURLsFromCollection(doc *goquery.Document, baseURL string) ([]string, error) {
	var productURLs []string

	// Find all <a> tags that contain "/products/" in their href
	doc.Find("a[href*='/products/']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// Clean and normalize the URL
		href = strings.TrimSpace(href)
		if href == "" {
			return
		}

		// Convert relative URLs to absolute URLs
		if strings.HasPrefix(href, "/") {
			href = baseURL + href
		} else if !strings.HasPrefix(href, "http") {
			href = baseURL + "/" + href
		}

		// Validate URL
		if _, err := url.Parse(href); err == nil {
			productURLs = append(productURLs, href)
		}
	})

	return productURLs, nil
}

// ExtractProductTitleFromDoc extracts the product title from an already parsed document
// This is a shared utility that can be used by all adapters
func (b *BaseAdapter) ExtractProductTitleFromDoc(doc *goquery.Document) (string, error) {
	// Try different selectors for product title
	selectors := []string{
		"h1.product-title",
		"h1[class*='title']",
		".product-name h1",
		".product-info h1",
		".product-details h1",
		"h1",
	}

	for _, selector := range selectors {
		title, err := b.ExtractText(doc, selector)
		if err == nil && title != "" {
			return title, nil
		}
	}

	return "", fmt.Errorf("product title not found on page")
}

// Config returns the config field of the BaseAdapter
func (b *BaseAdapter) Config() *types.Config {
	return b.config
}
