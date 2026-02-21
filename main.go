package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Load configuration
	config := LoadConfig()

	// Validate bucket name
	if config.BucketName1 == "" {
		log.Fatal("GCS_BUCKET_NAME_1 environment variable is required")
	}

	// Check if service account file exists
	if _, err := os.Stat(config.ServiceAccountPath1); os.IsNotExist(err) {
		log.Fatalf("Service account file not found at: %s\nPlease place your service-account-key.json file in the project root.", config.ServiceAccountPath1)
	}

	// Create context
	ctx := context.Background()

	// Initialize GCS client
	darlingimagesClientProd, err := NewGCSClient(ctx, config.BucketName1, config.ServiceAccountPath1)
	if err != nil {
		log.Fatalf("Failed to initialize GCS client: %v", err)
	}
	defer darlingimagesClientProd.Close()

	// Configure CORS for the bucket
	log.Printf("⚙️  Configuring CORS for bucket %s with origins: %v", config.BucketName1, config.AllowedOrigins)
	if err := darlingimagesClientProd.ConfigureCORS(ctx, config.AllowedOrigins); err != nil {
		log.Printf("⚠️  Warning: Failed to configure bucket CORS: %v", err)
		log.Println("   Uploads from browser might fail if CORS is not already configured correctly.")
	} else {
		log.Println("✅ Bucket CORS configured successfully")
	}
	
	// Initialize GCS client
	darlingimagesClientDev, err := NewGCSClient(ctx, config.BucketName2, config.ServiceAccountPath1)
	if err != nil {
		log.Fatalf("Failed to initialize GCS client: %v", err)
	}
	defer darlingimagesClientDev.Close()

	// Configure CORS for the bucket
	log.Printf("⚙️  Configuring CORS for bucket %s with origins: %v", config.BucketName2, config.AllowedOrigins)
	if err := darlingimagesClientDev.ConfigureCORS(ctx, config.AllowedOrigins); err != nil {
		log.Printf("⚠️  Warning: Failed to configure bucket CORS: %v", err)
		log.Println("   Uploads from browser might fail if CORS is not already configured correctly.")
	} else {
		log.Println("✅ Bucket CORS configured successfully")
	}

	// Apply authentication middleware (only to /upload endpoint)
	authenticatedMux := http.NewServeMux()
	authenticatedMux.HandleFunc("/health", HandleHealth)
	authenticatedMux.Handle("/metrics", promhttp.Handler())
	
	// Only apply auth middleware if API key is configured
	if config.APIKey1 != "" {
		log.Println("🔒 Authentication enabled")
		if len(config.AllowedIPs) > 0 {
			log.Printf("🔒 IP Whitelist enabled: %v", config.AllowedIPs)
		}
		authenticatedMux.Handle("/upload", AuthMiddleware(config.APIKey1, config.AllowedIPs)(http.HandlerFunc(HandleUpload(darlingimagesClientProd, config))))
		authenticatedMux.Handle("/signedurl", AuthMiddleware(config.APIKey1, config.AllowedIPs)(http.HandlerFunc(HandleGenerateSignedUrl(darlingimagesClientProd))))
		authenticatedMux.Handle("/upload-dev", AuthMiddleware(config.APIKey1, config.AllowedIPs)(http.HandlerFunc(HandleUpload(darlingimagesClientDev, config))))
		authenticatedMux.Handle("/signedurl-dev", AuthMiddleware(config.APIKey1, config.AllowedIPs)(http.HandlerFunc(HandleGenerateSignedUrl(darlingimagesClientDev))))
	} else {
		log.Println("⚠️  WARNING: No API key configured - authentication disabled!")
		authenticatedMux.HandleFunc("/upload", HandleUpload(darlingimagesClientProd, config))
	}
	
	// Apply CORS and Metrics middleware
	var handler http.Handler = MetricsMiddleware(CORSMiddleware(config.AllowedOrigins)(authenticatedMux))

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%s", config.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 Server starting on port %s", config.Port)
		log.Printf("📦 Bucket: %s", config.BucketName1)
		log.Printf("🔐 Authentication: %s", func() string {
			if config.APIKey1 != "" {
				return "Enabled"
			}
			return "Disabled"
		}())
		log.Printf("📝 Endpoints:")
		log.Printf("   - GET  http://localhost:%s/health", config.Port)
		log.Printf("   - POST http://localhost:%s/upload", config.Port)
		log.Printf("   - GET  http://localhost:%s/metrics", config.Port)
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel

	log.Println("🛑 Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server stopped gracefully")
}
