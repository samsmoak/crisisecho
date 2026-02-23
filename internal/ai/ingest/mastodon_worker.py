"""
mastodon_worker.py — Mastodon instance streaming API → social_raw / mastodon
"""

import logging
import os

from mastodon import Mastodon, StreamListener

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

CRISIS_TERMS = {
    "earthquake", "flood", "wildfire", "tornado", "hurricane",
    "tsunami", "explosion", "shooting", "evacuation", "emergency",
    "disaster", "shelter", "rescue",
}


class MastodonWorker(KafkaWorker):
    topic  = "social_raw"
    source = "mastodon"

    def __init__(self) -> None:
        super().__init__()
        self.mastodon = Mastodon(
            access_token  = os.environ["MASTODON_ACCESS_TOKEN"],
            api_base_url  = os.environ.get("MASTODON_INSTANCE_URL", "https://mastodon.social"),
        )

    def stream(self):
        """
        Connects to the Mastodon public streaming timeline and yields payloads
        for posts that contain at least one crisis-related keyword.
        Uses the streaming listener pattern (callback-based, non-generator).
        """
        worker = self

        class _Listener(StreamListener):
            def on_update(self, status):
                # Strip HTML tags from the content field
                import re
                text = re.sub(r"<[^>]+>", "", status.get("content", ""))
                if not any(term in text.lower() for term in CRISIS_TERMS):
                    return

                account = status.get("account", {})
                media   = status.get("media_attachments", [])
                payload = {
                    "post_id":    str(status["id"]),
                    "text":       text,
                    "user":       account.get("acct", ""),
                    "url":        status.get("url", ""),
                    "image_urls": [m["url"] for m in media if m.get("type") == "image"],
                }
                worker._produce_with_retry(payload)

        # Blocks until connection drops; caller should reconnect on exception.
        self.mastodon.stream_public(_Listener())

        return
        yield


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = MastodonWorker()
    try:
        w.run()
    finally:
        w.close()
