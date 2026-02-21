package main

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// GCSClient wraps the Google Cloud Storage client
type GCSClient struct {
	client     *storage.Client
	bucketName string
}

// NewGCSClient creates a new GCS client with service account credentials
func NewGCSClient(ctx context.Context, bucketName, credentialsPath string) (*GCSClient, error) {
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return &GCSClient{
		client:     client,
		bucketName: bucketName,
	}, nil
}

func (g *GCSClient) GenerateV4PutObjectSignedURL(w io.Writer, object, contentType string) (string, error) {
	// object := "object-name"

	// Signing a URL requires credentials authorized to sign a URL. You can pass
	// these in through SignedURLOptions with one of the following options:
	//    a. a Google service account private key, obtainable from the Google Developers Console
	//    b. a Google Access ID with iam.serviceAccounts.signBlob permissions
	//    c. a SignBytes function implementing custom signing.
	// In this example, none of these options are used, which means the SignedURL
	// function attempts to use the same authentication that was used to instantiate
	// the Storage client. This authentication must include a private key or have
	// iam.serviceAccounts.signBlob permissions.
	opts := &storage.SignedURLOptions{
		Scheme: storage.SigningSchemeV4,
		Method: "PUT",
		Headers: []string{
			fmt.Sprintf("Content-Type:%s", contentType),
		},
		Expires: time.Now().Add(15 * time.Minute), // 15 minutes is usually enough
	}

	u, err := g.client.Bucket(g.bucketName).SignedURL(object, opts)
	if err != nil {
		return "", fmt.Errorf("Bucket(%q).SignedURL: %w", g.bucketName, err)
	}

	fmt.Fprintln(w, "Generated PUT signed URL:")
	fmt.Fprintf(w, "%q\n", u)
	fmt.Fprintln(w, "You can use this URL with any user agent, for example:")
	fmt.Fprintf(w, "curl -X PUT -H 'Content-Type: %s' --upload-file my-file %q\n", contentType, u)
	return u, nil
}

// UploadImage uploads an image file to GCS and returns the public URL
func (g *GCSClient) UploadImage(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error) {
	// Generate unique filename with timestamp
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%d-%s%s", time.Now().Unix(), sanitizeFilename(header.Filename[:len(header.Filename)-len(ext)]), ext)

	// Create object handle
	obj := g.client.Bucket(g.bucketName).Object(filename)
	
	// Create writer
	writer := obj.NewWriter(ctx)
	
	// Set content type based on file extension
	writer.ContentType = getContentType(ext)


	// Copy file content to GCS
	if _, err := io.Copy(writer, file); err != nil {
		writer.Close()
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	// Return public URL
	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", g.bucketName, filename)
	return publicURL, nil
}

// Close closes the GCS client
func (g *GCSClient) Close() error {
	return g.client.Close()
}

// sanitizeFilename removes special characters from filename
func sanitizeFilename(filename string) string {
	// Simple sanitization - you might want to enhance this
	return filepath.Base(filename)
}

// getContentType returns the content type based on file extension
func getContentType(ext string) string {
	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".bmp":  "image/bmp",
		".svg":  "image/svg+xml",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// ConfigureCORS updates the CORS configuration for the bucket
func (g *GCSClient) ConfigureCORS(ctx context.Context, origins []string) error {
	bucket := g.client.Bucket(g.bucketName)

	cors := []storage.CORS{
		{
			MaxAge:          time.Hour,
			Methods:         []string{"GET", "HEAD", "PUT", "OPTIONS", "DELETE"},
			Origins:         origins,
			ResponseHeaders: []string{"Content-Type", "Access-Control-Allow-Origin", "X-Requested-With"},
		},
	}

	attrs := storage.BucketAttrsToUpdate{
		CORS: cors,
	}

	if _, err := bucket.Update(ctx, attrs); err != nil {
		return fmt.Errorf("failed to update bucket CORS: %w", err)
	}

	return nil
}
