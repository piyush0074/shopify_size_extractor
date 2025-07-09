package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"shopify-extractor/internal/types"

	"github.com/PuerkitoBio/goquery"
)

// LittleBoxIndiaAdapter handles extraction for littleboxindia.com
type LittleBoxIndiaAdapter struct {
	*BaseAdapter
}

// NewLittleBoxIndiaAdapter creates a new LittleBoxIndia adapter
func NewLittleBoxIndiaAdapter(config *types.Config, logger types.Logger) *LittleBoxIndiaAdapter {
	return &LittleBoxIndiaAdapter{
		BaseAdapter: NewBaseAdapter(config, logger),
	}
}

// GetStoreName returns the store name
func (l *LittleBoxIndiaAdapter) GetStoreName() string {
	return "littleboxindia.com"
}

// GetProductURLs returns a list of product URLs for LittleBoxIndia
func (l *LittleBoxIndiaAdapter) GetProductURLs(ctx types.Context) ([]string, error) {
	l.logger.Info("Starting product discovery for LittleBoxIndia")

	// Step 1: Get the products page
	productsPageURL := "https://www.littleboxindia.com/products"
	l.logger.Debugf("Fetching products page: %s", productsPageURL)

	html, err := l.GetPageContent(context.Background(), productsPageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get products page: %w", err)
	}

	doc, err := l.ParseHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to parse products page: %w", err)
	}

	// Step 2: Find all collection URLs
	collectionURLs, err := l.ExtractCollectionURLs(doc, "https://www.littleboxindia.com")
	if err != nil {
		return nil, fmt.Errorf("failed to extract collection URLs: %w", err)
	}

	l.logger.Infof("Found %d collections", len(collectionURLs))

	// Step 3: Iterate through collections to find product URLs
	var allProductURLs []string
	for i, collectionURL := range collectionURLs {
		l.logger.Debugf("Processing collection: %s %d", collectionURL, i+1)

		productURLs, err := l.extractProductURLsFromCollection(collectionURL)
		if err != nil {
			l.logger.Warnf("Failed to extract products from collection %s: %v", collectionURL, err)
			continue
		}

		allProductURLs = append(allProductURLs, productURLs...)
		l.logger.Debugf("Found %d products in collection %s", len(productURLs), collectionURL)
		// Process only first few collections for speed testing
		if i >= 4 { // Process first 3 collections only
			break
		}
	}

	// Remove duplicates
	uniqueProductURLs := l.RemoveDuplicateURLs(allProductURLs)

	l.logger.Infof("Total unique products found: %d", len(uniqueProductURLs))
	return uniqueProductURLs, nil
}

// extractProductURLsFromCollection extracts product URLs from a collection page
func (l *LittleBoxIndiaAdapter) extractProductURLsFromCollection(collectionURL string) ([]string, error) {
	l.logger.Debugf("Extracting products from collection: %s", collectionURL)

	// Get the collection page
	html, err := l.GetPageContent(context.Background(), collectionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection page: %w", err)
	}

	doc, err := l.ParseHTML(html)
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
			href = "https://www.littleboxindia.com" + href
		} else if !strings.HasPrefix(href, "http") {
			href = "https://www.littleboxindia.com/" + href
		}

		// Validate URL
		if _, err := url.Parse(href); err == nil {
			productURLs = append(productURLs, href)
		}
	})

	return productURLs, nil
}

