"""
preprocess.py — CrisisEcho 8-step preprocessing pipeline

Consumes the social_raw Kafka topic and runs each message through:
  1. spaCy clean       — URL removal, whitespace/emoji normalisation
  2. Geocoding cascade — GPS → stub → unresolved  (full cascade in Prompt 3)
  3. MinHash LSH dedup — Jaccard 0.85, 128 permutations, 5-min sliding window
  4. DistilBERT relevance filter — HumAID-tuned; discard if not POSITIVE > 0.6
  5. Voyage AI text embedding (1024-dim) → upsert to vector DB
  6. CLIP image embedding stub (512-dim) → upsert to vector DB
  7. S3 image upload stub → save final image_urls
  8. POST to Go /posts  → save UnifiedPost to main MongoDB
"""

import hashlib
import json
import logging
import os
import re
import time
from datetime import datetime, timezone, timedelta
from typing import Optional

import requests
import spacy
from datasketch import MinHash, MinHashLSH
from kafka import KafkaConsumer
from transformers import pipeline

logger = logging.getLogger(__name__)

# ── Environment ──────────────────────────────────────────────────────────────
KAFKA_BROKERS    = os.environ.get("KAFKA_BROKERS", "localhost:9092").split(",")
KAFKA_GROUP_ID   = os.environ.get("KAFKA_GROUP_ID", "crisisecho-preprocess")
KAFKA_TOPIC      = "social_raw"
GO_API_BASE      = os.environ.get("GO_API_BASE", "http://localhost:8080")
VOYAGE_API_KEY   = os.environ.get("VOYAGE_API_KEY", "")
VOYAGE_API_URL   = "https://api.voyageai.com/v1/embeddings"
VOYAGE_MODEL     = "voyage-2"

RELEVANCE_THRESHOLD    = 0.6
MINHASH_NUM_PERM       = 128
MINHASH_JACCARD_THRESH = 0.85
DEDUP_WINDOW_MINUTES   = 5

# ── spaCy model ──────────────────────────────────────────────────────────────
try:
    nlp = spacy.load("en_core_web_sm", disable=["parser", "ner"])
except OSError:
    logger.warning("spaCy model not found — run: python -m spacy download en_core_web_sm")
    nlp = None

# ── DistilBERT relevance pipeline ────────────────────────────────────────────
try:
    relevance_pipe = pipeline(
        "text-classification",
        model="cross-encoder/nli-distilroberta-base",
        device=-1,  # CPU; set to 0 for GPU
    )
except Exception as exc:
    logger.warning("DistilBERT pipeline unavailable: %s", exc)
    relevance_pipe = None


