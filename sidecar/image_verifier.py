"""
image_verifier.py — Image similarity cross-check via Atlas Vector Search.

Queries the image_embeddings collection to find visually similar images
across different posts in the same geographic area.
"""

import logging
import os
from dataclasses import dataclass, field

logger = logging.getLogger(__name__)

SIMILARITY_THRESHOLD = 0.85
BBOX_DEG             = 0.5       # ±0.5° geographic filter


@dataclass
class ImageCorroborationResult:
    corroborated: bool           = False
    matching_post_ids: list[str] = field(default_factory=list)
    match_count: int             = 0


class ImageVerifier:
    """
    Checks whether multiple posts near the same location share visually
    similar images — a strong signal of a real crisis event.
    """

    def __init__(self, mongo_vector_client) -> None:
        vector_db_name    = os.environ.get("MONGO_VECTOR_DB_DATABASE", "crisisecho_vector")
        self.vector_db    = mongo_vector_client[vector_db_name]

    # ── Public API ───────────────────────────────────────────────────────────

    def verify_images(
        self,
        post_ids: list[str],
        lat: float,
        lng: float,
    ) -> ImageCorroborationResult:
        """
        For each post that has image embeddings, perform an Atlas Vector Search
        for visually similar images in the same bounding box.

        Returns ImageCorroborationResult(corroborated=True) if two or more
        distinct posts share visually similar images (score > 0.85).
        """
        min_lat = lat - BBOX_DEG
        max_lat = lat + BBOX_DEG
        min_lng = lng - BBOX_DEG
        max_lng = lng + BBOX_DEG

        # Fetch image embedding docs for the candidate posts
        embedding_docs = self._get_embeddings(post_ids)
        if not embedding_docs:
            return ImageCorroborationResult()

        matching_post_ids: set[str] = set()

        for doc in embedding_docs:
            source_post_id = doc.get("post_id", "")
            vector         = doc.get("vector")
            if not vector:
                continue

            similar = self._vector_search_images(
                vector, min_lat, max_lat, min_lng, max_lng
            )

            for hit in similar:
                hit_post_id = hit.get("post_id", "")
                score       = float(hit.get("score", 0.0))

                if hit_post_id and hit_post_id != source_post_id and score >= SIMILARITY_THRESHOLD:
                    matching_post_ids.add(source_post_id)
                    matching_post_ids.add(hit_post_id)

        corroborated = len(matching_post_ids) >= 2
        return ImageCorroborationResult(
            corroborated=corroborated,
            matching_post_ids=list(matching_post_ids),
            match_count=len(matching_post_ids),
        )

    # ── Helpers ──────────────────────────────────────────────────────────────

    def _get_embeddings(self, post_ids: list[str]) -> list[dict]:
        if not post_ids:
            return []
        try:
            return list(self.vector_db.image_embeddings.find(
                {"post_id": {"$in": post_ids}}
            ))
        except Exception as exc:
            logger.warning("image embedding fetch error: %s", exc)
            return []

    def _vector_search_images(
        self,
        vector: list[float],
        min_lat: float, max_lat: float,
        min_lng: float, max_lng: float,
    ) -> list[dict]:
        pipeline = [
            {
                "$vectorSearch": {
                    "index":         "image_vector_index",
                    "path":          "vector",
                    "queryVector":   vector,
                    "numCandidates": 100,
                    "limit":         10,
                    "filter": {
                        "lat": {"$gte": min_lat, "$lte": max_lat},
                        "lng": {"$gte": min_lng, "$lte": max_lng},
                    },
                }
            },
            {
                "$project": {
                    "post_id": 1,
                    "score":   {"$meta": "vectorSearchScore"},
                }
            },
        ]
        try:
            return list(self.vector_db.image_embeddings.aggregate(pipeline))
        except Exception as exc:
            logger.warning("image vector search error: %s", exc)
            return []
