# ─── Stage 1: Build Go binary ────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Cache module downloads separately from source
# COPY go.mod go.sum ./
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

# System packages needed by Python ML libraries
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      python3 \
      python3-pip \
      python3-venv \
      libopenblas-dev \
      ca-certificates \
      curl && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Create Python virtual environment
RUN python3 -m venv /app/venv
ENV PATH="/app/venv/bin:$PATH"

# Install Python dependencies
RUN pip install --no-cache-dir \
      # LangChain + LLM providers
      langchain \
      langchain-anthropic \
      langchain-community \
      # Embeddings
      sentence-transformers \
      voyageai \
      open-clip-torch \
      # Kafka
      kafka-python \
      # MongoDB
      pymongo \
      motor \
      # ML / NLP
      transformers \
      torch \
      spacy \
      datasketch \
      pillow \
      # Cloud
      boto3 \
      # Scheduling + HTTP server
      apscheduler \
      fastapi \
      uvicorn[standard] \
      # Ingestion platform SDKs
      tweepy \
      praw \
      atproto \
      websocket-client \
      feedparser \
      requests \
      # Redis
      redis

# Download spaCy English model
RUN python -m spacy download en_core_web_sm

# Pre-warm the DistilBERT model to avoid cold start at runtime
RUN python -c "\
from transformers import pipeline; \
pipeline('text-classification', model='cross-encoder/nli-distilroberta-base', device=-1); \
print('DistilBERT model cached.')"

# Copy Go binary
COPY --from=builder /build/crisisecho /app/crisisecho

# Copy Python AI scripts
COPY internal/ai/ /app/internal/ai/

# Copy Aiven TLS certificates (ca.pem, service.cert, service.key)
COPY certs/ /app/certs/

# Copy entrypoint
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Run as non-root user
RUN useradd --system --no-create-home --shell /usr/sbin/nologin appuser && \
    chown -R appuser:appuser /app
USER appuser

EXPOSE 8080 8081

ENTRYPOINT ["/app/entrypoint.sh"]
