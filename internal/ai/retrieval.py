"""
retrieval.py — Hybrid retrieval module for CrisisEcho.

Combines Atlas Vector Search, MongoDB $near geo-proximity, official signal
lookup, and location DB enrichment into a single ranked RetrievalResult.
"""

import hashlib
import logging
import math
import os
from dataclasses import dataclass, field
from datetime import datetime, timezone, timedelta
from typing import Any

import pymongo
import requests

logger = logging.getLogger(__name__)

VOYAGE_API_KEY  = os.environ.get("VOYAGE_API_KEY", "")
VOYAGE_API_URL  = "https://api.voyageai.com/v1/embeddings"
VOYAGE_MODEL    = "voyage-2"
SEARCH_RADIUS_M = 50_000          # 50 km for $near
BBOX_DEG        = 0.5             # ±0.5° for vector search bounding box filter
LOOKBACK_S      = 7_200           # 2-hour window
MAX_GEO_RESULTS = 200

# Source authority weights for composite ranking
SOURCE_AUTHORITY: dict[str, float] = {
    "official":   1.0,
    "pulsepoint": 0.9,
    "nextdoor":   0.8,
    "twitter":    0.7,
    "reddit":     0.7,
    "bluesky":    0.6,
    "gdelt":      0.5,
    "mastodon":   0.6,
    "telegram":   0.6,
    "nws":        1.0,
    "usgs":       1.0,
    "patch":      0.6,
}


@dataclass
class RetrievalResult:
    posts: list[dict] = field(default_factory=list)
    official_context: list[dict] = field(default_factory=list)
    enriched_locations: dict[str, dict] = field(default_factory=dict)  # post_id → location dict


