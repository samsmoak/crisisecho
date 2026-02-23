/**
 * create_vector_indexes.js
 *
 * One-time Atlas Vector Search index setup script.
 * Run this against your vector database ONCE before the application starts.
 *
 * Usage:
 *   mongosh "<MONGO_VECTOR_URI>" --eval "use <MONGO_VECTOR_DB_DATABASE>" scripts/create_vector_indexes.js
 *
 * Or interactively:
 *   mongosh "<MONGO_VECTOR_URI>"
 *   > use <your-vector-db-name>
 *   > load("scripts/create_vector_indexes.js")
 *
 * Note: createSearchIndex() is an async Atlas API call. The index will show
 * status "BUILDING" and transition to "READY" within a few minutes.
 * The application's $vectorSearch queries will only succeed after the index
 * reaches READY status.
 */

// ─── text_vector_index on text_embeddings ────────────────────────────────────
// 1024-dimensional Voyage AI embeddings with cosine similarity.
// Filter fields: timestamp, lat, lng, is_relevant, crisis_type.

print("Creating text_vector_index on text_embeddings…");

db.text_embeddings.createSearchIndex({
  name: "text_vector_index",
  type: "vectorSearch",
  definition: {
    fields: [
      {
        type: "vector",
        path: "vector",
        numDimensions: 1024,
        similarity: "cosine",
      },
      { type: "filter", path: "timestamp" },
      { type: "filter", path: "lat" },
      { type: "filter", path: "lng" },
      { type: "filter", path: "is_relevant" },
      { type: "filter", path: "crisis_type" },
    ],
  },
});

print("text_vector_index creation request submitted.");

// ─── image_vector_index on image_embeddings ───────────────────────────────────
// 512-dimensional CLIP embeddings with cosine similarity.
// Filter fields: lat, lng, timestamp.

print("Creating image_vector_index on image_embeddings…");

db.image_embeddings.createSearchIndex({
  name: "image_vector_index",
  type: "vectorSearch",
  definition: {
    fields: [
      {
        type: "vector",
        path: "vector",
        numDimensions: 512,
        similarity: "cosine",
      },
      { type: "filter", path: "lat" },
      { type: "filter", path: "lng" },
      { type: "filter", path: "timestamp" },
    ],
  },
});

print("image_vector_index creation request submitted.");

// ─── Verify indexes exist ─────────────────────────────────────────────────────

print("\nCurrent search indexes on text_embeddings:");
db.text_embeddings.listSearchIndexes().forEach((idx) => {
  print("  " + idx.name + " — status: " + idx.status);
});

print("\nCurrent search indexes on image_embeddings:");
db.image_embeddings.listSearchIndexes().forEach((idx) => {
  print("  " + idx.name + " — status: " + idx.status);
});

print(
  "\nDone. Indexes will transition from BUILDING → READY within a few minutes."
);
print(
  "Run db.<collection>.listSearchIndexes() to monitor progress."
);
