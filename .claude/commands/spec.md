# /spec — CrisisEcho Project Spec

## Overview

CrisisEcho is a real-time hyperlocal crisis detection platform that aggregates social media posts, official alerts, and sensor data to detect, cluster, and surface local emergencies with geographic precision.

---

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend API | Go + Fiber |
| AI Pipeline | Python 3.11 + LangChain |
| Embeddings — Text | Voyage AI (1024-dim) |
| Embeddings — Image | CLIP (512-dim) |
| Frontend | React 18 + TypeScript + Leaflet.js + Recharts |
| Main DB | MongoDB Atlas M0 (geospatial + documents) |
| Location DB | MongoDB Atlas M0 (geocoding support + geo priors) |
| Vector DB | MongoDB Atlas Vector Search (`$vectorSearch` pipeline) |
| Cache / Pub-Sub | Redis 7 |
| Queue | Apache Kafka |
| NLP | spaCy + datasketch MinHash LSH |
| Geocoding | Carmen + spaCy NER (cascade) |
| LLM | Claude claude-haiku-4-5 (primary) / Ollama Llama3 (fallback) |
| Image Store | AWS S3 or GCS |
| Deployment | Docker Compose (local) → GCP Cloud Run (prod) |

---

## Go API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/hotspots?lat=&lng=&radius=` | Return active crisis clusters near coordinates |
| `GET` | `/api/pin?lat=&lng=` | Return crisis detail at a specific pin location |
| `GET` | `/api/query` | Semantic search over embeddings + cluster metadata |
| `POST` | `/api/subscribe` | Subscribe to real-time alerts for an area |
| `WS` | `/ws/alerts` | WebSocket stream of live alerts (Redis Pub/Sub) |

---

## Alert Publish Rules

An alert is published when ALL of the following thresholds are met:

1. **Severity** ≥ 3 (on a 1–5 scale)
2. **Independent contributors** ≥ 3 (distinct user accounts)
3. **Platform diversity** ≥ 2 distinct source platforms
4. **Geocoding confidence** above the configured threshold

---

## Domain Build Order

Build domains in this order (Prompt 2 onward):

```
post → cluster → crisis → alert → ingest → preprocess → rag → notify → query
```

Within each domain, build layers in this order:

```
model → repository → service → controller
```

---

## Data Sources

### Social Media (raw ingestion collections)
- `twitter_posts`
- `reddit_posts`
- `bluesky_posts`
- `mastodon_posts`
- `nextdoor_posts`
- `telegram_posts`

### Official / Authoritative
- `nws_alerts` (National Weather Service)
- `usgs_alerts` (US Geological Survey — earthquakes)
- `gdelt_posts` (GDELT global events)
- `patch_posts` (Patch local news)
- `pulsepoint_posts` (PulsePoint emergency incidents)

---

## Environment Variables

### Main DB
```
MONGO_URI
MONGO_DB_DATABASE
```

### Location DB
```
MONGO_LOCATION_URI
MONGO_LOCATION_DB_DATABASE
```

### Vector DB
```
MONGO_VECTOR_URI
MONGO_VECTOR_DB_DATABASE
```

### Redis
```
REDIS_URL
```

### API Keys
```
VOYAGE_API_KEY
ANTHROPIC_API_KEY
AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY
S3_BUCKET
JWT_SECRET
```

---

## Local Dev Note

For local development, all three databases can run on the same `mongo:7` container, separated by database name only:

```yaml
# docker-compose.yml (local)
services:
  mongo:
    image: mongo:7
    ports:
      - "27017:27017"
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

In production, each maps to an Atlas M0 cluster or a single Atlas cluster with separate databases.
