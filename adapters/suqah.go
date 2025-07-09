package adapters

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"shopify-extractor/internal/types"

	"github.com/PuerkitoBio/goquery"
)

// SuqahAdapter handles extraction for suqah.com
type SuqahAdapter struct {
	*BaseAdapter
}

// NewSuqahAdapter creates a new Suqah adapter
func NewSuqahAdapter(config *types.Config, logger types.Logger) *SuqahAdapter {
	config.UseHeadlessBrowser = true // Always use browser for Suqah
	return &SuqahAdapter{
		BaseAdapter: NewBaseAdapter(config, logger),
	}
}

// GetStoreName returns the store name
func (s *SuqahAdapter) GetStoreName() string {
	return "suqah.com"
}

// GetProductURLs returns a list of product URLs for Suqah
func (s *SuqahAdapter) GetProductURLs(ctx types.Context) ([]string, error) {
	s.logger.Info("Starting product discovery for Suqah")

	// Step 1: Get the products page
	productsPageURL := "https://www.suqah.com/products"
	s.logger.Debugf("Fetching products page: %s", productsPageURL)

	html, err := s.GetPageContent(context.Background(), productsPageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get products page: %w", err)
	}

	doc, err := s.ParseHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to parse products page: %w", err)
	}

	// Step 2: Find all collection URLs
	collectionURLs, err := s.ExtractCollectionURLs(doc, "https://www.suqah.com")
	if err != nil {
		return nil, fmt.Errorf("failed to extract collection URLs: %w", err)
	}

	s.logger.Infof("Found %d collections", len(collectionURLs))

	// Step 3: Iterate through collections to find product URLs
	var allProductURLs []string
	for i, collectionURL := range collectionURLs {
		s.logger.Debugf("Processing collection: %s %d", collectionURL, i+1)

		productURLs, err := s.extractProductURLsFromCollection(collectionURL)
		if err != nil {
			s.logger.Warnf("Failed to extract products from collection %s: %v", collectionURL, err)
			continue
		}

		allProductURLs = append(allProductURLs, productURLs...)
		s.logger.Debugf("Found %d products in collection %s", len(productURLs), collectionURL)
		// Process only first few collections for speed testing
		// if i >= 4 { // Process first 3 collections only
		// 	break
		// }
	}

	// Remove duplicates
	uniqueProductURLs := s.RemoveDuplicateURLs(allProductURLs)

	s.logger.Infof("Total unique products found: %d", len(uniqueProductURLs))
	return uniqueProductURLs, nil
}

// extractProductURLsFromCollection extracts product URLs from a collection page
func (s *SuqahAdapter) extractProductURLsFromCollection(collectionURL string) ([]string, error) {
	s.logger.Debugf("Extracting products from collection: %s", collectionURL)

	// Get the collection page
	html, err := s.GetPageContent(context.Background(), collectionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection page: %w", err)
	}

	doc, err := s.ParseHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to parse collection page: %w", err)
	}

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
			href = "https://www.suqah.com" + href
		} else if !strings.HasPrefix(href, "http") {
			href = "https://www.suqah.com/" + href
		}

		// Validate URL
		if _, err := url.Parse(href); err == nil {
			productURLs = append(productURLs, href)
		}
	})

	return productURLs, nil
}

// ExtractSizeChart extracts the size chart from a Suqah product page
func (s *SuqahAdapter) ExtractSizeChart(ctx types.Context, productURL string) (*types.SizeChart, error) {
	s.logger.Debugf("Extracting size chart from %s", productURL)

	// Get page content
	html, err := s.GetPageContent(context.Background(), productURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get page content: %w", err)
	}

	// Parse HTML
	doc, err := s.ParseHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Look for table tags that contain size-related content
	selectors := []string{
		".chart_block table",
		".chart_block",
		"table",
		".size-chart table",
		".product-size-chart table",
		".size-guide table",
		"table[class*='size']",
		"table[class*='chart']",
		".product-details table",
	}

	for _, selector := range selectors {
		s.logger.Debugf("Trying selector: %s", selector)
		sizeChart, err := s.extractSuqahTableData(doc, selector)
		if err != nil {
			s.logger.Debugf("Selector %s failed: %v", selector, err)
			continue
		}
		if s.IsValidSizeChart(sizeChart) {
			s.logger.Debugf("Successfully extracted size chart using selector: %s", selector)
			filtered := s.FilterSizeChart(sizeChart)
			if filtered != nil && len(filtered.Rows) > 0 {
				return filtered, nil
			}
		} else {
			s.logger.Debugf("Selector %s found table but it's not a valid size chart", selector)
		}
	}

	return nil, fmt.Errorf("no valid size chart found on page")
}

