"""
nws_worker.py — NOAA/NWS REST API poll → official_alerts / nws
"""

import logging
import time

import requests

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

NWS_ALERTS_URL  = "https://api.weather.gov/alerts/active"
POLL_INTERVAL_S = 60  # NWS updates alerts roughly every minute


class NWSWorker(KafkaWorker):
    topic  = "official_alerts"
    source = "nws"

    def __init__(self) -> None:
        super().__init__()
        self._seen_ids: set = set()

    def _fetch_alerts(self) -> list[dict]:
        headers = {
            "User-Agent":  "CrisisEcho/1.0 (crisisecho@example.com)",
            "Accept":      "application/geo+json",
        }
        try:
            resp = requests.get(NWS_ALERTS_URL, headers=headers, timeout=20)
            resp.raise_for_status()
        except requests.RequestException as exc:
            logger.warning("nws fetch error: %s", exc)
            return []

        data    = resp.json()
        alerts  = []
        for feature in data.get("features", []):
            props  = feature.get("properties", {})
            alert_id = props.get("id", "")
            if not alert_id or alert_id in self._seen_ids:
                continue
            self._seen_ids.add(alert_id)

            geometry = feature.get("geometry") or {}
            coords   = {}
            if geometry.get("type") == "Point":
                lng, lat = geometry["coordinates"]
                coords = {"lat": lat, "lng": lng}

            alerts.append({
                "post_id":     alert_id,
                "text":        f"{props.get('headline', '')} {props.get('description', '')}".strip(),
                "user":        "nws",
                "url":         props.get("@id", ""),
                "image_urls":  [],
                "crisis_type": props.get("event", ""),
                **coords,
            })
        return alerts

    def stream(self):
        """Poll the NWS API on a fixed interval; yield new alert payloads."""
        while True:
            for alert in self._fetch_alerts():
                yield alert
            time.sleep(POLL_INTERVAL_S)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = NWSWorker()
    try:
        w.run()
    finally:
        w.close()
