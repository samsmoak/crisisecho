# ─── CrisisEcho Python Sidecar ───────────────────────────────────────────────
# Build context: project root (set in docker-compose.yml)
# so that both sidecar/ source and certs/ are reachable.
FROM python:3.12-slim

# System packages needed by Python ML libraries
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      libopenblas-dev \
      ca-certificates \
      curl && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

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
      "uvicorn[standard]" \
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

# Copy sidecar source (build context is project root)
COPY sidecar/ /app/

# Copy Aiven TLS certificates
COPY certs/ /app/certs/

# Run as non-root user
RUN useradd --system --no-create-home --shell /usr/sbin/nologin appuser && \
    chown -R appuser:appuser /app
USER appuser

EXPOSE 8081

CMD ["python", "pipeline.py"]