// extractSuqahTableData extracts table data specifically for Suqah's table structure
func (s *SuqahAdapter) extractSuqahTableData(doc *goquery.Document, tableSelector string) (*types.SizeChart, error) {
	table := doc.Find(tableSelector)
	if table.Length() == 0 {
		return nil, fmt.Errorf("table not found with selector: %s", tableSelector)
	}

	s.logger.Debugf("Found %d elements with selector: %s", table.Length(), tableSelector)

	// If we found a chart_block container, look for tables inside it
	if strings.Contains(tableSelector, "chart_block") && !strings.Contains(tableSelector, "table") {
		s.logger.Debugf("Found chart_block container, looking for tables inside")
		table = table.Find("table")
		if table.Length() == 0 {
			return nil, fmt.Errorf("no table found inside chart_block")
		}
		s.logger.Debugf("Found %d tables inside chart_block", table.Length())
	}

	// Extract headers from the first row
	var headers []string
	firstRow := table.Find("tr").First()
	firstRow.Find("td, th").Each(func(i int, s *goquery.Selection) {
		headers = append(headers, strings.TrimSpace(s.Text()))
	})

	if len(headers) == 0 {
		return nil, fmt.Errorf("no headers found in table")
	}

	s.logger.Debugf("Original headers from table: %v", headers)

	// Add "Size" header if it doesn't exist
	hasSizeHeader := false
	for _, h := range headers {
		if strings.Contains(strings.ToLower(h), "size") {
			hasSizeHeader = true
			break
		}
	}
	if !hasSizeHeader {
		headers = append([]string{"Size"}, headers...)
	}

	s.logger.Debugf("Final headers after adding Size: %v", headers)

	// Extract rows, but filter out header-like rows
	var rows []map[string]string
	rowIndex := 0
	table.Find("tr").Each(func(i int, row *goquery.Selection) {
		if i == 0 {
			return // Skip the first row (headers)
		}

		rowData := make(map[string]string)
		var firstColumnValue string
		var cellValues []string

		row.Find("td, th").Each(func(j int, cell *goquery.Selection) {
			value := strings.TrimSpace(cell.Text())
			cellValues = append(cellValues, value)

			if j == 0 {
				// Store the first column value separately
				firstColumnValue = value
			}
		})

		// Map columns correctly based on whether we added Size header
		if !hasSizeHeader {
			// We added "Size" header, so map columns with offset
			// Original headers were: [HIPS, BUST, WAIST]
			// New headers are: [Size, HIPS, BUST, WAIST]

			// First column becomes Size
			if len(cellValues) > 0 {
				rowData["Size"] = firstColumnValue
			}

			// Map remaining columns to original headers
			if len(cellValues) > 1 {
				rowData["HIPS"] = cellValues[1]
			}
			if len(cellValues) > 2 {
				rowData["BUST"] = cellValues[2]
			}
			if len(cellValues) > 3 {
				rowData["WAIST"] = cellValues[3]
			}
		} else {
			// Size header already existed, map normally
			for j, value := range cellValues {
				if j < len(headers) {
					rowData[headers[j]] = value
				}
			}
		}

		// Only add rows that have meaningful data and are not header-like
		if len(rowData) > 0 && !s.isHeaderRow(rowData) {
			s.logger.Debugf("Row %d passed filtering, adding to results", i)
			// Check if the first column contains a size label
			if firstColumnValue != "" && s.looksLikeSize(firstColumnValue) {
				rowData["Size"] = firstColumnValue
			} else {
				// Generate size label based on row order
				sizeLabel := s.generateSizeLabel(rowIndex)
				rowData["Size"] = sizeLabel
				s.logger.Debugf("Generated size label '%s' for row %d", sizeLabel, rowIndex)
			}

			rows = append(rows, rowData)
			s.logger.Debug("row : ", rows)
			rowIndex++
		} else {
			s.logger.Debugf("Row %d filtered out - len(rowData): %d, isHeaderRow: %v", i, len(rowData), s.isHeaderRow(rowData))
		}
	})

	if len(rows) == 0 {
		return nil, fmt.Errorf("no data rows found in table")
	}

	s.logger.Debugf("Extracted %d rows", len(rows))

	return &types.SizeChart{
		Headers: headers,
		Rows:    rows,
	}, nil
}

// generateSizeLabel generates a size label based on row index
func (s *SuqahAdapter) generateSizeLabel(index int) string {
	// Common size progression
	sizes := []string{"XS", "S", "M", "L", "XL", "2XL", "3XL", "4XL", "5XL", "6XL", "7XL", "8XL"}

	if index < len(sizes) {
		return sizes[index]
	}

	// If we run out of predefined sizes, use numeric
	return fmt.Sprintf("%d", index+1)
}

// looksLikeSize checks if a value looks like a size label
func (s *SuqahAdapter) looksLikeSize(value string) bool {
	if value == "" {
		return false
	}

	upperValue := strings.ToUpper(strings.TrimSpace(value))

	// Common size patterns
	sizePatterns := []string{"XS", "S", "M", "L", "XL", "XXL", "2XL", "3XL", "4XL", "5XL", "6XL", "7XL", "8XL"}
	for _, pattern := range sizePatterns {
		if upperValue == pattern {
			return true
		}
	}

	// Check for numeric sizes (like "6", "8", "10", etc.)
	if len(upperValue) <= 3 && strings.ContainsAny(upperValue, "0123456789") {
		return true
	}

	return false
}

