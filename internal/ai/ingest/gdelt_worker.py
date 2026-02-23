"""
gdelt_worker.py — GDELT 2.0 Events stream → news_feed / gdelt
"""

import io
import logging
import time
import zipfile

import requests

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

# GDELT publishes a new 15-minute events CSV every 15 minutes.
# The master file lists all available update URLs.
GDELT_MASTER_URL  = "http://data.gdeltproject.org/gdeltv2/lastupdate.txt"
POLL_INTERVAL_S   = 900  # 15 minutes

CRISIS_EVENT_CODES = {
    "0211", "0231",  # appeals / protest
    "060",  "061",   # demand
    "070",  "071",   # disapprove
    "1011", "1012",  # coerce
    "120",  "121",   # reject
    "130",  "131",   # threaten
    "140",  "141",   # protest
    "150",  "151",   # exhibit military posture
    "160",  "161",   # reduce relations
    "170",  "171",   # coerce
    "180",  "181",   # assault
    "190",  "191",   # use unconventional mass violence
    "200",  "201",   # mass killing
}


class GDELTWorker(KafkaWorker):
    topic  = "news_feed"
    source = "gdelt"

    def __init__(self) -> None:
        super().__init__()
        self._seen_urls: set = set()

    def _get_latest_csv_url(self) -> str | None:
        try:
            resp = requests.get(GDELT_MASTER_URL, timeout=15)
            resp.raise_for_status()
        except requests.RequestException as exc:
            logger.warning("gdelt master fetch error: %s", exc)
            return None

        for line in resp.text.strip().splitlines():
            parts = line.split()
            if len(parts) >= 3 and "export" in parts[2].lower():
                return parts[2]
        return None

    def _fetch_events(self, csv_url: str) -> list[dict]:
        if csv_url in self._seen_urls:
            return []
        self._seen_urls.add(csv_url)

        try:
            resp = requests.get(csv_url, timeout=60)
            resp.raise_for_status()
        except requests.RequestException as exc:
            logger.warning("gdelt csv fetch error: %s", exc)
            return []

        events = []
        with zipfile.ZipFile(io.BytesIO(resp.content)) as zf:
            for name in zf.namelist():
                with zf.open(name) as f:
                    for line in f:
                        cols = line.decode("utf-8", errors="ignore").rstrip("\n").split("\t")
                        if len(cols) < 60:
                            continue
                        event_code = cols[26]
                        if not any(event_code.startswith(c) for c in CRISIS_EVENT_CODES):
                            continue
                        event_id = cols[0]
                        actor1   = cols[6]
                        actor2   = cols[16]
                        lat      = float(cols[55]) if cols[55] else 0.0
                        lng      = float(cols[56]) if cols[56] else 0.0
                        url      = cols[57] if len(cols) > 57 else ""
                        events.append({
                            "post_id":    event_id,
                            "text":       f"GDELT event {event_code}: {actor1} → {actor2}",
                            "user":       "gdelt",
                            "url":        url,
                            "image_urls": [],
                            "lat":        lat,
                            "lng":        lng,
                        })
        return events

    def stream(self):
        """Poll GDELT every 15 minutes and yield crisis-coded events."""
        while True:
            url = self._get_latest_csv_url()
            if url:
                for event in self._fetch_events(url):
                    yield event
            time.sleep(POLL_INTERVAL_S)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = GDELTWorker()
    try:
        w.run()
    finally:
        w.close()
