"""
gdacs_worker.py — GDACS GeoRSS feed poll → official_alerts / gdacs

Global Disaster Alert and Coordination System (GDACS) provides a public
GeoRSS feed of ongoing and recent major disasters. No API key required.
"""

import logging
import time

import feedparser

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

GDACS_FEED_URL  = "https://www.gdacs.org/xml/rss.xml"
POLL_INTERVAL_S = 600  # 10 minutes


class GDACSWorker(KafkaWorker):
    topic  = "official_alerts"
    source = "gdacs"

    def __init__(self) -> None:
        super().__init__()
        self._seen_ids: set = set()

    def _fetch_events(self) -> list[dict]:
        try:
            feed = feedparser.parse(GDACS_FEED_URL)
        except Exception as exc:
            logger.warning("gdacs fetch error: %s", exc)
            return []

        events = []
        for entry in feed.entries:
            entry_id = entry.get("id", entry.get("link", ""))
            if not entry_id or entry_id in self._seen_ids:
                continue
            self._seen_ids.add(entry_id)

            title   = entry.get("title", "")
            summary = entry.get("summary", "")
            text    = f"{title}. {summary}".strip(". ") if summary else title

            # GDACS GeoRSS entries carry geo:lat and geo:long tags
            lat = float(entry.get("geo_lat", entry.get("latitude", 0.0)) or 0.0)
            lng = float(entry.get("geo_long", entry.get("longitude", 0.0)) or 0.0)

            # Map GDACS event type to crisis_type
            event_type = entry.get("gdacs_eventtype", "").lower() or "disaster"

            events.append({
                "post_id":     entry_id,
                "text":        text,
                "user":        "gdacs",
                "url":         entry.get("link", ""),
                "image_urls":  [],
                "crisis_type": event_type,
                "lat":         lat,
                "lng":         lng,
            })
        return events

    def stream(self):
        """Poll the GDACS GeoRSS feed on a fixed interval."""
        while True:
            for event in self._fetch_events():
                yield event
            time.sleep(POLL_INTERVAL_S)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = GDACSWorker()
    try:
        w.run()
    finally:
        w.close()
