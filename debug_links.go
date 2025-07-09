package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"shopify-extractor/internal/types"
	"shopify-extractor/utils"
)

func main() {
	config := types.DefaultConfig()
	config.UseHeadlessBrowser = true // Use headless browser to test JavaScript-rendered content
	config.Timeout = 30 * config.Timeout

	logger := &debugLogger{}

	// Test Westside
	fmt.Println("=== Testing Westside ===")
	testStore("https://www.westside.com/products", "https://www.westside.com", config, logger)

	fmt.Println("\n=== Testing Suqah ===")
	testStore("https://www.suqah.com/products", "https://www.suqah.com", config, logger)
}

func testStore(productsURL, baseURL string, config *types.Config, logger types.Logger) {
	browserClient := utils.NewBrowserClient(config, logger)

	// Get the products page using headless browser
	html, err := browserClient.GetPageContent(context.Background(), productsURL)
	if err != nil {
		log.Printf("Failed to get products page: %v", err)
		return
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("Failed to parse HTML: %v", err)
		return
	}

	// Find all links
	fmt.Printf("Total links found: %d\n", doc.Find("a").Length())

	// Check for collection links
	collectionLinks := doc.Find("a[href*='collections']")
	fmt.Printf("Links with 'collections' in href: %d\n", collectionLinks.Length())

	collectionLinks.Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		text := strings.TrimSpace(s.Text())
		fmt.Printf("  %d: href='%s', text='%s'\n", i+1, href, text)
	})

	// Check for other patterns
	productLinks := doc.Find("a[href*='/products/']")
	fmt.Printf("Links with '/products/' in href: %d\n", productLinks.Length())

	// Check for any links that might be collections
	allLinks := doc.Find("a")
	fmt.Println("Sample of all links:")
	count := 0
	allLinks.Each(func(i int, s *goquery.Selection) {
		if count >= 10 {
			return
		}
		href, _ := s.Attr("href")
		text := strings.TrimSpace(s.Text())
		if href != "" && len(href) < 100 {
			fmt.Printf("  %d: href='%s', text='%s'\n", i+1, href, text)
			count++
		}
	})
}

type debugLogger struct{}

func (d *debugLogger) Debug(args ...interface{})                 { fmt.Println(args...) }
func (d *debugLogger) Info(args ...interface{})                  { fmt.Println(args...) }
func (d *debugLogger) Warn(args ...interface{})                  { fmt.Println(args...) }
func (d *debugLogger) Error(args ...interface{})                 { fmt.Println(args...) }
func (d *debugLogger) Debugf(format string, args ...interface{}) { fmt.Printf(format+"\n", args...) }
func (d *debugLogger) Infof(format string, args ...interface{})  { fmt.Printf(format+"\n", args...) }
func (d *debugLogger) Warnf(format string, args ...interface{})  { fmt.Printf(format+"\n", args...) }
func (d *debugLogger) Errorf(format string, args ...interface{}) { fmt.Printf(format+"\n", args...) } 