// ExtractSizeChart extracts the size chart from a LittleBoxIndia product page
func (l *LittleBoxIndiaAdapter) ExtractSizeChart(ctx types.Context, productURL string) (*types.SizeChart, error) {
	l.logger.Debugf("Extracting size chart from %s", productURL)

	// Get page content
	html, err := l.GetPageContent(context.Background(), productURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get page content: %w", err)
	}

	// Parse HTML
	doc, err := l.ParseHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Find the ks-table (custom size chart table)
	table := doc.Find("table.ks-table").First()
	if table.Length() == 0 {
		l.logger.Debugf("No table found with selector: table.ks-table")
		// Let's also check if there are any tables at all
		allTables := doc.Find("table")
		l.logger.Debugf("Found %d total tables on the page", allTables.Length())
		allTables.Each(func(i int, s *goquery.Selection) {
			class, _ := s.Attr("class")
			l.logger.Debugf("Table %d has class: %s", i, class)
		})
		return nil, fmt.Errorf("no valid size chart found on page")
	}
	l.logger.Debugf("Found table with selector: table.ks-table")

	// Get all rows with ks-table-row class
	rows := table.Find("tr.ks-table-row")
	if rows.Length() == 0 {
		l.logger.Debugf("No rows found with selector: tr.ks-table-row")
		return nil, fmt.Errorf("no valid size chart rows found")
	}
	l.logger.Debugf("Found %d rows with ks-table-row class", rows.Length())

	// Extract headers from the first row (skip the first cell "SIZE")
	var sizes []string
	firstRow := rows.First()
	firstRow.Find("td, th").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return // Skip the first cell which is "SIZE"
		}
		size := strings.TrimSpace(s.Text())
		if size != "" {
			sizes = append(sizes, size)
		}
	})
	l.logger.Debugf("Extracted sizes: %v", sizes)

	if len(sizes) == 0 {
		return nil, fmt.Errorf("no size headers found")
	}

	// Define the measurement types we want to extract
	wantedMeasurements := map[string]string{
		"TO FIT BUST":  "Bust",
		"TO FIT WAIST": "Waist",
		"TO FIT HIP":   "Hip",
	}

	// Prepare data for inches (unit "0") - default for single chart
	inchRows := [][]string{}

	// Process each row starting from the second row
	for i := 1; i < rows.Length(); i++ {
		row := rows.Eq(i)
		cells := row.Find("td, th")

		// Get the measurement label (first cell)
		label := strings.ToUpper(strings.TrimSpace(cells.First().Text()))
		outLabel, ok := wantedMeasurements[label]
		if !ok {
			continue // Skip rows we don't want
		}

		l.logger.Debugf("Processing measurement: %s -> %s", label, outLabel)

		// Prepare row for this measurement
		inchRow := []string{outLabel}

		// Process each data cell (skip the first cell which is the label)
		cells.Each(func(j int, cell *goquery.Selection) {
			if j == 0 {
				return // Skip the label cell
			}

			// Check for data-unit-values attribute
			dataUnitValues := cell.AttrOr("data-unit-values", "")
			if dataUnitValues != "" {
				// Parse the JSON data-unit-values
				// Replace &quot; with " for proper JSON parsing
				cleanJSON := strings.ReplaceAll(dataUnitValues, "&quot;", `"`)
				var unitMap map[string]string
				if err := json.Unmarshal([]byte(cleanJSON), &unitMap); err == nil {
					// "0" = inches, "1" = cm - use inches for single chart
					if inchVal, ok := unitMap["0"]; ok {
						inchRow = append(inchRow, inchVal)
					} else {
						inchRow = append(inchRow, "")
					}
				} else {
					l.logger.Debugf("Failed to parse data-unit-values: %s, error: %v", dataUnitValues, err)
					// Fallback to text content
					val := strings.TrimSpace(cell.Text())
					inchRow = append(inchRow, val)
				}
			} else {
				// No data-unit-values, use text content
				val := strings.TrimSpace(cell.Text())
				inchRow = append(inchRow, val)
			}
		})

		// Only add rows if we have the right number of values
		if len(inchRow) == len(sizes)+1 {
			inchRows = append(inchRows, inchRow)
		}
	}

	l.logger.Debugf("Extracted %d inch rows", len(inchRows))

	// Build the size chart for inches (default)
	if len(inchRows) > 0 {
		// Create headers: ["Size"] + sizes
		headers := []string{"Size"}
		headers = append(headers, sizes...)

		// Convert rows to the expected format
		var rows []map[string]string
		for _, row := range inchRows {
			rowMap := make(map[string]string)
			for j, value := range row {
				if j < len(headers) {
					rowMap[headers[j]] = value
				}
			}
			rows = append(rows, rowMap)
		}

		sizeChart := &types.SizeChart{
			Headers: headers,
			Rows:    rows,
		}

		if l.IsValidSizeChart(sizeChart) {
			l.logger.Debugf("Successfully extracted size chart with %d rows", len(rows))
			return sizeChart, nil
		}
	}

	return nil, fmt.Errorf("no valid size chart found on page")
}

// GetProductTitle extracts the product title from a LittleBoxIndia product page
func (l *LittleBoxIndiaAdapter) GetProductTitle(ctx types.Context, productURL string) (string, error) {
	l.logger.Debugf("Extracting product title from %s", productURL)

	// Get page content
	html, err := l.GetPageContent(context.Background(), productURL)
	if err != nil {
		return "", fmt.Errorf("failed to get page content: %w", err)
	}

	// Parse HTML
	doc, err := l.ParseHTML(html)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

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
		title, err := l.ExtractText(doc, selector)
		if err == nil && title != "" {
			l.logger.Debugf("Successfully extracted product title using selector: %s", selector)
			return title, nil
		}
	}

	return "", fmt.Errorf("product title not found on page")
}

