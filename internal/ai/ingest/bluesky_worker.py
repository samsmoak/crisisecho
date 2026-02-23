"""
bluesky_worker.py — AT Protocol Jetstream firehose → social_raw / bluesky
"""

import json
import logging
import os
import websocket

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

# AT Protocol Jetstream public endpoint (no auth required for public posts)
JETSTREAM_URL = "wss://jetstream2.us-east.bsky.network/subscribe?wantedCollections=app.bsky.feed.post"

CRISIS_TERMS = {
    "earthquake", "flood", "wildfire", "tornado", "hurricane",
    "tsunami", "explosion", "shooting", "evacuation", "emergency",
    "disaster", "shelter", "rescue",
}


class BlueskyWorker(KafkaWorker):
    topic  = "social_raw"
    source = "bluesky"

    def __init__(self) -> None:
        super().__init__()
        # Credentials are optional for Jetstream (public firehose)
        self.handle   = os.environ.get("BLUESKY_HANDLE", "")
        self.password = os.environ.get("BLUESKY_PASSWORD", "")

    def stream(self):
        """
        Connects to the AT Protocol Jetstream WebSocket and yields payloads
        for posts that contain at least one crisis-related keyword.
        """
        worker = self

        def on_message(ws, message):
            try:
                event = json.loads(message)
            except json.JSONDecodeError:
                return

            commit = event.get("commit", {})
            record = commit.get("record", {})
            if record.get("$type") != "app.bsky.feed.post":
                return

            text = record.get("text", "")
            if not any(term in text.lower() for term in CRISIS_TERMS):
                return

            did  = event.get("did", "")
            rkey = commit.get("rkey", "")
            payload = {
                "post_id":    f"{did}/{rkey}",
                "text":       text,
                "user":       did,
                "url":        f"https://bsky.app/profile/{did}/post/{rkey}",
                "image_urls": [],
            }
            worker._produce_with_retry(payload)

        def on_error(ws, error):
            logger.error("bluesky ws error: %s", error)

        def on_close(ws, close_status_code, close_msg):
            logger.info("bluesky ws closed: %s %s", close_status_code, close_msg)

        ws = websocket.WebSocketApp(
            JETSTREAM_URL,
            on_message=on_message,
            on_error=on_error,
            on_close=on_close,
        )
        ws.run_forever()

        # generator protocol
        return
        yield


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = BlueskyWorker()
    try:
        w.run()
    finally:
        w.close()
