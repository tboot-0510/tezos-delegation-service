# Use the official Go image as the base image
FROM golang:1.24-alpine AS builder

# Install build dependencies for CGO
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./

# download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Enable CGO and build with cgo enabled
ENV CGO_ENABLED=1
RUN go build -o app ./main.go

# Use a minimal alpine image for the final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Install sqlite3 for the database
RUN apk --no-cache add sqlite

# Create a non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/app .

# Create a directory for the database
RUN mkdir -p /app/data && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port 3000
EXPOSE 3000

# Run the application
CMD ["./app"]