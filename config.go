package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	BucketName1          string
	ServiceAccountPath1  string
	BucketName2          string
	ServiceAccountPath2  string
	Port                string
	MaxFileSize         int64 // in bytes
	APIKey1              string
	APIKey2             string
	AllowedIPs          []string
	AllowedOrigins      []string
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables or defaults")
	}

	maxFileSizeInt, _ := strconv.Atoi(getEnv("MAX_FILE_SIZE_MB", "10"))
	maxFileSize := int64(maxFileSizeInt)
	
	// Parse comma-separated IPs
	allowedIPsStr := getEnv("ALLOWED_IPS", "")
	var allowedIPs []string
	if allowedIPsStr != "" {
		allowedIPs = strings.Split(allowedIPsStr, ",")
		for i := range allowedIPs {
			allowedIPs[i] = strings.TrimSpace(allowedIPs[i])
		}
	}
	
	// Parse comma-separated origins
	allowedOriginsStr := getEnv("ALLOWED_ORIGINS", "*")
	allowedOrigins := strings.Split(allowedOriginsStr, ",")
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}
	
	config := &Config{
		BucketName1:         getEnv("GCS_BUCKET_NAME_1", ""),
		ServiceAccountPath1: getEnv("GCS_AUTH_1", "./service-account-key.json"),
		BucketName2:         getEnv("GCS_BUCKET_NAME_2", ""),
		ServiceAccountPath2: getEnv("GCS_AUTH_2", ""),
		Port:               getEnv("PORT", "8080"),
		MaxFileSize:        maxFileSize * 1024 * 1024,
		APIKey1:            getEnv("GCS_API_KEY_1", ""),
		APIKey2:            getEnv("GCS_API_KEY_2", ""),
		AllowedIPs:         allowedIPs,
		AllowedOrigins:     allowedOrigins,
	}

	return config
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
