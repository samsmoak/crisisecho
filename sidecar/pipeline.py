"""
pipeline.py — CrisisEcho pipeline orchestrator + Python sidecar HTTP server.

Responsibilities:
  1. APScheduler: run the retrieval→agent pipeline every 60 seconds
  2. Volume spike detection: >50 posts in 30 s from the same area → immediate run
  3. FastAPI sidecar on port 8081:
       POST /internal/pipeline/trigger — trigger pipeline for given lat/lng
       POST /internal/query           — answer a natural-language query
"""

import logging
import math
import os
import sys
import time
from collections import defaultdict
from datetime import datetime, timezone, timedelta

import pymongo
import uvicorn
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from fastapi import FastAPI
from fastapi.concurrency import run_in_threadpool
from pydantic import BaseModel

# ── path setup: allow importing sibling modules from internal/ai/ ────────────
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from retrieval      import HybridRetriever
from agent          import CrisisAgent

logger = logging.getLogger(__name__)
logging.basicConfig(level=logging.INFO)

# ── Environment ──────────────────────────────────────────────────────────────
MONGO_MAIN_URI     = os.environ.get("MONGO_URI",          "mongodb://localhost:27017")
MONGO_VECTOR_URI   = os.environ.get("MONGO_VECTOR_URI",   "mongodb://localhost:27017")
MONGO_LOCATION_URI = os.environ.get("MONGO_LOCATION_URI", "mongodb://localhost:27017")
MAIN_DB_NAME       = os.environ.get("MONGO_DB_DATABASE",          "crisisecho")
SIDECAR_PORT       = int(os.environ.get("PYTHON_SIDECAR_PORT",    "8081"))

# Default trigger coordinates (city centre of largest monitored area).
# Override per request via POST /internal/pipeline/trigger.
DEFAULT_LAT = float(os.environ.get("DEFAULT_LAT", "40.7128"))
DEFAULT_LNG = float(os.environ.get("DEFAULT_LNG", "-74.0060"))

# Volume spike detection
SPIKE_WINDOW_S    = 30
SPIKE_THRESHOLD   = 50
SPIKE_GRID_DEG    = 0.5           # ~55 km grid cells


# ── MongoDB clients (shared across requests) ─────────────────────────────────
_main_client     = pymongo.MongoClient(MONGO_MAIN_URI,     serverSelectionTimeoutMS=5000)
_vector_client   = pymongo.MongoClient(MONGO_VECTOR_URI,   serverSelectionTimeoutMS=5000)
_location_client = pymongo.MongoClient(MONGO_LOCATION_URI, serverSelectionTimeoutMS=5000)

retriever = HybridRetriever(_main_client, _vector_client, _location_client)
agent     = CrisisAgent(_main_client, _vector_client)


# ── Volume spike tracker ──────────────────────────────────────────────────────

class _SpikeTracker:
    """
    Tracks post counts per grid cell in a 30-second sliding window.
    Polls MongoDB for recent posts periodically.
    """

    def __init__(self) -> None:
        self._last_check = datetime.now(timezone.utc)
        self._triggered_cells: set = set()  # cells that already fired this minute

    def check(self) -> tuple[float, float] | None:
        """
        Returns (lat, lng) of the first grid cell that exceeded the spike
        threshold since the last check, or None.
        """
        now    = datetime.now(timezone.utc)
        cutoff = now - timedelta(seconds=SPIKE_WINDOW_S)
        main_db = _main_client[MAIN_DB_NAME]

        # Clear triggered cells at the start of each minute
        if now.second < 5:
            self._triggered_cells.clear()

        try:
            pipeline = [
                {"$match": {"timestamp": {"$gt": cutoff}}},
                {
                    "$group": {
                        "_id": {
                            "lat_bucket": {
                                "$multiply": [
                                    {"$floor": {"$divide": [
                                        {"$arrayElemAt": ["$location.coordinates", 1]},
                                        SPIKE_GRID_DEG,
                                    ]}},
                                    SPIKE_GRID_DEG,
                                ]
                            },
                            "lng_bucket": {
                                "$multiply": [
                                    {"$floor": {"$divide": [
                                        {"$arrayElemAt": ["$location.coordinates", 0]},
                                        SPIKE_GRID_DEG,
                                    ]}},
                                    SPIKE_GRID_DEG,
                                ]
                            },
                        },
                        "count": {"$sum": 1},
                    }
                },
                {"$match": {"count": {"$gt": SPIKE_THRESHOLD}}},
                {"$limit": 1},
            ]
            results = list(main_db.posts.aggregate(pipeline))
        except Exception as exc:
            logger.debug("spike tracker error: %s", exc)
            return None

        for row in results:
            lat = float(row["_id"]["lat_bucket"]) + SPIKE_GRID_DEG / 2
            lng = float(row["_id"]["lng_bucket"]) + SPIKE_GRID_DEG / 2
            cell_key = f"{lat:.2f},{lng:.2f}"
            if cell_key not in self._triggered_cells:
                self._triggered_cells.add(cell_key)
                logger.info("volume spike detected at cell (%s, %s) — triggering immediate run", lat, lng)
                return lat, lng

        return None


