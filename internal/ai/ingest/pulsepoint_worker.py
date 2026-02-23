"""
pulsepoint_worker.py — PulsePoint AED/incident API poll → official_alerts / pulsepoint
"""

import logging
import os
import time

import requests

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

PULSEPOINT_API_BASE = "https://api.pulsepoint.org/v1"
POLL_INTERVAL_S     = 30  # PulsePoint updates rapidly; poll every 30 s


class PulsePointWorker(KafkaWorker):
    topic  = "official_alerts"
    source = "pulsepoint"

    def __init__(self) -> None:
        super().__init__()
        self.api_key  = os.environ.get("PULSEPOINT_API_KEY", "")
        self.agency_ids = [
            a.strip()
            for a in os.environ.get("PULSEPOINT_AGENCY_IDS", "").split(",")
            if a.strip()
        ]
        self._seen_ids: set = set()

    def _fetch_incidents(self) -> list[dict]:
        if not self.api_key:
            logger.warning("PULSEPOINT_API_KEY not set — skipping fetch")
            return []

        incidents = []
        for agency_id in self.agency_ids or [""]:
            params = {"apikey": self.api_key}
            if agency_id:
                params["agency_id"] = agency_id

            try:
                resp = requests.get(
                    f"{PULSEPOINT_API_BASE}/incidents/active",
                    params=params,
                    timeout=15,
                )
                resp.raise_for_status()
            except requests.RequestException as exc:
                logger.warning("pulsepoint fetch error (agency=%s): %s", agency_id, exc)
                continue

            for incident in resp.json().get("incidents", {}).get("active", []):
                incident_id = incident.get("ID", "")
                if not incident_id or incident_id in self._seen_ids:
                    continue
                self._seen_ids.add(incident_id)

                call_type = incident.get("CallType", "")
                address   = incident.get("FullDisplayAddress", "")
                lat       = float(incident.get("Latitude",  0))
                lng       = float(incident.get("Longitude", 0))

                incidents.append({
                    "post_id":     incident_id,
                    "text":        f"{call_type} at {address}",
                    "user":        "pulsepoint",
                    "url":         "",
                    "image_urls":  [],
                    "crisis_type": call_type,
                    "lat":         lat,
                    "lng":         lng,
                })
        return incidents

    def stream(self):
        """Poll PulsePoint every POLL_INTERVAL_S seconds."""
        while True:
            for incident in self._fetch_incidents():
                yield incident
            time.sleep(POLL_INTERVAL_S)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = PulsePointWorker()
    try:
        w.run()
    finally:
        w.close()
