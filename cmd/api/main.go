package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"shopify-extractor/extractor"
	"shopify-extractor/internal/types"
)

// APIRequest represents the request body for the API
type APIRequest struct {
	Stores []string `json:"stores"`
}

// APIResponse represents the response from the API
type APIResponse struct {
	Success bool                    `json:"success"`
	Data    *types.ExtractionResult `json:"data,omitempty"`
	Error   string                  `json:"error,omitempty"`
}

// Server holds the API server configuration
type Server struct {
	logger *logrus.Logger
	config *types.Config
}

// NewServer creates a new API server
func NewServer() *Server {
	// Load .env file if present
	_ = godotenv.Load()

	// Setup logging
	logger := logrus.New()
	
	// Set timestamp format with milliseconds
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})
	
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		if level, err := logrus.ParseLevel(levelStr); err == nil {
			logger.SetLevel(level)
		}
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// Create configuration
	config := &types.Config{
		RequestDelay:           1 * time.Second,
		MaxRetries:            3,
		Timeout:               30 * time.Second,
		MaxConcurrentRequests: 5,
		UseHeadlessBrowser:    true,
		UserAgent:             "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}

	return &Server{
		logger: logger,
		config: config,
	}
}

// handleExtract handles the extraction API endpoint
func (s *Server) handleExtract(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow POST requests
	if r.Method != "POST" {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.Stores) == 0 {
		s.sendError(w, "No stores provided", http.StatusBadRequest)
		return
	}

	// Clean store names
	for i, store := range req.Stores {
		req.Stores[i] = strings.TrimSpace(store)
	}

	s.logger.Infof("API request received for stores: %v", req.Stores)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Extract size charts using individual store extractors
	var storeResults []types.StoreResult
	
	for _, store := range req.Stores {
		s.logger.Infof("Processing store: %s", store)
		
		var storeExtractor interface {
			ExtractAll(context.Context) ([]types.Product, error)
			Close()
		}
		
		// Create the appropriate extractor based on store name
		switch store {
		case "westside.com":
			storeExtractor = extractor.NewWestsideExtractor(s.config, s.logger)
		case "littleboxindia.com":
			storeExtractor = extractor.NewLittleBoxIndiaExtractor(s.config, s.logger)
		case "suqah.com":
			storeExtractor = extractor.NewSuqahExtractor(s.config, s.logger)
		default:
			s.logger.Warnf("Unknown store: %s, skipping", store)
			continue
		}
		
		defer storeExtractor.Close()
		
		// Extract from this store
		products, err := storeExtractor.ExtractAll(ctx)
		if err != nil {
			s.logger.Warnf("Failed to extract from %s: %v", store, err)
			continue
		}
		
		// Create store result with actual store name
		storeResult := types.StoreResult{
			StoreName: store,
			Products:  products,
		}
		storeResults = append(storeResults, storeResult)
	}
	
	// Create the final result structure with separate store results
	results := &types.ExtractionResult{
		Stores: storeResults,
	}

	// Send success response
	response := APIResponse{
		Success: true,
		Data:    results,
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Errorf("Failed to encode response: %v", err)
	}
}

// sendError sends an error response
func (s *Server) sendError(w http.ResponseWriter, message string, statusCode int) {
	response := APIResponse{
		Success: false,
		Error:   message,
	}

	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Errorf("Failed to encode error response: %v", err)
	}
}

// handleHealth handles the health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// Start starts the API server
func (s *Server) Start(port string) error {
	// Setup routes
	http.HandleFunc("/extract", s.handleExtract)
	http.HandleFunc("/health", s.handleHealth)

	s.logger.Infof("Starting API server on port %s", port)
	s.logger.Info("Available endpoints:")
	s.logger.Info("  POST /extract - Extract size charts from multiple stores")
	s.logger.Info("  GET  /health  - Health check")

	return http.ListenAndServe(":"+port, nil)
}

// Close closes the server and cleanup resources
func (s *Server) Close() {
	// No cleanup needed since we create extractors per request
}

func main() {
	// Get port from environment variable, default to 8080
	serverPort := "8080"
	if envPort := os.Getenv("API_PORT"); envPort != "" {
		serverPort = envPort
		fmt.Printf("Using port from environment variable API_PORT: %s\n", serverPort)
	} else {
		fmt.Printf("No API_PORT environment variable found, using default: %s\n", serverPort)
	}

	// Create and start server
	server := NewServer()
	defer server.Close()

	// Start the server
	log.Printf("Starting API server on port %s", serverPort)
	log.Fatal(server.Start(serverPort))
} 