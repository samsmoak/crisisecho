"""
nextdoor_worker.py — Nextdoor scraper → social_raw / nextdoor

NOTE: Nextdoor has no public API.  This worker uses a session-cookie-based
scraper approach.  Requires NEXTDOOR_SESSION_COOKIE to be set in the
environment.  Posts are polled on a fixed interval rather than streamed.
"""

import logging
import os
import time

import requests
from bs4 import BeautifulSoup

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

POLL_INTERVAL_S = 60  # seconds between scrape cycles

CRISIS_TERMS = {
    "earthquake", "flood", "wildfire", "tornado", "hurricane",
    "tsunami", "explosion", "shooting", "evacuation", "emergency",
    "disaster", "shelter", "rescue", "alert", "warning",
}

NEXTDOOR_FEED_URL = "https://nextdoor.com/news_feed/"


class NextdoorWorker(KafkaWorker):
    topic  = "social_raw"
    source = "nextdoor"

    def __init__(self) -> None:
        super().__init__()
        self.session_cookie = os.environ.get("NEXTDOOR_SESSION_COOKIE", "")
        self._seen_ids: set = set()

    def _fetch_posts(self) -> list[dict]:
        """Scrape the Nextdoor public feed and return a list of raw post dicts."""
        headers = {
            "Cookie":     f"__cf_bm={self.session_cookie}",
            "User-Agent": "Mozilla/5.0 (compatible; CrisisEcho/1.0)",
        }
        try:
            resp = requests.get(NEXTDOOR_FEED_URL, headers=headers, timeout=15)
            resp.raise_for_status()
        except requests.RequestException as exc:
            logger.warning("nextdoor fetch error: %s", exc)
            return []

        soup = BeautifulSoup(resp.text, "html.parser")
        posts = []
        for card in soup.select("[data-testid='post-card']"):
            post_id = card.get("data-post-id", "")
            text_el = card.select_one("[data-testid='post-body']")
            text    = text_el.get_text(separator=" ") if text_el else ""
            author_el = card.select_one("[data-testid='author-name']")
            author    = author_el.get_text() if author_el else ""
            link_el   = card.select_one("a[href*='/p/']")
            url       = f"https://nextdoor.com{link_el['href']}" if link_el else ""

            if post_id and post_id not in self._seen_ids:
                posts.append({"post_id": post_id, "text": text, "user": author, "url": url, "image_urls": []})
                self._seen_ids.add(post_id)

        return posts

    def stream(self):
        """Poll-based generator — yields one payload per new post found each cycle."""
        while True:
            for post in self._fetch_posts():
                text_lower = post["text"].lower()
                if any(term in text_lower for term in CRISIS_TERMS):
                    yield post
            time.sleep(POLL_INTERVAL_S)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = NextdoorWorker()
    try:
        w.run()
    finally:
        w.close()
