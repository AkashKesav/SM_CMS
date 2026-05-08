# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

# Install certificates for HTTPS requests to Supabase
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/main .
# Copy migration file if needed
COPY --from=builder /app/migration.sql .

# Expose the port the app runs on
EXPOSE 8081

# Run the application
CMD ["./main"]
