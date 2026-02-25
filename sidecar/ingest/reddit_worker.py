"""
reddit_worker.py — PRAW subreddit stream → social_raw / reddit
"""

import logging
import os

import praw

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

# Subreddits relevant to crisis/emergency reporting
SUBREDDITS = [
    "news", "worldnews", "earthquake", "weather",
    "emergency", "wildfires", "flooding", "hurricanes",
]


class RedditWorker(KafkaWorker):
    topic  = "social_raw"
    source = "reddit"

    def __init__(self) -> None:
        super().__init__()
        self.reddit = praw.Reddit(
            client_id     = os.environ["REDDIT_CLIENT_ID"],
            client_secret = os.environ["REDDIT_CLIENT_SECRET"],
            user_agent    = "crisisecho:ingest:v1 (by /u/crisisecho_bot)",
        )

    def stream(self):
        """Yields one payload dict per new submission from the watched subreddits."""
        multi = self.reddit.subreddit("+".join(SUBREDDITS))
        for submission in multi.stream.submissions(skip_existing=True):
            yield {
                "post_id":    submission.id,
                "text":       f"{submission.title}\n{submission.selftext}".strip(),
                "user":       str(submission.author) if submission.author else "",
                "url":        f"https://www.reddit.com{submission.permalink}",
                "image_urls": [submission.url] if submission.url.endswith((".jpg", ".jpeg", ".png", ".gif")) else [],
            }


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = RedditWorker()
    try:
        w.run()
    finally:
        w.close()
