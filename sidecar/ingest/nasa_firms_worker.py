"""
nasa_firms_worker.py — NASA FIRMS active fire data poll → official_alerts / nasa_firms

NASA Fire Information for Resource Management System (FIRMS) provides
near-real-time active fire and thermal anomaly data via CSV API.
Requires NASA_FIRMS_MAP_KEY environment variable (free registration at
https://firms.modaps.eosdis.nasa.gov/api/map_key/).
"""

import csv
import hashlib
import io
import logging
import os
import time

import requests

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

NASA_FIRMS_MAP_KEY  = os.environ.get("NASA_FIRMS_MAP_KEY", "")
# VIIRS SNPP NRT — global coverage, 375m resolution, 1-day lookback
NASA_FIRMS_API_URL  = "https://firms.modaps.eosdis.nasa.gov/api/area/csv/{key}/VIIRS_SNPP_NRT/world/1"
POLL_INTERVAL_S     = 600  # 10 minutes
BRIGHTNESS_THRESHOLD = 320.0  # Kelvin — filter low-confidence detections


class NASAFIRMSWorker(KafkaWorker):
    topic  = "official_alerts"
    source = "nasa_firms"

    def __init__(self) -> None:
        super().__init__()
        self._seen_ids: set = set()

    def _fetch_fires(self) -> list[dict]:
        if not NASA_FIRMS_MAP_KEY:
            logger.warning("NASA_FIRMS_MAP_KEY not set — skipping FIRMS poll")
            return []

        url = NASA_FIRMS_API_URL.format(key=NASA_FIRMS_MAP_KEY)
        try:
            resp = requests.get(url, timeout=60)
            resp.raise_for_status()
        except requests.RequestException as exc:
            logger.warning("nasa_firms fetch error: %s", exc)
            return []

        fires = []
        reader = csv.DictReader(io.StringIO(resp.text))
        for row in reader:
            try:
                lat        = float(row.get("latitude", 0))
                lng        = float(row.get("longitude", 0))
                brightness = float(row.get("bright_ti4", row.get("brightness", 0)) or 0)
                acq_date   = row.get("acq_date", "")
                acq_time   = row.get("acq_time", "")
            except (ValueError, TypeError):
                continue

            if brightness < BRIGHTNESS_THRESHOLD:
                continue

            # Stable ID from location + acquisition time
            raw_id  = f"{lat:.4f}_{lng:.4f}_{acq_date}_{acq_time}"
            fire_id = hashlib.sha256(raw_id.encode()).hexdigest()[:16]
            if fire_id in self._seen_ids:
                continue
            self._seen_ids.add(fire_id)

            fires.append({
                "post_id":     fire_id,
                "text":        f"Active fire detected at {lat:.4f}, {lng:.4f} on {acq_date}. Brightness: {brightness}K",
                "user":        "nasa_firms",
                "url":         "https://firms.modaps.eosdis.nasa.gov/",
                "image_urls":  [],
                "crisis_type": "wildfire",
                "lat":         lat,
                "lng":         lng,
                "brightness":  brightness,
            })
        return fires

    def stream(self):
        """Poll the NASA FIRMS API on a fixed interval."""
        while True:
            for fire in self._fetch_fires():
                yield fire
            time.sleep(POLL_INTERVAL_S)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = NASAFIRMSWorker()
    try:
        w.run()
    finally:
        w.close()
