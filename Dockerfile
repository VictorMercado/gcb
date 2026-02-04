# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git for fetching dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gcloud-image-upload .

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS requests to GCS
RUN apk --no-cache add ca-certificates

# Copy the binary from builder
COPY --from=builder /app/gcloud-image-upload .

# Expose the port
EXPOSE 8080

# Run the binary
CMD ["./gcloud-image-upload"]