class HybridRetriever:
    """
    Fetches candidate posts from three MongoDB sources, deduplicates,
    enriches unresolved locations, and returns a ranked list.
    """

    def __init__(self, mongo_main_client, mongo_vector_client, mongo_location_client) -> None:
        main_db_name     = os.environ.get("MONGO_DB_DATABASE",          "crisisecho")
        vector_db_name   = os.environ.get("MONGO_VECTOR_DB_DATABASE",   "crisisecho_vector")
        location_db_name = os.environ.get("MONGO_LOCATION_DB_DATABASE", "crisisecho_location")

        self.main_db     = mongo_main_client[main_db_name]
        self.vector_db   = mongo_vector_client[vector_db_name]
        self.location_db = mongo_location_client[location_db_name]

    # ── Public API ───────────────────────────────────────────────────────────

    def retrieve(self, lat: float, lng: float, trigger_time: datetime) -> RetrievalResult:
        """
        Run all four sub-queries and return a merged, ranked RetrievalResult.
        trigger_time is used as the upper bound of the 2-hour lookback window.
        """
        cutoff_ts = int((trigger_time - timedelta(seconds=LOOKBACK_S)).timestamp())
        min_lat, max_lat = lat - BBOX_DEG, lat + BBOX_DEG
        min_lng, max_lng = lng - BBOX_DEG, lng + BBOX_DEG
        bounding_polygon = self._bounding_polygon(lat, lng, BBOX_DEG)

        # Q1 — Atlas Vector Search
        trigger_embedding = self._get_trigger_embedding(lat, lng)
        vector_results = self._query_vector(
            trigger_embedding, cutoff_ts,
            min_lat, max_lat, min_lng, max_lng,
        )

        # Q2 — Geo $near on posts
        geo_results = self._query_geo(lat, lng, trigger_time)

        # Q3 — Official signals
        official = self._query_official(bounding_polygon)

        # Q4 — Location enrichment for unresolved posts
        enriched = self._enrich_locations(geo_results)

        # Merge Q1 + Q2 and rank
        merged = self._merge_and_rank(
            vector_results, geo_results,
            trigger_time,
        )

        return RetrievalResult(
            posts=merged,
            official_context=official,
            enriched_locations=enriched,
        )

    # ── Sub-queries ──────────────────────────────────────────────────────────

    def _query_vector(
        self,
        embedding: list[float],
        cutoff_ts: int,
        min_lat: float, max_lat: float,
        min_lng: float, max_lng: float,
    ) -> list[dict]:
        if not embedding:
            return []
        pipeline = [
            {
                "$vectorSearch": {
                    "index":        "text_vector_index",
                    "path":         "vector",
                    "queryVector":  embedding,
                    "numCandidates": 200,
                    "limit":        50,
                    "filter": {
                        "timestamp":  {"$gt": cutoff_ts},
                        "lat":        {"$gte": min_lat, "$lte": max_lat},
                        "lng":        {"$gte": min_lng, "$lte": max_lng},
                        "is_relevant": True,
                    },
                }
            },
            {
                "$project": {
                    "post_id":    1,
                    "source":     1,
                    "lat":        1,
                    "lng":        1,
                    "crisis_type": 1,
                    "score": {"$meta": "vectorSearchScore"},
                }
            },
        ]
        try:
            return list(self.vector_db.text_embeddings.aggregate(pipeline))
        except Exception as exc:
            logger.warning("vector search error: %s", exc)
            return []

    def _query_geo(self, lat: float, lng: float, trigger_time: datetime) -> list[dict]:
        cutoff = trigger_time - timedelta(seconds=LOOKBACK_S)
        try:
            return list(self.main_db.posts.find(
                {
                    "location": {
                        "$near": {
                            "$geometry": {"type": "Point", "coordinates": [lng, lat]},
                            "$maxDistance": SEARCH_RADIUS_M,
                        }
                    },
                    "timestamp": {"$gt": cutoff},
                },
                limit=MAX_GEO_RESULTS,
            ))
        except Exception as exc:
            logger.warning("geo $near error: %s", exc)
            return []

    def _query_official(self, bounding_polygon: dict) -> list[dict]:
        try:
            return list(self.main_db.official_alerts.find({
                "location": {
                    "$geoIntersects": {"$geometry": bounding_polygon}
                },
                "active": True,
            }))
        except Exception as exc:
            logger.warning("official signals query error: %s", exc)
            return []

    def _enrich_locations(self, posts: list[dict]) -> dict[str, dict]:
        enriched: dict[str, dict] = {}
        unresolved = [p for p in posts if p.get("location_source") == "unresolved"]
        for post in unresolved:
            post_id  = str(post.get("_id", ""))
            text     = post.get("text", "")
            text_hash = hashlib.sha256(text.encode()).hexdigest()

            # Try location cache first
            cached = None
            try:
                cached = self.location_db.location_cache.find_one({"text_hash": text_hash})
            except Exception:
                pass

            if cached:
                enriched[post_id] = {
                    "lat": cached.get("lat", 0.0),
                    "lng": cached.get("lng", 0.0),
                    "location_confidence": cached.get("confidence", 0.5),
                    "location_source": "location_cache",
                }
                continue

            # Try geo_priors $near based on surrounding resolved posts
            resolved = [p for p in posts if p.get("location_source") != "unresolved"]
            if resolved:
                ref = resolved[0]
                ref_coords = ref.get("location", {}).get("coordinates", [0, 0])
                try:
                    prior = self.location_db.geo_priors.find_one({
                        "coordinates": {
                            "$near": {
                                "$geometry": {"type": "Point", "coordinates": ref_coords},
                                "$maxDistance": SEARCH_RADIUS_M,
                            }
                        }
                    })
                    if prior:
                        enriched[post_id] = {
                            "lat": prior.get("lat", 0.0),
                            "lng": prior.get("lng", 0.0),
                            "location_confidence": prior.get("confidence", 0.3),
                            "location_source": "geo_prior",
                        }
                except Exception as exc:
                    logger.debug("geo prior lookup error: %s", exc)

        return enriched

    # ── Merge + rank ─────────────────────────────────────────────────────────

    def _merge_and_rank(
        self,
        vector_results: list[dict],
        geo_results: list[dict],
        trigger_time: datetime,
    ) -> list[dict]:
        # Build a lookup from vector results: post_id → vector_score
        vector_scores: dict[str, float] = {}
        for doc in vector_results:
            pid = doc.get("post_id", "")
            if pid:
                vector_scores[pid] = float(doc.get("score", 0.0))

        # Combine all posts, dedup by post_id
        seen: set = set()
        combined: list[dict] = []
        for post in geo_results:
            pid = post.get("post_id", str(post.get("_id", "")))
            if pid not in seen:
                seen.add(pid)
                combined.append(post)

        # Score and sort
        now_ts = trigger_time.timestamp()
        def composite_score(post: dict) -> float:
            pid          = post.get("post_id", str(post.get("_id", "")))
            vscore       = vector_scores.get(pid, 0.0)
            ts           = post.get("timestamp")
            if isinstance(ts, datetime):
                age_s = max(0, now_ts - ts.timestamp())
            else:
                age_s = LOOKBACK_S
            recency      = max(0.0, 1.0 - age_s / LOOKBACK_S)
            source       = post.get("source", "")
            authority    = SOURCE_AUTHORITY.get(source, 0.5)
            return vscore * 0.5 + recency * 0.3 + authority * 0.2

        combined.sort(key=composite_score, reverse=True)
        return combined

    # ── Helpers ──────────────────────────────────────────────────────────────

    def _get_trigger_embedding(self, lat: float, lng: float) -> list[float]:
        """Generate a trigger embedding from location context text via Voyage AI."""
        if not VOYAGE_API_KEY:
            return []
        text = f"Crisis event near latitude {lat:.4f} longitude {lng:.4f}"
        try:
            resp = requests.post(
                VOYAGE_API_URL,
                headers={"Authorization": f"Bearer {VOYAGE_API_KEY}"},
                json={"input": [text], "model": VOYAGE_MODEL},
                timeout=10,
            )
            resp.raise_for_status()
            return resp.json()["data"][0]["embedding"]
        except Exception as exc:
            logger.warning("trigger embedding error: %s", exc)
            return []

    @staticmethod
    def _bounding_polygon(lat: float, lng: float, deg: float) -> dict:
        """Return a GeoJSON Polygon bounding box ±deg around (lat, lng)."""
        min_lat, max_lat = lat - deg, lat + deg
        min_lng, max_lng = lng - deg, lng + deg
        return {
            "type": "Polygon",
            "coordinates": [[
                [min_lng, min_lat],
                [max_lng, min_lat],
                [max_lng, max_lat],
                [min_lng, max_lat],
                [min_lng, min_lat],  # close the ring
            ]],
        }