// ExtractAllSizeCharts extracts all size charts from a LittleBoxIndia product page
func (l *LittleBoxIndiaAdapter) ExtractAllSizeCharts(ctx types.Context, productURL string) ([]*types.SizeChart, error) {
	l.logger.Debugf("Extracting all size charts from %s", productURL)
	var charts []*types.SizeChart

	// Get page content
	html, err := l.GetPageContent(context.Background(), productURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get page content: %w", err)
	}

	// Parse HTML
	doc, err := l.ParseHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Find the ks-table (custom size chart table)
	table := doc.Find("table.ks-table").First()
	if table.Length() == 0 {
		l.logger.Debugf("No table found with selector: table.ks-table")
		return nil, fmt.Errorf("no valid size chart found on page")
	}
	l.logger.Debugf("Found table with selector: table.ks-table")

	// Get all rows with ks-table-row class
	rows := table.Find("tr.ks-table-row")
	if rows.Length() == 0 {
		l.logger.Debugf("No rows found with selector: tr.ks-table-row")
		return nil, fmt.Errorf("no valid size chart rows found")
	}
	l.logger.Debugf("Found %d rows with ks-table-row class", rows.Length())

	// Extract headers from the first row (skip the first cell "SIZE")
	var sizes []string
	firstRow := rows.First()
	firstRow.Find("td, th").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return // Skip the first cell which is "SIZE"
		}
		size := strings.TrimSpace(s.Text())
		if size != "" {
			sizes = append(sizes, size)
		}
	})
	l.logger.Debugf("Extracted sizes: %v", sizes)

	if len(sizes) == 0 {
		return nil, fmt.Errorf("no size headers found")
	}

	// Define the measurement types we want to extract
	wantedMeasurements := map[string]string{
		"TO FIT BUST":  "Bust",
		"TO FIT WAIST": "Waist",
		"TO FIT HIP":   "Hip",
	}

	// Prepare data structures for inches and centimeters
	// Each size will have its own row with measurements as columns
	inchData := make(map[string]map[string]string) // size -> measurement -> value
	cmData := make(map[string]map[string]string)   // size -> measurement -> value

	// Initialize data structures for each size
	for _, size := range sizes {
		inchData[size] = make(map[string]string)
		cmData[size] = make(map[string]string)
	}

	// Process each row starting from the second row
	for i := 1; i < rows.Length(); i++ {
		row := rows.Eq(i)
		cells := row.Find("td, th")

		// Get the measurement label (first cell)
		label := strings.ToUpper(strings.TrimSpace(cells.First().Text()))
		outLabel, ok := wantedMeasurements[label]
		if !ok {
			continue // Skip rows we don't want
		}

		l.logger.Debugf("Processing measurement: %s -> %s", label, outLabel)

		// Process each data cell (skip the first cell which is the label)
		cellIndex := 0
		cells.Each(func(j int, cell *goquery.Selection) {
			if j == 0 {
				return // Skip the label cell
			}

			if cellIndex < len(sizes) {
				size := sizes[cellIndex]

				// Check for data-unit-values attribute
				dataUnitValues := cell.AttrOr("data-unit-values", "")
				if dataUnitValues != "" {
					// Parse the JSON data-unit-values
					// Replace &quot; with " for proper JSON parsing
					cleanJSON := strings.ReplaceAll(dataUnitValues, "&quot;", `"`)
					var unitMap map[string]string
					if err := json.Unmarshal([]byte(cleanJSON), &unitMap); err == nil {
						// "0" = inches, "1" = cm
						if inchVal, ok := unitMap["0"]; ok {
							inchData[size][outLabel] = inchVal
						}
						if cmVal, ok := unitMap["1"]; ok {
							cmData[size][outLabel] = cmVal
						}
					} else {
						l.logger.Debugf("Failed to parse data-unit-values: %s, error: %v", dataUnitValues, err)
						// Fallback to text content
						val := strings.TrimSpace(cell.Text())
						inchData[size][outLabel] = val
						cmData[size][outLabel] = val
					}
				} else {
					// No data-unit-values, use text content
					val := strings.TrimSpace(cell.Text())
					inchData[size][outLabel] = val
					cmData[size][outLabel] = val
				}
			}
			cellIndex++
		})
	}

	l.logger.Debugf("Extracted data for %d sizes", len(sizes))

	// Build size chart for inches
	inchHeaders := []string{"Size", "Bust (in)", "Waist (in)", "Hip (in)"}
	var inchRows []map[string]string
	for _, size := range sizes {
		row := map[string]string{"Size": size}
		for _, measurement := range []string{"Bust", "Waist", "Hip"} {
			if val, ok := inchData[size][measurement]; ok {
				row[measurement+" (in)"] = val
			}
		}
		// Only add row if it has at least one measurement
		if row["Bust (in)"] != "" || row["Waist (in)"] != "" || row["Hip (in)"] != "" {
			inchRows = append(inchRows, row)
		}
	}

	if len(inchRows) > 0 {
		inchChart := &types.SizeChart{
			Headers: inchHeaders,
			Rows:    inchRows,
		}

		if l.IsValidSizeChart(inchChart) {
			l.logger.Debugf("Successfully extracted inches size chart with %d rows", len(inchRows))
			charts = append(charts, inchChart)
		}
	}

	// Build size chart for centimeters
	cmHeaders := []string{"Size", "Bust (cm)", "Waist (cm)", "Hip (cm)"}
	var cmRows []map[string]string
	for _, size := range sizes {
		row := map[string]string{"Size": size}
		for _, measurement := range []string{"Bust", "Waist", "Hip"} {
			if val, ok := cmData[size][measurement]; ok {
				row[measurement+" (cm)"] = val
			}
		}
		// Only add row if it has at least one measurement
		if row["Bust (cm)"] != "" || row["Waist (cm)"] != "" || row["Hip (cm)"] != "" {
			cmRows = append(cmRows, row)
		}
	}

	if len(cmRows) > 0 {
		cmChart := &types.SizeChart{
			Headers: cmHeaders,
			Rows:    cmRows,
		}

		if l.IsValidSizeChart(cmChart) {
			l.logger.Debugf("Successfully extracted centimeters size chart with %d rows", len(cmRows))
			charts = append(charts, cmChart)
		}
	}

	if len(charts) == 0 {
		return nil, fmt.Errorf("no valid size chart found on page")
	}
	return charts, nil
}