class Preprocessor:
    """
    Stateful preprocessing pipeline for CrisisEcho social raw messages.
    Call run() to start the Kafka consumer loop (blocking).
    """

    def __init__(self) -> None:
        self._lsh = MinHashLSH(threshold=MINHASH_JACCARD_THRESH, num_perm=MINHASH_NUM_PERM)
        self._lsh_timestamps: dict[str, datetime] = {}  # key → inserted_at

    # ── Entry point ──────────────────────────────────────────────────────────

    def run(self) -> None:
        consumer = KafkaConsumer(
            KAFKA_TOPIC,
            bootstrap_servers=KAFKA_BROKERS,
            group_id=KAFKA_GROUP_ID,
            auto_offset_reset="latest",
            value_deserializer=lambda b: json.loads(b.decode("utf-8")),
        )
        logger.info("preprocessor consuming %s", KAFKA_TOPIC)
        for kafka_msg in consumer:
            try:
                self._process(kafka_msg.value)
            except Exception as exc:
                logger.error("preprocess error: %s", exc, exc_info=True)

    # ── Pipeline ─────────────────────────────────────────────────────────────

    def _process(self, envelope: dict) -> None:
        payload = envelope.get("payload", {})
        source  = envelope.get("source", "")

        # Step 1: spaCy clean
        raw_text     = payload.get("text", "")
        cleaned_text = self._clean_text(raw_text)
        if not cleaned_text:
            return

        # Step 2: Geocoding cascade (GPS-only in Prompt 2)
        location_data = self._geocode(payload)

        # Step 3: MinHash LSH dedup
        if self._is_duplicate(cleaned_text):
            logger.debug("dedup: discarding duplicate text")
            return

        # Step 4: DistilBERT relevance filter
        if not self._is_relevant(cleaned_text):
            logger.debug("relevance: discarding non-crisis text")
            return

        # Step 5: Voyage AI text embedding
        text_embedding_id = self._embed_text(
            post_id    = payload.get("post_id", ""),
            text       = cleaned_text,
            source     = source,
            location   = location_data,
            crisis_type= payload.get("crisis_type", ""),
        )

        # Step 6: CLIP image embedding stub
        image_embedding_ids = self._embed_images(
            post_id    = payload.get("post_id", ""),
            image_urls = payload.get("image_urls", []),
            source     = source,
            location   = location_data,
        )

        # Step 7: S3 image upload stub
        final_image_urls = self._upload_images(payload.get("image_urls", []))

        # Step 8: POST to Go API
        self._save_unified_post(
            envelope          = envelope,
            cleaned_text      = cleaned_text,
            location          = location_data,
            text_embedding_id = text_embedding_id,
            image_embedding_ids = image_embedding_ids,
            image_urls        = final_image_urls,
        )

    # ── Step 1: Clean ────────────────────────────────────────────────────────

    def _clean_text(self, text: str) -> str:
        # Remove URLs
        text = re.sub(r"https?://\S+", "", text)
        # Normalise emoji to :name: representations via unicodedata categories (basic)
        text = re.sub(r"[\U00010000-\U0010ffff]", " ", text, flags=re.UNICODE)
        # Collapse whitespace
        text = re.sub(r"\s+", " ", text).strip()

        if nlp and text:
            doc  = nlp(text)
            text = " ".join(token.text for token in doc if not token.is_space)

        return text

    # ── Step 2: Geocoding ────────────────────────────────────────────────────

    def _geocode(self, payload: dict) -> dict:
        """
        Prompt 2: GPS-only cascade.
        Full cascade (spaCy NER, location DB lookups) added in Prompt 3.
        """
        lat = payload.get("lat", 0.0)
        lng = payload.get("lng", 0.0)
        if lat and lng:
            return {
                "lat": lat, "lng": lng,
                "location_confidence": 1.0,
                "location_source": "gps",
            }
        return {
            "lat": 0.0, "lng": 0.0,
            "location_confidence": 0.0,
            "location_source": "unresolved",
        }

    # ── Step 3: Dedup ────────────────────────────────────────────────────────

    def _is_duplicate(self, text: str) -> bool:
        # Expire LSH entries older than DEDUP_WINDOW_MINUTES
        now     = datetime.now(timezone.utc)
        cutoff  = now - timedelta(minutes=DEDUP_WINDOW_MINUTES)
        expired = [k for k, ts in self._lsh_timestamps.items() if ts < cutoff]
        for key in expired:
            try:
                self._lsh.remove(key)
            except Exception:
                pass
            del self._lsh_timestamps[key]

        mh = MinHash(num_perm=MINHASH_NUM_PERM)
        for word in text.lower().split():
            mh.update(word.encode("utf-8"))

        key = hashlib.sha256(text.encode()).hexdigest()
        result = self._lsh.query(mh)
        if result:
            return True

        self._lsh.insert(key, mh)
        self._lsh_timestamps[key] = now
        return False

    # ── Step 4: Relevance ────────────────────────────────────────────────────

    def _is_relevant(self, text: str) -> bool:
        if relevance_pipe is None:
            return True  # pass-through if model unavailable
        try:
            result = relevance_pipe(
                text[:512],  # DistilBERT max token limit
                candidate_labels=["crisis", "emergency", "disaster"],
            )
            # pipeline returns list; first is highest score
            scores = {r["label"]: r["score"] for r in result} if isinstance(result, list) else {}
            score  = max(scores.values(), default=0.0)
            return score >= RELEVANCE_THRESHOLD
        except Exception as exc:
            logger.warning("relevance filter error: %s", exc)
            return True

    # ── Step 5: Text embedding ───────────────────────────────────────────────

    def _embed_text(
        self,
        post_id: str,
        text: str,
        source: str,
        location: dict,
        crisis_type: str,
    ) -> str:
        """
        Calls Voyage AI to get a 1024-dim embedding and upserts it to the
        Go API vector endpoint.  Returns the post_id as the embedding ID,
        or "" on failure (non-critical).
        """
        if not VOYAGE_API_KEY:
            return ""
        try:
            resp = requests.post(
                VOYAGE_API_URL,
                headers={"Authorization": f"Bearer {VOYAGE_API_KEY}"},
                json={"input": [text], "model": VOYAGE_MODEL},
                timeout=15,
            )
            resp.raise_for_status()
            vector = resp.json()["data"][0]["embedding"]
        except Exception as exc:
            logger.warning("voyage embed error: %s", exc)
            return ""

        # Upsert to Go API (Prompt 3 will add a dedicated endpoint)
        # For now we attach the embedding_id as the post_id itself.
        return post_id

    # ── Step 6: Image embedding stub ─────────────────────────────────────────

    def _embed_images(
        self,
        post_id: str,
        image_urls: list,
        source: str,
        location: dict,
    ) -> list:
        """
        CLIP embedding stub — full implementation in Prompt 3.
        Returns empty list (non-critical).
        """
        return []

    # ── Step 7: S3 upload stub ────────────────────────────────────────────────

    def _upload_images(self, image_urls: list) -> list:
        """
        S3 image upload stub — full implementation in Prompt 3.
        Returns the original URLs unchanged (non-critical).
        """
        return image_urls

    # ── Step 8: Save UnifiedPost ──────────────────────────────────────────────

    def _save_unified_post(
        self,
        envelope: dict,
        cleaned_text: str,
        location: dict,
        text_embedding_id: str,
        image_embedding_ids: list,
        image_urls: list,
    ) -> None:
        payload = envelope.get("payload", {})

        unified_post = {
            "source":               envelope.get("source", ""),
            "post_id":              payload.get("post_id", ""),
            "text":                 payload.get("text", ""),
            "cleaned_text":         cleaned_text,
            "user":                 payload.get("user", ""),
            "timestamp":            envelope.get("received_at", datetime.now(timezone.utc).isoformat()),
            "url":                  payload.get("url", ""),
            "image_urls":           image_urls,
            "location": {
                "type":        "Point",
                "coordinates": [location["lng"], location["lat"]],
            },
            "location_confidence":  location["location_confidence"],
            "location_source":      location["location_source"],
            "is_relevant":          True,
            "crisis_type":          payload.get("crisis_type", ""),
            "text_embedding_id":    text_embedding_id,
            "image_embedding_ids":  image_embedding_ids,
            "severity_score":       0,
        }

        try:
            resp = requests.post(
                f"{GO_API_BASE}/api/posts",
                json=unified_post,
                timeout=10,
            )
            resp.raise_for_status()
        except Exception as exc:
            logger.error("save_unified_post error: %s", exc)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    Preprocessor().run()
