"""
usgs_worker.py — USGS Earthquake Hazards REST API poll → official_alerts / usgs
"""

import logging
import time

import requests

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

# GeoJSON feed: all earthquakes M2.5+ in the last hour, updated every minute
USGS_FEED_URL   = "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/2.5_hour.geojson"
POLL_INTERVAL_S = 60


class USGSWorker(KafkaWorker):
    topic  = "official_alerts"
    source = "usgs"

    def __init__(self) -> None:
        super().__init__()
        self._seen_ids: set = set()

    def _fetch_quakes(self) -> list[dict]:
        try:
            resp = requests.get(USGS_FEED_URL, timeout=20)
            resp.raise_for_status()
        except requests.RequestException as exc:
            logger.warning("usgs fetch error: %s", exc)
            return []

        data   = resp.json()
        quakes = []
        for feature in data.get("features", []):
            quake_id = feature.get("id", "")
            if not quake_id or quake_id in self._seen_ids:
                continue
            self._seen_ids.add(quake_id)

            props    = feature.get("properties", {})
            geometry = feature.get("geometry", {})
            coords   = geometry.get("coordinates", [0.0, 0.0, 0.0])
            lng, lat = coords[0], coords[1]

            magnitude = props.get("mag", 0)
            place     = props.get("place", "")

            quakes.append({
                "post_id":     quake_id,
                "text":        f"M{magnitude} earthquake {place}",
                "user":        "usgs",
                "url":         props.get("url", ""),
                "image_urls":  [],
                "crisis_type": "earthquake",
                "lat":         lat,
                "lng":         lng,
                "magnitude":   magnitude,
            })
        return quakes

    def stream(self):
        """Poll the USGS earthquake feed on a fixed interval."""
        while True:
            for quake in self._fetch_quakes():
                yield quake
            time.sleep(POLL_INTERVAL_S)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = USGSWorker()
    try:
        w.run()
    finally:
        w.close()
