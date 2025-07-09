package adapters

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"shopify-extractor/internal/types"

	"github.com/PuerkitoBio/goquery"
)

// WestsideAdapter handles extraction for westside.com
type WestsideAdapter struct {
	*BaseAdapter
}

// NewWestsideAdapter creates a new Westside adapter
func NewWestsideAdapter(config *types.Config, logger types.Logger) *WestsideAdapter {
	config.UseHeadlessBrowser = true // Always use browser for Westside
	return &WestsideAdapter{
		BaseAdapter: NewBaseAdapter(config, logger),
	}
}

// GetStoreName returns the store name
func (w *WestsideAdapter) GetStoreName() string {
	return "westside.com"
}

// GetProductURLs returns a list of product URLs for Westside
func (w *WestsideAdapter) GetProductURLs(ctx types.Context) ([]string, error) {
	startTime := time.Now()
	w.logger.Info("Starting product discovery for Westside")

	// Step 1: Get the products page
	productsPageURL := "https://www.westside.com/products"
	w.logger.Debugf("Fetching products page: %s", productsPageURL)

	html, err := w.GetPageContent(context.Background(), productsPageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get products page: %w", err)
	}

	doc, err := w.ParseHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to parse products page: %w", err)
	}

	// Step 2: Find all collection URLs
	collectionURLs, err := w.ExtractCollectionURLs(doc, "https://www.westside.com")
	if err != nil {
		return nil, fmt.Errorf("failed to extract collection URLs: %w", err)
	}

	w.logger.Infof("Found %d collections", len(collectionURLs))

	// Step 3: Iterate through collections to find product URLs
	var allProductURLs []string
	totalProductsFound := 0
	for i, collectionURL := range collectionURLs {
		// if i == 0 {
		// 	continue
		// }
		collectionStartTime := time.Now()
		w.logger.Debugf("Processing collection %d/%d: %s", i+1, len(collectionURLs), collectionURL)

		productURLs, err := w.extractProductURLsFromCollection(collectionURL)
		if err != nil {
			w.logger.Warnf("Failed to extract products from collection %s: %v", collectionURL, err)
			continue
		}

		collectionTime := time.Since(collectionStartTime)
		allProductURLs = append(allProductURLs, productURLs...)
		totalProductsFound += len(productURLs)
		w.logger.Debugf("Collection %s processed in %v, found %d products (total so far: %d)", collectionURL, collectionTime, len(productURLs), totalProductsFound)

		// Process only first few collections for speed testing
		// if i >= 4 { // Process first 3 collections only
		// 	break
		// }
	}

	// Remove duplicates
	uniqueProductURLs := w.RemoveDuplicateURLs(allProductURLs)

	totalTime := time.Since(startTime)
	w.logger.Infof("Product discovery completed in %v", totalTime)
	w.logger.Infof("Total unique products found: %d", len(uniqueProductURLs))
	return uniqueProductURLs, nil
}

// extractProductURLsFromCollection extracts product URLs from a collection page
func (w *WestsideAdapter) extractProductURLsFromCollection(collectionURL string) ([]string, error) {
	w.logger.Debugf("Extracting products from collection: %s", collectionURL)

	// Get the collection page
	html, err := w.GetPageContent(context.Background(), collectionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection page: %w", err)
	}

	doc, err := w.ParseHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to parse collection page: %w", err)
	}

	var productURLs []string

	// First, try to find products in the wizzy-search-results container (much faster)
	doc.Find(".wizzy-search-results a[href*='/products/']").Each(func(i int, s *goquery.Selection) {
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
			href = "https://www.westside.com" + href
		} else if !strings.HasPrefix(href, "http") {
			href = "https://www.westside.com/" + href
		}

		// Validate URL and ensure it's a Westside product
		if parsedURL, err := url.Parse(href); err == nil {
			// Only include URLs from westside.com domain
			if strings.Contains(parsedURL.Hostname(), "westside.com") {
				productURLs = append(productURLs, href)
			}
		}
	})

	// Also try to find products in swiper containers
	doc.Find(".swiper a[href*='/products/']").Each(func(i int, s *goquery.Selection) {
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
			href = "https://www.westside.com" + href
		} else if !strings.HasPrefix(href, "http") {
			href = "https://www.westside.com/" + href
		}

		// Validate URL and ensure it's a Westside product
		if parsedURL, err := url.Parse(href); err == nil {
			// Only include URLs from westside.com domain
			if strings.Contains(parsedURL.Hostname(), "westside.com") {
				productURLs = append(productURLs, href)
			}
		}
	})

	// If no products found in wizzy-search-results and swiper, fall back to searching the entire page
	if len(productURLs) == 0 {
		w.logger.Debugf("No products found in .wizzy-search-results or .swiper, searching entire page")
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
				href = "https://www.westside.com" + href
			} else if !strings.HasPrefix(href, "http") {
				href = "https://www.westside.com/" + href
			}

			// Validate URL and ensure it's a Westside product
			if parsedURL, err := url.Parse(href); err == nil {
				// Only include URLs from westside.com domain
				if strings.Contains(parsedURL.Hostname(), "westside.com") {
					productURLs = append(productURLs, href)
				}
			}
		})
	}

	w.logger.Debugf("Found %d products using .wizzy-search-results and .swiper selectors", len(productURLs))
	return productURLs, nil
}

