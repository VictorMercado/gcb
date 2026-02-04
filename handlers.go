package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	// "path/filepath"
	"log"
	"strings"
)

// Response structures
type UploadResponse struct {
	Success   bool   `json:"success"`
	URL       string `json:"url,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// HandleHealth returns a simple health check response
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status:  "healthy",
		Message: "GCS Image Upload Service is running",
	})
}

// HandleUpload handles image upload requests
func HandleUpload(gcsClient *GCSClient, config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Only allow POST method
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(UploadResponse{
				Success: false,
				Error:   "Method not allowed. Use POST.",
			})
			return
		}

		// Parse multipart form
		if err := r.ParseMultipartForm(config.MaxFileSize); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(UploadResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to parse form: %v", err),
			})
			return
		}

		// Get the file from form data
		file, header, err := r.FormFile("image")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(UploadResponse{
				Success: false,
				Error:   "No image file provided. Use 'image' as the form field name.",
			})
			return
		}
		defer file.Close()

		// Validate file size
		if header.Size > config.MaxFileSize {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(UploadResponse{
				Success: false,
				Error:   fmt.Sprintf("File too large. Max size: %d MB", config.MaxFileSize/(1024*1024)),
			})
			return
		}

		// Validate file type
		if !isValidImageType(header.Filename) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(UploadResponse{
				Success: false,
				Error:   "Invalid file type. Allowed: jpg, jpeg, png, gif, webp, bmp, svg",
			})
			return
		}

		// Upload to GCS
		url, err := gcsClient.UploadImage(r.Context(), file, header)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(UploadResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to upload image: %v", err),
			})
			return
		}

		// Success response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UploadResponse{
			Success: true,
			URL:     url,
			Message: "Image uploaded successfully",
		})
	}
}

type SignedUrlRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
}

// HandleGenerateSignedUrl handles requests to generate a signed URL for direct upload
func HandleGenerateSignedUrl(gcsClient *GCSClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(UploadResponse{
				Success: false,
				Error:   "Method not allowed. Use POST.",
			})
			return
		}

		var req SignedUrlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(UploadResponse{
				Success: false,
				Error:   "Invalid request body",
			})
			return
		}

		if req.Filename == "" || req.ContentType == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(UploadResponse{
				Success: false,
				Error:   "Filename and ContentType are required",
			})
			return
		}

		if !isValidImageType(req.Filename) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(UploadResponse{
				Success: false,
				Error:   "Invalid file type",
			})
			return
		}

		// Generate a unique filename effectively
		// ext := filepath.Ext(req.Filename)
		// filename := fmt.Sprintf("%d-%s%s", time.Now().Unix(), sanitizeFilename(req.Filename[:len(req.Filename)-len(ext)]), ext)
		// filename := fmt.Sprintf("%s", req.Filename)
		log.Println("Filename: " + req.Filename)
		// Create a pipe to capture the output (since the existing method writes to io.Writer)
		// But wait, the method returns the URL string too. The io.Writer is just for logging/debugging in the example.
		// We can just pass io.Discard if we don't want the output, or a buffer if we do.
		// However, looking at the previous file content, the method signature I updated is:
		// func (g *GCSClient) GenerateV4PutObjectSignedURL(w io.Writer, object, contentType string) (string, error)
		
		url, err := gcsClient.GenerateV4PutObjectSignedURL(io.Discard, req.Filename, req.ContentType)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(UploadResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to generate signed URL: %v", err),
			})
			return
		}

		// Increment signed URL counter with hostname and client IP
		hostname := r.Host
		clientIP := getClientIP(r)
		IncrementSignedURLCounter(hostname, clientIP)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UploadResponse{
			Success: true,
			URL:     url,
			Message: "Signed URL generated successfully",
		})
	}
}

// isValidImageType checks if the file has a valid image extension
func isValidImageType(filename string) bool {
	validExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg"}
	filename = strings.ToLower(filename)
	
	for _, ext := range validExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}
