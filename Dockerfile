# =============================================================================
# PRODUCTION DOCKERFILE
# Multi-stage build for minimal final image
# =============================================================================

# The two build stages run on the machine doing the build ($BUILDPLATFORM), not
# under emulation for each target: the frontend bundle is the same for every
# architecture, and Go cross-compiles. Under QEMU, npm ci for arm64 could hang
# for the better part of an hour.

# Stage 1: Build frontend
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-builder

WORKDIR /app/web

# Install dependencies first (cache layer)
COPY web/package*.json ./
RUN npm ci --production=false

# Build frontend
COPY web/ ./
RUN npm run build

# Stage 2: Build backend
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS backend-builder

# Set by buildx for each image it builds; a plain docker build fills them in
# for the machine it runs on.
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Install dependencies first (cache layer)
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Copy built frontend for embedding
COPY --from=frontend-builder /app/web/dist ./web/dist

# Build optimized binary for the image's architecture. This was fixed to amd64,
# which put an x86-64 binary into the arm64 image as well.
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="-w -s -X main.Version=$(date +%Y%m%d)" \
    -o /jellyshare ./cmd/server

# Stage 3: Final minimal image
FROM alpine:3.20

LABEL org.opencontainers.image.title="Jellyshare"
LABEL org.opencontainers.image.description="Jellyshare - one-time share links for Jellyfin"
LABEL org.opencontainers.image.source="https://github.com/melth-cmd/jellyshare"

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=backend-builder /jellyshare .

# Copy migrations for embedded FS
COPY --from=backend-builder /app/migrations ./migrations

# Copy frontend dist for embedded FS
COPY --from=frontend-builder /app/web/dist ./web/dist

# Create non-root user
RUN addgroup -g 1000 jellyshare && \
    adduser -D -u 1000 -G jellyshare jellyshare && \
    chown -R jellyshare:jellyshare /app

USER jellyshare

EXPOSE 8097

# Follow JELLYSHARE_PORT rather than hard-coding it: a fixed port here left the
# container permanently unhealthy whenever the port was overridden.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider "http://localhost:${JELLYSHARE_PORT:-8097}/health" || exit 1

ENTRYPOINT ["/app/jellyshare"]
