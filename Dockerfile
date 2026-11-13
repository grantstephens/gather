# Frontend stage. Output is architecture-independent, so it always runs on the
# build host's own platform. Only rebuilt when something under frontend/ changes.
FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend

WORKDIR /build/frontend

# Cache npm install (keyed on package.json only)
COPY frontend/package*.json ./
RUN npm install

COPY frontend/ ./
RUN npm run build

# Backend stage (needs CGO for libwebp)
FROM golang:1.27-alpine AS builder

RUN apk add --no-cache \
    gcc \
    musl-dev \
    libwebp-dev

WORKDIR /build

# Cache Go modules
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source, then the built frontend so the binary embeds it
COPY . .
COPY --from=frontend /build/frontend/dist ./frontend/dist

# No -a: let Go reuse its build cache instead of recompiling the std lib and every dependency
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o gather .

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
