# /schema — CrisisEcho Database Schemas

---

## DATABASE 1 — MongoDB Main

**Env:** `MONGO_URI` / `MONGO_DB_DATABASE`

Purpose: primary document and geospatial store.

> **Index rule:** Create a `2dsphere` index on every `location` field that contains GeoJSON coordinates.

---

### Collection: `posts`

Normalized, deduplicated post after preprocessing.

```json
{
  "_id":           "ObjectId",
  "source":        "string",           // twitter | reddit | bluesky | …
  "source_id":     "string",           // original ID from source
  "text":          "string",
  "clean_text":    "string",           // after NLP preprocessing
  "author_id":     "string",
  "timestamp":     "int64",            // Unix seconds
  "location": {
    "type":        "Point",
    "coordinates": [lng, lat]          // GeoJSON — 2dsphere index
  },
  "geo_confidence": "float64",         // 0.0–1.0
  "geo_method":    "string",           // carmen | ner | geo_prior | manual
  "images":        ["string"],         // S3/GCS URLs
  "has_image":     "bool",
  "cluster_id":    "ObjectId | null",
  "crisis_id":     "ObjectId | null",
  "is_relevant":   "bool",
  "crisis_type":   "string",           // fire | flood | earthquake | …
  "severity_score": "int",             // 1–5
  "embedding_id":  "ObjectId | null",  // ref to text_embeddings
  "minhash":       "[]uint64",         // datasketch MinHash for LSH dedup
  "created_at":    "int64"
}
```

---

### Collection: `clusters`

Group of geographically and temporally proximate posts about the same event.

```json
{
  "_id":             "ObjectId",
  "centroid": {
    "type":          "Point",
    "coordinates":   [lng, lat]        // 2dsphere index
  },
  "radius_m":        "float64",
  "post_ids":        ["ObjectId"],
  "source_counts":   {"twitter": 3, "reddit": 1},
  "contributor_count": "int",
  "platform_count":  "int",
  "crisis_type":     "string",
  "severity_score":  "int",
  "first_seen":      "int64",
  "last_seen":       "int64",
  "crisis_id":       "ObjectId | null",
  "is_active":       "bool",
  "updated_at":      "int64"
}
```

---

### Collection: `crises`

Confirmed crisis event, formed from one or more clusters.

```json
{
  "_id":             "ObjectId",
  "title":           "string",
  "crisis_type":     "string",
  "severity":        "int",            // 1–5
  "cluster_ids":     ["ObjectId"],
  "bounding_box":    [[lng, lat], [lng, lat]],
  "centroid": {
    "type":          "Point",
    "coordinates":   [lng, lat]        // 2dsphere index
  },
  "summary":         "string",         // LLM-generated
  "status":          "string",         // active | resolved | monitoring
  "first_seen":      "int64",
  "last_updated":    "int64",
  "alert_sent":      "bool"
}
```

---

### Collection: `alerts`

Published alert derived from a crisis.

```json
{
  "_id":          "ObjectId",
  "crisis_id":    "ObjectId",
  "title":        "string",
  "body":         "string",
  "crisis_type":  "string",
  "severity":     "int",
  "location": {
    "type":       "Point",
    "coordinates": [lng, lat]          // 2dsphere index
  },
  "radius_m":     "float64",
  "published_at": "int64",
  "channel":      "string"             // websocket | push | sms
}
```

---

### Collection: `subscriptions`

User alert subscription for an area.

```json
{
  "_id":        "ObjectId",
  "user_id":    "string",
  "location": {
    "type":     "Point",
    "coordinates": [lng, lat]          // 2dsphere index
  },
  "radius_m":   "float64",
  "crisis_types": ["string"],          // [] = all types
  "min_severity": "int",
  "channel":    "string",              // websocket | push
  "created_at": "int64",
  "expires_at": "int64"
}
```

---

### Collection: `official_alerts`

Deduplicated official alerts from NWS, USGS, etc.

```json
{
  "_id":         "ObjectId",
  "source":      "string",             // nws | usgs | …
  "source_id":   "string",
  "title":       "string",
  "description": "string",
  "crisis_type": "string",
  "severity":    "int",
  "location": {
    "type":      "Point",
    "coordinates": [lng, lat]          // 2dsphere index
  },
  "area_polygon": "GeoJSON Polygon",
  "issued_at":   "int64",
  "expires_at":  "int64"
}
```

---

### Per-Source Raw Collections

One collection per ingestion source. All share the same base schema:

```json
{
  "_id":       "ObjectId",
  "source_id": "string",               // dedup key — unique index
  "raw":       "object",               // full original payload
  "text":      "string",
  "author_id": "string",
  "timestamp": "int64",
  "images":    ["string"],
  "fetched_at": "int64",
  "processed": "bool"
}
```

Collections: `twitter_posts`, `reddit_posts`, `bluesky_posts`, `mastodon_posts`,
`nextdoor_posts`, `telegram_posts`, `nws_alerts`, `usgs_alerts`, `gdelt_posts`,
`patch_posts`, `pulsepoint_posts`

---

## DATABASE 2 — MongoDB Location

**Env:** `MONGO_LOCATION_URI` / `MONGO_LOCATION_DB_DATABASE`

Purpose: geocoding support, geographic inference, and result caching.