// ExtractProductTitleAndSizeCharts extracts both product title and size charts from a LittleBoxIndia product page
// This method fetches the page once and extracts both pieces of information
func (l *LittleBoxIndiaAdapter) ExtractProductTitleAndSizeCharts(ctx types.Context, productURL string) (string, []*types.SizeChart, error) {
	l.logger.Debugf("Extracting product title and size charts from %s", productURL)

	// Get page content once
	html, err := l.GetPageContent(context.Background(), productURL)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get page content: %w", err)
	}

	// Parse HTML once
	doc, err := l.ParseHTML(html)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract product title
	var title string
	selectors := []string{
		"h1.product-title",
		"h1[class*='title']",
		".product-name h1",
		".product-info h1",
		".product-details h1",
		"h1",
	}

	for _, selector := range selectors {
		title, err = l.ExtractText(doc, selector)
		if err == nil && title != "" {
			l.logger.Debugf("Successfully extracted product title using selector: %s", selector)
			break
		}
	}

	if title == "" {
		title = "Unknown Product"
	}

	// Extract size charts using the same document
	var charts []*types.SizeChart

	// Find the ks-table (custom size chart table)
	table := doc.Find("table.ks-table").First()
	if table.Length() == 0 {
		l.logger.Debugf("No table found with selector: table.ks-table")
		return title, nil, fmt.Errorf("no valid size chart found on page")
	}
	l.logger.Debugf("Found table with selector: table.ks-table")

	// Get all rows with ks-table-row class
	rows := table.Find("tr.ks-table-row")
	if rows.Length() == 0 {
		l.logger.Debugf("No rows found with selector: tr.ks-table-row")
		return title, nil, fmt.Errorf("no valid size chart rows found")
	}
	l.logger.Debugf("Found %d rows with ks-table-row class", rows.Length())

	// Extract headers from the first row (skip the first cell "SIZE")
	var sizes []string
	firstRow := rows.First()
	firstRow.Find("td, th").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return // Skip the first cell which is "SIZE"
		}
		size := strings.TrimSpace(s.Text())
		if size != "" {
			sizes = append(sizes, size)
		}
	})
	l.logger.Debugf("Extracted sizes: %v", sizes)

	if len(sizes) == 0 {
		return title, nil, fmt.Errorf("no size headers found")
	}

	// Define the measurement types we want to extract
	wantedMeasurements := map[string]string{
		"TO FIT BUST":  "Bust",
		"TO FIT WAIST": "Waist",
		"TO FIT HIP":   "Hip",
	}

	// Prepare data structures for inches and centimeters
	// Each size will have its own row with measurements as columns
	inchData := make(map[string]map[string]string) // size -> measurement -> value
	cmData := make(map[string]map[string]string)   // size -> measurement -> value

	// Initialize data structures for each size
	for _, size := range sizes {
		inchData[size] = make(map[string]string)
		cmData[size] = make(map[string]string)
	}

	// Process each row starting from the second row
	for i := 1; i < rows.Length(); i++ {
		row := rows.Eq(i)
		cells := row.Find("td, th")

		// Get the measurement label (first cell)
		label := strings.ToUpper(strings.TrimSpace(cells.First().Text()))
		outLabel, ok := wantedMeasurements[label]
		if !ok {
			continue // Skip rows we don't want
		}

		l.logger.Debugf("Processing measurement: %s -> %s", label, outLabel)

		// Process each data cell (skip the first cell which is the label)
		cellIndex := 0
		cells.Each(func(j int, cell *goquery.Selection) {
			if j == 0 {
				return // Skip the label cell
			}

			if cellIndex < len(sizes) {
				size := sizes[cellIndex]

				// Check for data-unit-values attribute
				dataUnitValues := cell.AttrOr("data-unit-values", "")
				if dataUnitValues != "" {
					// Parse the JSON data-unit-values
					// Replace &quot; with " for proper JSON parsing
					cleanJSON := strings.ReplaceAll(dataUnitValues, "&quot;", `"`)
					var unitMap map[string]string
					if err := json.Unmarshal([]byte(cleanJSON), &unitMap); err == nil {
						// "0" = inches, "1" = cm
						if inchVal, ok := unitMap["0"]; ok {
							inchData[size][outLabel] = inchVal
						}
						if cmVal, ok := unitMap["1"]; ok {
							cmData[size][outLabel] = cmVal
						}
					} else {
						l.logger.Debugf("Failed to parse data-unit-values: %s, error: %v", dataUnitValues, err)
						// Fallback to text content
						val := strings.TrimSpace(cell.Text())
						inchData[size][outLabel] = val
						cmData[size][outLabel] = val
					}
				} else {
					// No data-unit-values, use text content
					val := strings.TrimSpace(cell.Text())
					inchData[size][outLabel] = val
					cmData[size][outLabel] = val
				}
			}
			cellIndex++
		})
	}

	l.logger.Debugf("Extracted data for %d sizes", len(sizes))

	// Build size chart for inches
	inchHeaders := []string{"Size", "Bust (in)", "Waist (in)", "Hip (in)"}
	var inchRows []map[string]string
	for _, size := range sizes {
		row := map[string]string{"Size": size}
		for _, measurement := range []string{"Bust", "Waist", "Hip"} {
			if val, ok := inchData[size][measurement]; ok {
				row[measurement+" (in)"] = val
			}
		}
		// Only add row if it has at least one measurement
		if row["Bust (in)"] != "" || row["Waist (in)"] != "" || row["Hip (in)"] != "" {
			inchRows = append(inchRows, row)
		}
	}

	if len(inchRows) > 0 {
		inchChart := &types.SizeChart{
			Headers: inchHeaders,
			Rows:    inchRows,
		}

		if l.IsValidSizeChart(inchChart) {
			l.logger.Debugf("Successfully extracted inches size chart with %d rows", len(inchRows))
			charts = append(charts, inchChart)
		}
	}

	// Build size chart for centimeters
	cmHeaders := []string{"Size", "Bust (cm)", "Waist (cm)", "Hip (cm)"}
	var cmRows []map[string]string
	for _, size := range sizes {
		row := map[string]string{"Size": size}
		for _, measurement := range []string{"Bust", "Waist", "Hip"} {
			if val, ok := cmData[size][measurement]; ok {
				row[measurement+" (cm)"] = val
			}
		}
		// Only add row if it has at least one measurement
		if row["Bust (cm)"] != "" || row["Waist (cm)"] != "" || row["Hip (cm)"] != "" {
			cmRows = append(cmRows, row)
		}
	}

	if len(cmRows) > 0 {
		cmChart := &types.SizeChart{
			Headers: cmHeaders,
			Rows:    cmRows,
		}

		if l.IsValidSizeChart(cmChart) {
			l.logger.Debugf("Successfully extracted centimeters size chart with %d rows", len(cmRows))
			charts = append(charts, cmChart)
		}
	}

	return title, charts, nil
}