// ExtractSizeChart extracts the size chart from a Westside product page
func (w *WestsideAdapter) ExtractSizeChart(ctx types.Context, productURL string) (*types.SizeChart, error) {
	startTime := time.Now()
	w.logger.Debugf("Extracting size chart from %s", productURL)

	// Get page content
	html, err := w.GetPageContent(context.Background(), productURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get page content: %w", err)
	}

	// Parse HTML
	doc, err := w.ParseHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Use the specific sizeguide selector for faster extraction
	selector := ".sizeguide table"
	table := doc.Find(selector).First()
	if table.Length() == 0 {
		return nil, fmt.Errorf("size chart table not found in .sizeguide container")
	}

	w.logger.Debugf("Found size chart table using selector: %s", selector)

	// Extract both inches and centimeters from the same table
	// The table contains both units in span elements with classes "default" (cm) and "alt" (inches)
	result, err := w.extractDualUnitSizeChart(doc, selector)
	if err == nil {
		extractionTime := time.Since(startTime)
		w.logger.Debugf("Size chart extraction completed in %v", extractionTime)
	}
	return result, err
}

// extractDualUnitSizeChart extracts both inches and centimeters from the Westside size chart
func (w *WestsideAdapter) extractDualUnitSizeChart(doc *goquery.Document, selector string) (*types.SizeChart, error) {
	table := doc.Find(selector).First()
	if table.Length() == 0 {
		return nil, fmt.Errorf("size chart table not found")
	}

	// Extract headers
	headers := []string{}
	table.Find("thead tr th, tr:first-child th, tr:first-child td").Each(func(i int, s *goquery.Selection) {
		header := strings.TrimSpace(s.Text())
		if header != "" {
			headers = append(headers, header)
		}
	})

	if len(headers) == 0 {
		return nil, fmt.Errorf("no headers found in size chart")
	}

	w.logger.Debugf("Found headers: %v", headers)

	// Create size chart with clean headers
	sizeChart := &types.SizeChart{
		Headers: []string{"Size"},
		Rows:    []map[string]string{},
	}

	// Add measurement headers (cm and inches)
	for _, header := range headers {
		if !strings.Contains(strings.ToLower(header), "size") {
			cleanHeader := w.cleanHeader(header)
			if cleanHeader != "" { // Only add if it's a recognized measurement
				sizeChart.Headers = append(sizeChart.Headers, cleanHeader+" (cm)")
				sizeChart.Headers = append(sizeChart.Headers, cleanHeader+" (in)")
			}
		}
	}

	w.logger.Debugf("Final headers: %v", sizeChart.Headers)

	// Extract rows
	table.Find("tbody tr, tr:not(:first-child)").Each(func(i int, s *goquery.Selection) {
		row := make(map[string]string)
		colIndex := 0

		s.Find("td, th").Each(func(j int, cell *goquery.Selection) {
			if colIndex >= len(headers) {
				return
			}

			header := headers[colIndex]
			if strings.Contains(strings.ToLower(header), "size") {
				// Extract size
				sizeText := strings.TrimSpace(cell.Find("span.default").First().Text())
				if sizeText == "" {
					sizeText = strings.TrimSpace(cell.Text())
				}
				sizeText = w.cleanSizeText(sizeText)
				row["Size"] = sizeText
				colIndex++
			} else {
				// Extract measurements (cm and inches)
				cmValue := strings.TrimSpace(cell.Find("span.default").First().Text())
				inValue := strings.TrimSpace(cell.Find("span.alt").First().Text())

				cleanHeader := w.cleanHeader(header)
				if cleanHeader != "" { // Only add if it's a recognized measurement
					row[cleanHeader+" (cm)"] = cmValue
					row[cleanHeader+" (in)"] = inValue
				}
				colIndex++
			}
		})

		if len(row) > 0 {
			sizeChart.Rows = append(sizeChart.Rows, row)
		}
	})

	if len(sizeChart.Rows) == 0 {
		return nil, fmt.Errorf("no data rows found in size chart")
	}

	return sizeChart, nil
}

