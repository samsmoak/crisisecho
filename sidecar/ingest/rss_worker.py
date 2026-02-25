"""
rss_worker.py — Generic RSS feed poll → social_raw / rss

Reads comma-separated feed URLs from RSS_FEED_URLS environment variable.
Each entry from an RSS feed is treated as a social post for preprocessing.
"""

import logging
import os
import time

import feedparser

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

POLL_INTERVAL_S = 300  # 5 minutes
RSS_FEED_URLS   = [u.strip() for u in os.environ.get("RSS_FEED_URLS", "").split(",") if u.strip()]


class RSSWorker(KafkaWorker):
    topic  = "social_raw"
    source = "rss"

    def __init__(self) -> None:
        super().__init__()
        self._seen_ids: set = set()

    def _fetch_feed(self, url: str) -> list[dict]:
        try:
            feed = feedparser.parse(url)
        except Exception as exc:
            logger.warning("rss fetch error (%s): %s", url, exc)
            return []

        items = []
        for entry in feed.entries:
            entry_id = entry.get("id", entry.get("link", ""))
            if not entry_id or entry_id in self._seen_ids:
                continue
            self._seen_ids.add(entry_id)

            title   = entry.get("title", "")
            summary = entry.get("summary", "")
            text    = f"{title}. {summary}".strip(". ") if summary else title

            items.append({
                "post_id":    entry_id,
                "text":       text,
                "user":       feed.feed.get("title", "rss"),
                "url":        entry.get("link", ""),
                "image_urls": [],
            })
        return items

    def stream(self):
        """Poll all configured RSS feed URLs on a fixed interval."""
        while True:
            for url in RSS_FEED_URLS:
                for item in self._fetch_feed(url):
                    yield item
            time.sleep(POLL_INTERVAL_S)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = RSSWorker()
    try:
        w.run()
    finally:
        w.close()
