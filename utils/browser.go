package utils

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/chromedp/chromedp"
	"shopify-extractor/internal/types"
)

// BrowserClient provides headless browser functionality
type BrowserClient struct {
	config *types.Config
	logger types.Logger
}

// NewBrowserClient creates a new browser client
func NewBrowserClient(config *types.Config, logger types.Logger) *BrowserClient {
	// Suppress chromedp debug logging
	log.SetOutput(io.Discard)
	
	return &BrowserClient{
		config: config,
		logger: logger,
	}
}

// GetPageContent retrieves the HTML content of a page using headless browser
func (b *BrowserClient) GetPageContent(ctx context.Context, url string) (string, error) {
	// Create a new browser context
	browserCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	// Set timeout
	browserCtx, cancel = context.WithTimeout(browserCtx, b.config.Timeout)
	defer cancel()

	var html string
	
	// Navigate to the page and wait for it to load
	err := chromedp.Run(browserCtx,
		chromedp.Navigate(url),
		chromedp.Sleep(500*time.Millisecond), // Reduced wait time for dynamic content
		chromedp.OuterHTML("html", &html),
	)

	if err != nil {
		return "", fmt.Errorf("failed to get page content: %w", err)
	}

	b.logger.Debugf("Successfully retrieved page content from %s (%d bytes)", url, len(html))
	return html, nil
}

// ExecuteJavaScript executes JavaScript code on the page
func (b *BrowserClient) ExecuteJavaScript(ctx context.Context, url string, script string) (string, error) {
	// Create a new browser context
	browserCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	// Set timeout
	browserCtx, cancel = context.WithTimeout(browserCtx, b.config.Timeout)
	defer cancel()

	var result string
	
	// Navigate to the page and execute JavaScript
	err := chromedp.Run(browserCtx,
		chromedp.Navigate(url),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Evaluate(script, &result),
	)

	if err != nil {
		return "", fmt.Errorf("failed to execute JavaScript: %w", err)
	}

	return result, nil
}

// WaitForElement waits for a specific element to appear on the page
func (b *BrowserClient) WaitForElement(ctx context.Context, url string, selector string) error {
	// Create a new browser context
	browserCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	// Set timeout
	browserCtx, cancel = context.WithTimeout(browserCtx, b.config.Timeout)
	defer cancel()

	// Navigate to the page and wait for element
	err := chromedp.Run(browserCtx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(selector),
	)

	if err != nil {
		return fmt.Errorf("failed to wait for element %s: %w", selector, err)
	}

	return nil
}

// GetElementText retrieves the text content of a specific element
func (b *BrowserClient) GetElementText(ctx context.Context, url string, selector string) (string, error) {
	// Create a new browser context
	browserCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	// Set timeout
	browserCtx, cancel = context.WithTimeout(browserCtx, b.config.Timeout)
	defer cancel()

	var text string
	
	// Navigate to the page and get element text
	err := chromedp.Run(browserCtx,
		chromedp.Navigate(url),
		chromedp.Text(selector, &text),
	)

	if err != nil {
		return "", fmt.Errorf("failed to get element text for %s: %w", selector, err)
	}

	return text, nil
}

// GetElementAttribute retrieves the value of a specific attribute of an element
func (b *BrowserClient) GetElementAttribute(ctx context.Context, url string, selector string, attribute string) (string, error) {
	// Create a new browser context
	browserCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	// Set timeout
	browserCtx, cancel = context.WithTimeout(browserCtx, b.config.Timeout)
	defer cancel()

	var value string
	
	// Navigate to the page and get element attribute
	err := chromedp.Run(browserCtx,
		chromedp.Navigate(url),
		chromedp.AttributeValue(selector, attribute, &value, nil),
	)

	if err != nil {
		return "", fmt.Errorf("failed to get attribute %s for %s: %w", attribute, selector, err)
	}

	return value, nil
} 