# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies for libwebp and Node.js
RUN apk add --no-cache \
    gcc \
    musl-dev \
    libwebp-dev \
    nodejs \
    npm

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy frontend files and build
COPY frontend ./frontend
RUN cd frontend && npm install && npm run build

# Copy Go source and build binary
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags="-s -w" -o gather .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies for libwebp
RUN apk add --no-cache \
    libwebp \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1000 gather && \
    adduser -D -u 1000 -G gather gather

WORKDIR /app

# Copy binary and entrypoint from builder
COPY --from=builder /build/gather .
COPY docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh

# Create data directory with proper permissions
RUN mkdir -p pb_data && chown -R gather:gather /app

USER gather

EXPOSE 8090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8090/api/health || exit 1

ENTRYPOINT ["./docker-entrypoint.sh"]
