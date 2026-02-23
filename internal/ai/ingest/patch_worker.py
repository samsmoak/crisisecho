"""
patch_worker.py — Patch.com RSS poll → news_feed / patch
"""

import logging
import time

import feedparser

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

POLL_INTERVAL_S = 300  # 5 minutes

# Patch publishes local RSS feeds per city/region.
# Extend this list with feeds relevant to your deployment geography.
PATCH_RSS_FEEDS = [
    "https://patch.com/us/rss",
]

CRISIS_TERMS = {
    "earthquake", "flood", "wildfire", "tornado", "hurricane",
    "tsunami", "explosion", "shooting", "evacuation", "emergency",
    "disaster", "shelter", "rescue", "fire", "storm", "alert",
}


class PatchWorker(KafkaWorker):
    topic  = "news_feed"
    source = "patch"

    def __init__(self) -> None:
        super().__init__()
        self._seen_ids: set = set()

    def _fetch_feed(self, rss_url: str) -> list[dict]:
        feed    = feedparser.parse(rss_url)
        results = []
        for entry in feed.entries:
            entry_id = entry.get("id", entry.get("link", ""))
            if not entry_id or entry_id in self._seen_ids:
                continue

            title   = entry.get("title", "")
            summary = entry.get("summary", "")
            text    = f"{title} {summary}".strip()

            if not any(term in text.lower() for term in CRISIS_TERMS):
                self._seen_ids.add(entry_id)
                continue

            self._seen_ids.add(entry_id)
            results.append({
                "post_id":    entry_id,
                "text":       text,
                "user":       entry.get("author", "patch"),
                "url":        entry.get("link", ""),
                "image_urls": [],
            })
        return results

    def stream(self):
        """Poll each Patch RSS feed every POLL_INTERVAL_S seconds."""
        while True:
            for url in PATCH_RSS_FEEDS:
                for item in self._fetch_feed(url):
                    yield item
            time.sleep(POLL_INTERVAL_S)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = PatchWorker()
    try:
        w.run()
    finally:
        w.close()