> **Index rule:** 2dsphere on all coordinate fields. TTL index on `location_cache.cached_at`.

---

### Collection: `geo_priors`

Known location references used for inferring coordinates from unresolved post text.

```json
{
  "_id":             "ObjectId",
  "text":            "string",         // original mention text
  "normalized_text": "string",         // lowercased, stripped
  "location": {
    "type":          "Point",
    "coordinates":   [lng, lat]        // 2dsphere index
  },
  "confidence":      "float64",
  "source":          "string",         // carmen | manual | crowd
  "hit_count":       "int",            // times this prior was used
  "created_at":      "int64",
  "updated_at":      "int64"
}
```

---

### Collection: `place_index`

Named places: neighborhoods, intersections, zip codes, landmarks.

```json
{
  "_id":        "ObjectId",
  "name":       "string",
  "type":       "string",              // neighborhood | intersection | zip | landmark
  "zip_code":   "string",
  "location": {
    "type":     "Point",
    "coordinates": [lng, lat]          // 2dsphere index
  },
  "aliases":    ["string"],
  "bounding_box": {
    "type":     "Polygon",
    "coordinates": [[[lng, lat], …]]   // 2dsphere index
  },
  "population": "int",
  "city":       "string",
  "state":      "string",
  "country":    "string"
}
```

---

### Collection: `location_cache`

Cached results of Carmen/NER geocoding keyed by text hash (prevents recomputation).

```json
{
  "_id":           "ObjectId",
  "text_hash":     "string",           // SHA-256 of normalized input text — unique index
  "original_text": "string",
  "location": {
    "type":        "Point",
    "coordinates": [lng, lat]          // 2dsphere index
  },
  "confidence":    "float64",
  "provider":      "string",           // carmen | ner
  "cached_at":     "int64"             // Unix — TTL index (e.g. 7 days)
}
```

TTL index: `db.location_cache.createIndex({ "cached_at": 1 }, { expireAfterSeconds: 604800 })`

---

## DATABASE 3 — MongoDB Vector

**Env:** `MONGO_VECTOR_URI` / `MONGO_VECTOR_DB_DATABASE`

Purpose: Atlas Vector Search — semantic similarity retrieval using `$vectorSearch`.

---

### Collection: `text_embeddings`

One document per processed post. 1024-dimensional Voyage AI vector.

```json
{
  "_id":           "ObjectId",
  "post_id":       "string",           // ref to posts._id — unique index
  "source":        "string",
  "vector":        [0.123, -0.456, …], // float64[1024] — vector search index
  "timestamp":     "int64",            // Unix — filter field
  "lat":           "float64",          // filter field
  "lng":           "float64",          // filter field
  "crisis_type":   "string",           // filter field
  "severity_score": "int",
  "is_relevant":   "bool"              // filter field
}
```

---

### Collection: `image_embeddings`

One document per post image. 512-dimensional CLIP vector.

```json
{
  "_id":       "ObjectId",
  "post_id":   "string",               // ref to posts._id
  "image_url": "string",               // S3/GCS URL
  "source":    "string",
  "vector":    [0.123, -0.456, …],     // float64[512] — vector search index
  "timestamp": "int64",                // filter field
  "lat":       "float64",              // filter field
  "lng":       "float64"               // filter field
}
```

---

## Atlas Vector Search Index Definitions

Run `scripts/create_vector_indexes.js` once against the vector database.

### `text_vector_index` on `text_embeddings`

```json
{
  "name": "text_vector_index",
  "type": "vectorSearch",
  "definition": {
    "fields": [
      { "type": "vector", "path": "vector", "numDimensions": 1024, "similarity": "cosine" },
      { "type": "filter", "path": "timestamp" },
      { "type": "filter", "path": "lat" },
      { "type": "filter", "path": "lng" },
      { "type": "filter", "path": "is_relevant" },
      { "type": "filter", "path": "crisis_type" }
    ]
  }
}
```

### `image_vector_index` on `image_embeddings`

```json
{
  "name": "image_vector_index",
  "type": "vectorSearch",
  "definition": {
    "fields": [
      { "type": "vector", "path": "vector", "numDimensions": 512, "similarity": "cosine" },
      { "type": "filter", "path": "lat" },
      { "type": "filter", "path": "lng" },
      { "type": "filter", "path": "timestamp" }
    ]
  }
}
```

### `$vectorSearch` Aggregation Pipeline Shape

```go
pipeline := mongo.Pipeline{
    bson.D{{"$vectorSearch", bson.D{
        {"index", "text_vector_index"},
        {"path", "vector"},
        {"queryVector", queryVector},
        {"numCandidates", 200},
        {"limit", limit},
        {"filter", bson.D{
            {"timestamp", bson.D{{"$gt", filter.MinTimestamp}}},
            {"lat", bson.D{{"$gte", filter.MinLat}, {"$lte", filter.MaxLat}}},
            {"lng", bson.D{{"$gte", filter.MinLng}, {"$lte", filter.MaxLng}}},
        }},
    }}},
    bson.D{{"$project", bson.D{
        {"post_id", 1}, {"source", 1}, {"lat", 1}, {"lng", 1},
        {"crisis_type", 1}, {"score", bson.D{{"$meta", "vectorSearchScore"}}},
    }}},
}
```
