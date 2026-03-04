# ─── CrisisEcho Go API ───────────────────────────────────────────────────────

# ─── Stage 1: Build Go binary ────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Cache module downloads separately from source
COPY go.mod go.sum* ./
RUN go mod download

# Copy source and build a fully-static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
      -ldflags '-s -w' \
      -tags netgo \
      -o crisisecho \
      ./cmd/api

# ─── Stage 2: Runtime ─────────────────────────────────────────────────────────
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      curl && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy Go binary
COPY --from=builder /build/crisisecho /app/crisisecho

# Copy Aiven TLS certificates (ca.pem, service.cert, service.key)
COPY certs/ /app/certs/

# Run as non-root user
RUN useradd --system --no-create-home --shell /usr/sbin/nologin appuser && \
    chown -R appuser:appuser /app
USER appuser

EXPOSE 8080

CMD ["/app/crisisecho"]
