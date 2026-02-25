"""
twitter_worker.py — Tweepy filtered stream → social_raw / twitter
"""

import logging
import os

import tweepy

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

# Crisis-related keywords for the filtered stream rule
CRISIS_KEYWORDS = (
    "earthquake OR flood OR wildfire OR tornado OR hurricane OR "
    "tsunami OR explosion OR shooting OR evacuation OR emergency"
)


class TwitterWorker(KafkaWorker):
    topic  = "social_raw"
    source = "twitter"

    def __init__(self) -> None:
        super().__init__()
        self.bearer_token = os.environ["TWITTER_BEARER_TOKEN"]

    def stream(self):
        """
        Uses Tweepy's StreamingClient (Twitter API v2 filtered stream).
        Yields payload dicts for each matching tweet.
        """
        worker = self
        payloads = []  # collector for the generator bridge

        class _Handler(tweepy.StreamingClient):
            def on_tweet(self, tweet):
                payload = {
                    "post_id":    str(tweet.id),
                    "text":       tweet.text,
                    "user":       tweet.author_id or "",
                    "url":        f"https://twitter.com/i/web/status/{tweet.id}",
                    "image_urls": [],
                }
                worker._produce_with_retry(payload)

            def on_errors(self, errors):
                logger.error("twitter stream errors: %s", errors)

            def on_exception(self, exception):
                logger.error("twitter stream exception: %s", exception)

        client = _Handler(self.bearer_token)

        # Replace existing rules to avoid duplicate costs.
        existing = client.get_rules()
        if existing.data:
            ids = [r.id for r in existing.data]
            client.delete_rules(ids)
        client.add_rules(tweepy.StreamRule(CRISIS_KEYWORDS))

        # filter() blocks; we never reach the line below while the stream is up.
        client.filter(tweet_fields=["author_id", "text"])

        # generator protocol: yield nothing (streaming is handled by callbacks)
        return
        yield  # make this a generator


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = TwitterWorker()
    try:
        w.run()
    finally:
        w.close()
