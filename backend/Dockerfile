# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies with checksum verification disabled for local dev
# This is safe because we have go.sum file which will still verify checksums
ENV GOSUMDB=off
RUN go mod download

# Copy source code
COPY . .

# Build the application
# Keep GOSUMDB=off for build step too
RUN CGO_ENABLED=0 GOOS=linux GOSUMDB=off go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates and tzdata for timezone support
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./main"]
