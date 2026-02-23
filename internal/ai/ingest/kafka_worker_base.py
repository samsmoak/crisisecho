"""
kafka_worker_base.py — Base class for all CrisisEcho ingestion workers.

Every platform worker extends KafkaWorker and overrides stream().
The base class handles:
  - Kafka producer setup
  - SHA-256 username hashing
  - Message envelope construction (KafkaMessage schema)
  - produce() with exponential-backoff retry
"""

import hashlib
import json
import logging
import os
import time
from abc import ABC, abstractmethod
from datetime import datetime, timezone

from kafka import KafkaProducer
from kafka.errors import KafkaError

logger = logging.getLogger(__name__)


class KafkaWorker(ABC):
    """
    Abstract base for all ingestion workers.

    Subclass must set:
      topic  (str)  — Kafka topic (e.g. "social_raw")
      source (str)  — Platform identifier (e.g. "twitter")

    Subclass must implement:
      stream() → generator yielding raw payload dicts
    """

    topic: str = ""
    source: str = ""

    # Retry parameters for produce()
    MAX_RETRIES: int = 5
    BASE_BACKOFF_S: float = 1.0
    MAX_BACKOFF_S: float = 60.0

    def __init__(self) -> None:
        brokers_env = os.environ.get("KAFKA_BROKERS", "localhost:9092")
        self.brokers = [b.strip() for b in brokers_env.split(",")]
        self.producer = self._build_producer()

    # ── Kafka setup ──────────────────────────────────────────────────────────

    def _build_producer(self) -> KafkaProducer:
        kwargs = dict(
            bootstrap_servers=self.brokers,
            value_serializer=lambda v: json.dumps(v).encode("utf-8"),
            acks="all",
            retries=3,
            max_block_ms=10_000,
        )
        # Aiven TLS — enabled when KAFKA_SSL_CA_FILE is set
        ca_file   = os.environ.get("KAFKA_SSL_CA_FILE")
        cert_file = os.environ.get("KAFKA_SSL_CERT_FILE")
        key_file  = os.environ.get("KAFKA_SSL_KEY_FILE")
        if ca_file:
            kwargs["security_protocol"] = "SSL"
            kwargs["ssl_cafile"]        = ca_file
            if cert_file:
                kwargs["ssl_certfile"] = cert_file
            if key_file:
                kwargs["ssl_keyfile"] = key_file
        return KafkaProducer(**kwargs)

    # ── Public API ───────────────────────────────────────────────────────────

    @abstractmethod
    def stream(self):
        """
        Generator yielding raw payload dicts.  Each dict becomes the
        ``payload`` field of the KafkaMessage envelope.

        Required payload keys (add any platform-specific extras):
          text      (str)
          post_id   (str)
          user      (str)  — plaintext; hashed here before sending
          url       (str, optional)
          image_urls (list[str], optional)
        """

    def run(self) -> None:
        """Entry point — consume stream() and produce each item."""
        logger.info("worker started: source=%s topic=%s brokers=%s",
                    self.source, self.topic, self.brokers)
        for payload in self.stream():
            self._produce_with_retry(payload)

    # ── Helpers ──────────────────────────────────────────────────────────────

    @staticmethod
    def hash_username(username: str) -> str:
        """Return SHA-256 hex digest of the username."""
        return hashlib.sha256(username.encode("utf-8")).hexdigest()

    def _build_envelope(self, payload: dict) -> dict:
        """
        Wrap a raw payload dict in the KafkaMessage envelope expected by the
        Go consumer (ingest/model/ingest.go).

        The ``user`` field is hashed in-place before serialisation.
        """
        if "user" in payload and payload["user"]:
            payload = dict(payload)  # shallow copy — do not mutate caller's dict
            payload["user"] = self.hash_username(payload["user"])

        return {
            "topic":       self.topic,
            "source":      self.source,
            "payload":     payload,
            "received_at": datetime.now(timezone.utc).isoformat(),
        }

    def produce(self, payload: dict) -> None:
        """Produce one message to Kafka (without retry — use _produce_with_retry)."""
        envelope = self._build_envelope(payload)
        future = self.producer.send(self.topic, value=envelope)
        future.get(timeout=10)  # propagate any send error immediately

    def _produce_with_retry(self, payload: dict) -> None:
        """Produce one message with exponential-backoff retry on KafkaError."""
        backoff = self.BASE_BACKOFF_S
        for attempt in range(1, self.MAX_RETRIES + 1):
            try:
                self.produce(payload)
                return
            except KafkaError as exc:
                if attempt == self.MAX_RETRIES:
                    logger.error(
                        "produce failed after %d attempts (source=%s): %s",
                        attempt, self.source, exc,
                    )
                    return
                sleep_s = min(backoff, self.MAX_BACKOFF_S)
                logger.warning(
                    "produce attempt %d/%d failed, retrying in %.1fs: %s",
                    attempt, self.MAX_RETRIES, sleep_s, exc,
                )
                time.sleep(sleep_s)
                backoff *= 2
            except Exception as exc:
                logger.error("produce unexpected error (source=%s): %s", self.source, exc)
                return

    def close(self) -> None:
        """Flush and close the Kafka producer."""
        self.producer.flush(timeout=10)
        self.producer.close()