_spike_tracker = _SpikeTracker()


# ── Core pipeline run ─────────────────────────────────────────────────────────

def _run_pipeline(lat: float, lng: float) -> dict:
    """Synchronous pipeline: retrieve → agent → return summary."""
    start_ms = time.time() * 1000
    trigger_time = datetime.now(timezone.utc)

    logger.info("pipeline run: lat=%.4f lng=%.4f", lat, lng)

    try:
        result = retriever.retrieve(lat, lng, trigger_time)
    except Exception as exc:
        logger.error("retrieval error: %s", exc)
        return {"error": str(exc)}

    posts_retrieved = len(result.posts)

    try:
        alerts = agent.run(result, lat, lng)
    except Exception as exc:
        logger.error("agent error: %s", exc)
        alerts = []

    latency_ms = time.time() * 1000 - start_ms

    summary = {
        "posts_retrieved":  posts_retrieved,
        "clusters_found":   len(alerts),
        "alerts_published": len(alerts),
        "latency_ms":       round(latency_ms),
    }
    logger.info("pipeline done: %s", summary)
    return summary


async def _scheduled_run() -> None:
    """APScheduler callback — runs every 60 seconds."""
    # Check for volume spike first
    spike = _spike_tracker.check()
    if spike:
        spike_lat, spike_lng = spike
        await run_in_threadpool(_run_pipeline, spike_lat, spike_lng)

    # Scheduled run at default coordinates
    await run_in_threadpool(_run_pipeline, DEFAULT_LAT, DEFAULT_LNG)


# ── FastAPI application ───────────────────────────────────────────────────────

app = FastAPI(title="CrisisEcho Python Sidecar", version="1.0")
scheduler = AsyncIOScheduler()


@app.on_event("startup")
async def _startup():
    scheduler.add_job(_scheduled_run, "interval", seconds=60, id="pipeline_cron")
    scheduler.start()
    logger.info("APScheduler started — pipeline runs every 60 s")


@app.on_event("shutdown")
async def _shutdown():
    scheduler.shutdown(wait=False)


# ── Request/response models ───────────────────────────────────────────────────

class TriggerRequest(BaseModel):
    lat: float = DEFAULT_LAT
    lng: float = DEFAULT_LNG


class QueryRequest(BaseModel):
    text: str
    lat:  float = DEFAULT_LAT
    lng:  float = DEFAULT_LNG


class QueryResponse(BaseModel):
    digest:   str
    clusters: list[dict]


# ── Endpoints ─────────────────────────────────────────────────────────────────

@app.post("/internal/pipeline/trigger")
async def trigger_pipeline(req: TriggerRequest) -> dict:
    """Trigger an immediate pipeline run for the given coordinates."""
    summary = await run_in_threadpool(_run_pipeline, req.lat, req.lng)
    return summary


@app.post("/internal/query", response_model=QueryResponse)
async def run_query(req: QueryRequest) -> QueryResponse:
    """Answer a natural-language question about active crisis clusters."""
    digest, clusters = await run_in_threadpool(
        agent.answer_query, req.text, req.lat, req.lng
    )
    # Make clusters JSON-serializable (ObjectId → str)
    safe_clusters = []
    for c in clusters:
        safe = {k: str(v) if hasattr(v, "__str__") and not isinstance(v, (str, int, float, bool, list, dict, type(None))) else v
                for k, v in c.items()}
        safe_clusters.append(safe)

    return QueryResponse(digest=digest, clusters=safe_clusters)


@app.get("/health")
async def health() -> dict:
    return {"status": "ok", "scheduler_running": scheduler.running}


# ── Entry point ───────────────────────────────────────────────────────────────

if __name__ == "__main__":
    uvicorn.run(
        "pipeline:app",
        host="0.0.0.0",
        port=SIDECAR_PORT,
        log_level="info",
    )