// cleanHeader cleans up header text for consistent naming
func (w *WestsideAdapter) cleanHeader(header string) string {
	header = strings.ToLower(strings.TrimSpace(header))

	// Handle common measurement types
	if strings.Contains(header, "shoulder") || strings.Contains(header, "to fit shoulder") {
		return "Shoulder"
	}
	if strings.Contains(header, "chest") || strings.Contains(header, "to fit chest") {
		return "Chest"
	}
	if strings.Contains(header, "waist") || strings.Contains(header, "to fit waist") {
		return "Waist"
	}
	if strings.Contains(header, "hip") || strings.Contains(header, "to fit hip") {
		return "Hip"
	}
	if strings.Contains(header, "bust") || strings.Contains(header, "to fit bust") {
		return "Bust"
	}

	// If not a recognized measurement, return empty to skip it
	return ""
}

// cleanSizeText removes duplicate size text
func (w *WestsideAdapter) cleanSizeText(sizeText string) string {
	// Remove duplicates like "XS - 36XS - 36" -> "XS - 36"
	parts := strings.Fields(sizeText)
	if len(parts) >= 2 {
		// Take first two parts (e.g., "XS - 36")
		return strings.Join(parts[:2], " ")
	}
	return sizeText
}

// GetProductTitle extracts the product title from a Westside product page
func (w *WestsideAdapter) GetProductTitle(ctx types.Context, productURL string) (string, error) {
	w.logger.Debugf("Extracting product title from %s", productURL)

	// Get page content
	html, err := w.GetPageContent(context.Background(), productURL)
	if err != nil {
		return "", fmt.Errorf("failed to get page content: %w", err)
	}

	// Parse HTML
	doc, err := w.ParseHTML(html)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Try different selectors for product title
	selectors := []string{
		".product__title h1",
		"h1.product-title",
		"h1[class*='title']",
		".product-name h1",
		".product-info h1",
		".product-details h1",
		"h1",
	}

	for _, selector := range selectors {
		title, err := w.ExtractText(doc, selector)
		if err == nil && title != "" {
			w.logger.Debugf("Successfully extracted product title using selector: %s", selector)
			return title, nil
		}
	}

	return "", fmt.Errorf("product title not found on page")
}

// GetProductTitleFromDoc extracts the product title from an already parsed document
func (w *WestsideAdapter) GetProductTitleFromDoc(doc *goquery.Document) (string, error) {
	// Try different selectors for product title
	selectors := []string{
		".product__title h1",
		"h1.product-title",
		"h1[class*='title']",
		".product-name h1",
		".product-info h1",
		".product-details h1",
		"h1",
	}

	for _, selector := range selectors {
		title, err := w.ExtractText(doc, selector)
		if err == nil && title != "" {
			w.logger.Debugf("Successfully extracted product title using selector: %s", selector)
			return title, nil
		}
	}

	return "", fmt.Errorf("product title not found on page")
}

func normalizeHeader(header, unit string) string {
	h := strings.ToLower(header)
	if strings.Contains(h, "bust") {
		return "Bust (" + unit + ")"
	}
	if strings.Contains(h, "waist") {
		return "Waist (" + unit + ")"
	}
	if strings.Contains(h, "hip") {
		return "Hip (" + unit + ")"
	}
	if strings.Contains(h, "size") {
		return "Size"
	}
	return header
}

