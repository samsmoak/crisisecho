"""
telegram_worker.py — Telethon channel monitor → social_raw / telegram
"""

import asyncio
import logging
import os

from telethon import TelegramClient, events
from telethon.tl.types import MessageMediaPhoto, MessageMediaDocument

from .kafka_worker_base import KafkaWorker

logger = logging.getLogger(__name__)

# Public Telegram channels / groups known for emergency/crisis reporting.
# Operators should extend this list with relevant local channels.
CHANNELS = [
    "disasteralerts",
    "earthquakealerts",
    "weatheralerts",
]

CRISIS_TERMS = {
    "earthquake", "flood", "wildfire", "tornado", "hurricane",
    "tsunami", "explosion", "shooting", "evacuation", "emergency",
    "disaster", "shelter", "rescue",
}


class TelegramWorker(KafkaWorker):
    topic  = "social_raw"
    source = "telegram"

    def __init__(self) -> None:
        super().__init__()
        self.bot_token = os.environ["TELEGRAM_BOT_TOKEN"]
        self.api_id    = int(os.environ.get("TELEGRAM_API_ID", "0"))
        self.api_hash  = os.environ.get("TELEGRAM_API_HASH", "")

    def stream(self):
        """
        Uses Telethon's async event handler.  Runs the asyncio event loop
        internally; the generator returns immediately (callbacks handle produce).
        """
        worker = self

        async def _run():
            client = TelegramClient(
                "crisisecho_telegram_session",
                self.api_id,
                self.api_hash,
            )
            await client.start(bot_token=self.bot_token)

            @client.on(events.NewMessage(chats=CHANNELS))
            async def handler(event):
                text = event.message.text or ""
                if not any(term in text.lower() for term in CRISIS_TERMS):
                    return

                msg    = event.message
                chat   = await event.get_chat()
                images = []
                if isinstance(msg.media, (MessageMediaPhoto, MessageMediaDocument)):
                    images = []  # URL not directly available; placeholder

                payload = {
                    "post_id":    str(msg.id),
                    "text":       text,
                    "user":       str(msg.sender_id or ""),
                    "url":        f"https://t.me/{getattr(chat, 'username', '')}/{msg.id}",
                    "image_urls": images,
                }
                worker._produce_with_retry(payload)

            await client.run_until_disconnected()

        asyncio.run(_run())

        return
        yield


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    w = TelegramWorker()
    try:
        w.run()
    finally:
        w.close()
