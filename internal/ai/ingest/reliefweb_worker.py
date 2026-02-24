"""
reliefweb_worker.py — ReliefWeb Disasters API poll → official_alerts / reliefweb

ReliefWeb provides a free humanitarian crisis API. Set RELIEFWEB_APP_NAME
to identify your application (recommended but not required for public access).
"""

import logging
import os
import time

import requests

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

RELIEFWEB_API_URL = "https://api.reliefweb.int/v1/disasters"
RELIEFWEB_APP_NAME = os.environ.get("RELIEFWEB_APP_NAME", "crisisecho")
POLL_INTERVAL_S    = 900  # 15 minutes
PAGE_SIZE          = 50


class ReliefWebWorker(KafkaWorker):
    topic  = "official_alerts"
    source = "reliefweb"

    def __init__(self) -> None:
        super().__init__()
        self._seen_ids: set = set()

    def _fetch_disasters(self) -> list[dict]:
        params = {
            "appname": RELIEFWEB_APP_NAME,
            "limit":   PAGE_SIZE,
            "sort[]":  "date.created:desc",
            "fields[include][]": ["id", "name", "type", "country", "date", "status", "description"],
        }
        try:
            resp = requests.get(RELIEFWEB_API_URL, params=params, timeout=20)
            resp.raise_for_status()
        except requests.RequestException as exc:
            logger.warning("reliefweb fetch error: %s", exc)
            return []

        items = []
        for item in resp.json().get("data", []):
            item_id = str(item.get("id", ""))
            if not item_id or item_id in self._seen_ids:
                continue
            self._seen_ids.add(item_id)

            fields      = item.get("fields", {})
            name        = fields.get("name", "")
            description = fields.get("description", "")
            text        = f"{name}. {description}".strip(". ") if description else name

            # Extract country coordinates if available
            countries = fields.get("country", [])
            lat, lng  = 0.0, 0.0
            if countries:
                loc = countries[0].get("location", {})
                lat = float(loc.get("lat", 0.0) or 0.0)
                lng = float(loc.get("lon", 0.0) or 0.0)

            # Map ReliefWeb disaster type to crisis_type
            types      = fields.get("type", [])
            crisis_type = types[0].get("name", "disaster").lower() if types else "disaster"

            items.append({
                "post_id":     item_id,
                "text":        text,
                "user":        "reliefweb",
                "url":         f"https://reliefweb.int/disaster/{item_id}",
                "image_urls":  [],
                "crisis_type": crisis_type,
                "lat":         lat,
                "lng":         lng,
            })
        return items

    def stream(self):
        """Poll the ReliefWeb disasters API on a fixed interval."""
        while True:
            for disaster in self._fetch_disasters():
                yield disaster
            time.sleep(POLL_INTERVAL_S)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = ReliefWebWorker()
    try:
        w.run()
    finally:
        w.close()