func splitValue(val string) (string, string) {
	// Split on space, return (in, cm) or (cm, in) based on order
	parts := strings.Fields(val)
	if len(parts) == 2 {
		return parts[1], parts[0] // (in, cm) or (cm, in)
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return "", ""
}

// ExtractAllSizeCharts extracts all size charts from a Westside product page
func (w *WestsideAdapter) ExtractAllSizeCharts(ctx types.Context, productURL string) (string, []*types.SizeChart, error) {
	startTime := time.Now()
	w.logger.Debugf("Extracting all size charts from %s", productURL)

	// Get page content once and reuse it
	html, err := w.GetPageContent(context.Background(), productURL)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get page content: %w", err)
	}

	// Parse HTML once
	doc, err := w.ParseHTML(html)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract both title and size chart from the same document
	title, _ := w.GetProductTitleFromDoc(doc)
	if title != "" {
		w.logger.Debugf("Extracted title: %s", title)
	}

	// Extract size chart using the cached document
	sizeChart, err := w.extractSizeChartFromDoc(doc, productURL)
	if err != nil {
		return title, nil, err
	}

	extractionTime := time.Since(startTime)
	w.logger.Debugf("Complete product extraction completed in %v", extractionTime)

	if sizeChart == nil {
		return title, nil, fmt.Errorf("no size chart found")
	}

	// Build two separate charts: one for inches, one for centimeters
	var charts []*types.SizeChart

	// Extract measurement names from headers (excluding Size and unit suffixes)
	var measurements []string
	for _, header := range sizeChart.Headers {
		if header == "Size" {
			continue
		}
		baseName := strings.TrimSuffix(strings.TrimSuffix(header, " (cm)"), " (in)")
		if baseName != header {
			measurements = append(measurements, baseName)
		}
	}
	// Remove duplicates
	uniqueMeasurements := make([]string, 0)
	seen := make(map[string]bool)
	for _, m := range measurements {
		if !seen[m] {
			seen[m] = true
			uniqueMeasurements = append(uniqueMeasurements, m)
		}
	}

	// Build inches chart
	inchesChart := &types.SizeChart{
		Headers: []string{"Size"},
		Rows:    []map[string]string{},
	}
	for _, measurement := range uniqueMeasurements {
		inchesChart.Headers = append(inchesChart.Headers, measurement+" (in)")
	}
	for _, row := range sizeChart.Rows {
		inchesRow := make(map[string]string)
		if size, exists := row["Size"]; exists {
			inchesRow["Size"] = size
		}
		for _, measurement := range uniqueMeasurements {
			if inValue, exists := row[measurement+" (in)"]; exists {
				inchesRow[measurement+" (in)"] = inValue
			}
		}
		inchesChart.Rows = append(inchesChart.Rows, inchesRow)
	}
	if len(inchesChart.Rows) > 0 {
		charts = append(charts, inchesChart)
	}

	// Build centimeters chart
	cmChart := &types.SizeChart{
		Headers: []string{"Size"},
		Rows:    []map[string]string{},
	}
	for _, measurement := range uniqueMeasurements {
		cmChart.Headers = append(cmChart.Headers, measurement+" (cm)")
	}
	for _, row := range sizeChart.Rows {
		cmRow := make(map[string]string)
		if size, exists := row["Size"]; exists {
			cmRow["Size"] = size
		}
		for _, measurement := range uniqueMeasurements {
			if cmValue, exists := row[measurement+" (cm)"]; exists {
				cmRow[measurement+" (cm)"] = cmValue
			}
		}
		cmChart.Rows = append(cmChart.Rows, cmRow)
	}
	if len(cmChart.Rows) > 0 {
		charts = append(charts, cmChart)
	}

	if len(charts) == 0 {
		return title, nil, fmt.Errorf("no valid size chart found")
	}
	return title, charts, nil
}

// extractSizeChartFromDoc extracts size chart from an already parsed document
func (w *WestsideAdapter) extractSizeChartFromDoc(doc *goquery.Document, productURL string) (*types.SizeChart, error) {
	startTime := time.Now()
	w.logger.Debugf("Extracting size chart from document for %s", productURL)

	// Use the specific sizeguide selector for faster extraction
	selector := ".sizeguide table"
	table := doc.Find(selector).First()
	if table.Length() == 0 {
		return nil, fmt.Errorf("size chart table not found in .sizeguide container")
	}

	w.logger.Debugf("Found size chart table using selector: %s", selector)

	// Extract both inches and centimeters from the same table
	// The table contains both units in span elements with classes "default" (cm) and "alt" (inches)
	result, err := w.extractDualUnitSizeChart(doc, selector)
	if err == nil {
		extractionTime := time.Since(startTime)
		w.logger.Debugf("Size chart extraction completed in %v", extractionTime)
	}
	return result, err
}
