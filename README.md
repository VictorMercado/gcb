# GCS Image Upload Service 🚀

A simple Go service for uploading images to Google Cloud Storage.

## Quick Start

1. **Configure environment variables:**
   ```bash
   # Edit .env file with your bucket name
   GCS_BUCKET_NAME=your-bucket-name
   ```

2. **Run the server:**
   ```bash
   go run .
   ```

3. **Test the service:**
   - Open `test.html` in your browser for a visual interface
   - Or use the cURL commands below

## API Endpoints

### Health Check
```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "gcs-image-upload"
}
```

### Upload Image

**Using cURL:**
```bash
# Upload an image
curl -X POST http://localhost:8080/upload \
  -F "image=@/path/to/your/image.jpg"
```

**Create a test image and upload:**
```bash
# Create a test image (macOS)
curl -o test-image.jpg https://picsum.photos/800/600

# Upload it
curl -X POST http://localhost:8080/upload \
  -F "image=@test-image.jpg"
```

**Success Response:**
```json
{
  "url": "https://storage.googleapis.com/your-bucket/uploads/abc123_image.jpg",
  "filename": "abc123_image.jpg",
  "size": 245670,
  "content_type": "image/jpeg"
}
```

**Error Response:**
```json
{
  "error": "description of error"
}
```

## Testing with HTML

Open `test.html` in your browser for a beautiful drag-and-drop interface to test uploads.

## Configuration

Environment variables (set in `.env`):

- `GCS_BUCKET_NAME` - **Required**. Your GCS bucket name
- `GOOGLE_APPLICATION_CREDENTIALS` - Path to service account key (default: `./service-account-key.json`)
- `PORT` - Server port (default: `8080`)

## Supported File Types

- JPEG/JPG
- PNG
- GIF
- WebP

**Maximum file size:** 10MB

## Architecture

```
├── main.go        - Server setup and routing
├── config.go      - Configuration management
├── handlers.go    - HTTP request handlers
├── gcs.go         - Google Cloud Storage client
├── .env           - Environment variables
└── test.html      - Testing interface
```