// isHeaderRow checks if a row contains header-like data (like "BUST", "WAIST", "HIPS")
func (s *SuqahAdapter) isHeaderRow(row map[string]string) bool {
	// Count how many values look like headers
	headerCount := 0
	totalValues := 0

	for _, value := range row {
		if value == "" {
			continue
		}
		totalValues++

		upperValue := strings.ToUpper(strings.TrimSpace(value))
		// Check if the value looks like a header (all caps, common measurement terms)
		if upperValue == "BUST" || upperValue == "WAIST" || upperValue == "HIP" ||
			upperValue == "HIPS" || upperValue == "SIZE" || upperValue == "CHEST" {
			headerCount++
		}
	}

	// Only consider it a header row if most values are header-like
	// This prevents filtering out rows that have some measurement values
	if totalValues > 0 && headerCount >= totalValues/2 {
		return true
	}

	return false
}

// GetProductTitle extracts the product title from a Suqah product page
func (s *SuqahAdapter) GetProductTitle(ctx types.Context, productURL string) (string, error) {
	s.logger.Debugf("Extracting product title from %s", productURL)

	// Get page content
	html, err := s.GetPageContent(context.Background(), productURL)
	if err != nil {
		return "", fmt.Errorf("failed to get page content: %w", err)
	}

	// Parse HTML
	doc, err := s.ParseHTML(html)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	return s.GetProductTitleFromDoc(doc)
}

// GetProductTitleFromDoc extracts the product title from an already parsed document
func (s *SuqahAdapter) GetProductTitleFromDoc(doc *goquery.Document) (string, error) {
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
		title, err := s.ExtractText(doc, selector)
		if err == nil && title != "" {
			s.logger.Debugf("Successfully extracted product title using selector: %s", selector)
			return title, nil
		}
	}

	return "", fmt.Errorf("product title not found on page")
}

// ExtractAllSizeCharts extracts all size charts from a Suqah product page
func (s *SuqahAdapter) ExtractAllSizeCharts(ctx types.Context, productURL string) ([]*types.SizeChart, error) {
	s.logger.Debugf("Extracting all size charts from %s", productURL)

	// Get page content once and reuse it
	html, err := s.GetPageContent(context.Background(), productURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get page content: %w", err)
	}

	// Parse HTML once
	doc, err := s.ParseHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract both title and size chart from the same document
	title, _ := s.GetProductTitleFromDoc(doc)
	if title != "" {
		s.logger.Debugf("Extracted title: %s", title)
	}

	// Extract size chart using the cached document
	sizeChart, err := s.extractSizeChartFromDoc(doc, productURL)
	if err != nil {
		return nil, err
	}

	if sizeChart != nil {
		return []*types.SizeChart{sizeChart}, nil
	}
	return nil, fmt.Errorf("no size chart found")
}

// ExtractProductData extracts both title and size charts in a single page fetch
func (s *SuqahAdapter) ExtractProductData(ctx types.Context, productURL string) (string, []*types.SizeChart, error) {
	s.logger.Debugf("Extracting product data from %s", productURL)

	// Get page content once
	html, err := s.GetPageContent(context.Background(), productURL)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get page content: %w", err)
	}

	// Parse HTML once
	doc, err := s.ParseHTML(html)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract title
	title, err := s.GetProductTitleFromDoc(doc)
	if err != nil {
		s.logger.Debugf("Failed to extract title: %v", err)
		title = "Unknown Product"
	}

	// Extract size chart
	sizeChart, err := s.extractSizeChartFromDoc(doc, productURL)
	if err != nil {
		s.logger.Debugf("Failed to extract size chart: %v", err)
	}

	var sizeCharts []*types.SizeChart
	if sizeChart != nil {
		sizeCharts = append(sizeCharts, sizeChart)
	}

	return title, sizeCharts, nil
}

// extractSizeChartFromDoc extracts size chart from an already parsed document
func (s *SuqahAdapter) extractSizeChartFromDoc(doc *goquery.Document, productURL string) (*types.SizeChart, error) {
	s.logger.Debugf("Extracting size chart from document for %s", productURL)

	// Look for table tags that contain size-related content
	selectors := []string{
		".chart_block table",
		".chart_block",
		"table",
		".size-chart table",
		".product-size-chart table",
		".size-guide table",
		"table[class*='size']",
		"table[class*='chart']",
		".product-details table",
	}

	for _, selector := range selectors {
		s.logger.Debugf("Trying selector: %s", selector)
		sizeChart, err := s.extractSuqahTableData(doc, selector)
		if err != nil {
			s.logger.Debugf("Selector %s failed: %v", selector, err)
			continue
		}
		if s.IsValidSizeChart(sizeChart) {
			s.logger.Debugf("Successfully extracted size chart using selector: %s", selector)
			filtered := s.FilterSizeChart(sizeChart)
			if filtered != nil && len(filtered.Rows) > 0 {
				return filtered, nil
			}
		} else {
			s.logger.Debugf("Selector %s found table but it's not a valid size chart", selector)
		}
	}

	return nil, fmt.Errorf("no valid size chart found on page")
}
