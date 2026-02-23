/**
 * create_geo_indexes.js
 *
 * Creates all 2dsphere indexes required for geospatial queries in CrisisEcho.
 * Run with:
 *   mongosh "<MONGO_URI>" scripts/create_geo_indexes.js
 *
 * Collections in the main DB require MONGO_DB_DATABASE to be set below.
 */

// ── Configuration ──────────────────────────────────────────────────────────
// Adjust these to match your environment variables if running interactively.
const MAIN_DB        = process.env.MONGO_DB_DATABASE   || "crisisecho";
const LOCATION_DB    = process.env.MONGO_LOCATION_DB_DATABASE || "crisisecho_location";

// ── Helper ─────────────────────────────────────────────────────────────────
function createGeoIndex(db, collectionName, fieldPath, extra) {
  const index = {};
  index[fieldPath] = "2dsphere";
  Object.assign(index, extra || {});
  try {
    const result = db.getCollection(collectionName).createIndex(index, { background: true });
    print(`  ✓ ${collectionName}.${fieldPath} → ${JSON.stringify(result)}`);
  } catch (e) {
    print(`  ✗ ${collectionName}.${fieldPath} → ${e.message}`);
  }
}

// ── Main DB ────────────────────────────────────────────────────────────────
print(`\n[main DB: ${MAIN_DB}]`);
const mainDB = db.getSiblingDB(MAIN_DB);

// unified posts
createGeoIndex(mainDB, "posts", "location");

// clusters — centroid + affected_area polygon
createGeoIndex(mainDB, "clusters", "centroid");
createGeoIndex(mainDB, "clusters", "affected_area");

// crises
createGeoIndex(mainDB, "crises", "location");

// alerts
createGeoIndex(mainDB, "alerts", "centroid");

// subscriptions
createGeoIndex(mainDB, "subscriptions", "location");

// official_alerts (NWS/USGS raw)
createGeoIndex(mainDB, "official_alerts", "location");

// per-platform raw source collections
const sourceCollections = [
  "twitter_posts",
  "reddit_posts",
  "bluesky_posts",
  "mastodon_posts",
  "nextdoor_posts",
  "telegram_posts",
  "nws_alerts",
  "usgs_alerts",
  "gdelt_posts",
  "patch_posts",
  "pulsepoint_posts",
];
for (const coll of sourceCollections) {
  createGeoIndex(mainDB, coll, "location");
}

// ── Location DB ────────────────────────────────────────────────────────────
print(`\n[location DB: ${LOCATION_DB}]`);
const locationDB = db.getSiblingDB(LOCATION_DB);

// geo_priors — used by $nearSphere in FindNearbyPriors
createGeoIndex(locationDB, "geo_priors", "coordinates");

// place_index — bounding-box centroid
createGeoIndex(locationDB, "place_index", "coordinates");

// location_cache — optional; speeds up nearby-cache lookups
createGeoIndex(locationDB, "location_cache", "coordinates");

print("\nDone. All 2dsphere indexes created (or already exist).");
