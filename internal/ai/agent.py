"""
agent.py — LangChain LCEL 3-step crisis detection agent.

Primary LLM:  Claude claude-haiku-4-5-20251001 (Anthropic API)
Fallback LLM: Ollama Llama3 (when ANTHROPIC_API_KEY is not set)

Chain pipeline:
  1. cluster_chain  — identify distinct events in post batch
  2. severity_chain — score each cluster 1-5
  3. alert_chain    — write public alert text

Outputs are written to MongoDB (clusters, crises, alerts) and
published to Redis alerts:live.
"""

import json
import logging
import os
import re
from dataclasses import dataclass, field
from datetime import datetime, timezone

import pymongo
import redis as redis_lib
from bson import ObjectId

from image_verifier import ImageVerifier, ImageCorroborationResult
from crisis_verifier import CrisisVerifier, VerificationResult
from retrieval import RetrievalResult

logger = logging.getLogger(__name__)

# ── LLM setup ────────────────────────────────────────────────────────────────

def _build_llm():
    api_key = os.environ.get("ANTHROPIC_API_KEY", "")
    if api_key:
        from langchain_anthropic import ChatAnthropic
        return ChatAnthropic(
            model="claude-haiku-4-5-20251001",
            anthropic_api_key=api_key,
            max_tokens=2048,
            temperature=0.2,
        )
    # Fallback: Ollama Llama3
    try:
        from langchain_community.llms import Ollama
        logger.warning("ANTHROPIC_API_KEY not set — falling back to Ollama llama3")
        return Ollama(model="llama3")
    except Exception as exc:
        logger.error("LLM init failed: %s", exc)
        return None


# ── Data classes ─────────────────────────────────────────────────────────────

@dataclass
class AlertPayload:
    cluster_id:           str
    unified_post_id:      str
    crisis_id:            str
    alert_id:             str
    event_type:           str
    location_description: str
    severity:             int
    alert_text:           str
    verification:         VerificationResult
    sources:              list[str] = field(default_factory=list)
    lat:                  float = 0.0
    lng:                  float = 0.0


# ── Prompt templates ─────────────────────────────────────────────────────────

CLUSTER_SYSTEM = (
    "You are a crisis detection expert. Analyse social media posts and identify "
    "distinct real-world emergency events. Respond ONLY with a JSON array."
)

CLUSTER_HUMAN = (
    "Given these {n} social media posts from the past two hours near {location}:\n\n"
    "{posts_text}\n\n"
    "Identify distinct real-world events. For each event output a JSON object with:\n"
    "  event_type (string), location_description (string), estimated_time (ISO-8601 string),\n"
    "  contributing_post_ids (array of post_id strings), confidence_score (float 0-1).\n"
    "Output a JSON array only. No prose."
)

SEVERITY_SYSTEM = (
    "You are a crisis severity analyst. Rate events on a 1-5 scale. "
    "Respond ONLY with a JSON object."
)

SEVERITY_HUMAN = (
    "Given this cluster of {n} posts describing '{event_type}' near {location}:\n\n"
    "{posts_text}\n\n"
    "Rate severity 1-5:\n"
    "  1=unconfirmed minor, 2=possible minor, 3=confirmed moderate,\n"
    "  4=confirmed major, 5=confirmed mass casualty or critical infrastructure failure.\n"
    "Consider: language urgency, independent reporter count ({contributor_count}), "
    "official corroboration ({official_corroboration}), image evidence ({image_corroborated}).\n"
    "Output JSON only: {{\"severity_score\": int, \"rationale\": \"two sentences\"}}"
)

ALERT_SYSTEM = (
    "You are a public safety communications officer. Write clear, calm, factual alerts. "
    "Never include usernames or unverified speculation."
)

ALERT_HUMAN = (
    "Write a 2-3 sentence public alert about this severity-{severity}/5 "
    "{event_type} event near {location}.\n"
    "Be factual, calm, and actionable. Do not include usernames.\n"
    "End with: Sources: {platform_list}"
)

DIGEST_SYSTEM = "You are a crisis information analyst. Answer questions about active crisis events factually."

DIGEST_HUMAN = (
    "Given these crisis clusters near {location}:\n\n{clusters_text}\n\n"
    "Answer the following question: {question}\n"
    "Be factual and cite sources. Keep response under 200 words."
)


class CrisisAgent:
    """
    Orchestrates the 3-step LLM pipeline for a batch of retrieved posts.
    """

    CONFIDENCE_THRESHOLD = 0.6

    def __init__(self, mongo_main_client, mongo_vector_client) -> None:
        main_db_name   = os.environ.get("MONGO_DB_DATABASE", "crisisecho")
        self.main_db   = mongo_main_client[main_db_name]

        redis_url = os.environ.get("REDIS_URL", "redis://localhost:6379")
        self._redis = redis_lib.from_url(redis_url)

        self._image_verifier  = ImageVerifier(mongo_vector_client)
        self._crisis_verifier = CrisisVerifier()
        self._llm             = _build_llm()
        self._chains          = self._build_chains() if self._llm else None

    # ── Public API ───────────────────────────────────────────────────────────

    def run(
        self,
        retrieval_result: RetrievalResult,
        lat: float,
        lng: float,
    ) -> list[AlertPayload]:
        if not self._chains:
            logger.error("LLM unavailable — agent cannot run")
            return []

        posts          = retrieval_result.posts
        official_posts = retrieval_result.official_context
        if not posts:
            return []

        location_str   = f"{lat:.4f}, {lng:.4f}"
        posts_text     = self._posts_to_text(posts[:50])
        official_signal = len(official_posts) > 0

        # Step 1: Identify clusters
        clusters = self._run_cluster_chain(len(posts), location_str, posts_text)
        if not clusters:
            return []

        results: list[AlertPayload] = []

        for cluster_spec in clusters:
            if float(cluster_spec.get("confidence_score", 0)) < self.CONFIDENCE_THRESHOLD:
                continue

            event_type    = cluster_spec.get("event_type", "unknown")
            location_desc = cluster_spec.get("location_description", location_str)
            post_ids      = cluster_spec.get("contributing_post_ids", [])
            cluster_posts = [p for p in posts if p.get("post_id", "") in post_ids] or posts[:10]

            # Step 2: Severity
            severity_result = self._run_severity_chain(
                cluster_posts, event_type, location_desc, official_signal
            )
            severity_score    = severity_result.get("severity_score", 1)
            severity_rationale = severity_result.get("rationale", "")

            if severity_score < 3:
                continue

            # Image verification
            img_corroboration = self._image_verifier.verify_images(post_ids, lat, lng)

            # Crisis verification
            verification = self._crisis_verifier.verify(
                cluster_posts, img_corroboration, official_signal
            )

            if not verification.verified:
                continue

            # Step 3: Alert text
            sources       = list({p.get("source", "") for p in cluster_posts if p.get("source")})
            platform_list = ", ".join(sources)
            alert_text    = self._run_alert_chain(
                severity_score, event_type, location_desc, platform_list
            )

            # Compute centroid — use only GPS-sourced posts with full confidence
            gps_posts = [p for p in cluster_posts
                         if p.get("location_source") == "gps"
                         and p.get("location_confidence", 0.0) == 1.0]
            # Fall back to trigger coordinates if no GPS posts available
            if gps_posts:
                clat = sum(p.get("location", {}).get("coordinates", [0, 0])[1] for p in gps_posts) / len(gps_posts)
                clng = sum(p.get("location", {}).get("coordinates", [0, 0])[0] for p in gps_posts) / len(gps_posts)
            else:
                clat = lat
                clng = lng

            # Persist to MongoDB
            cluster_id = self._write_cluster(
                cluster_posts, event_type, severity_score, severity_rationale,
                cluster_spec.get("estimated_time"), sources, official_signal,
                clat, clng, verification,
            )
            # Write the unified post — the system's synthesised view of this cluster
            unified_post_id = self._write_unified_post(
                cluster_id=cluster_id,
                cluster_posts=cluster_posts,
                event_type=event_type,
                location_desc=location_desc,
                severity=severity_score,
                severity_rationale=severity_rationale,
                sources=sources,
                official_signal=official_signal,
                clat=clat,
                clng=clng,
                verification=verification,
            )
            # Create crisis only if the unified post is verified
            crisis_id = self._write_crisis(
                unified_post_id=unified_post_id,
                event_type=event_type,
                clat=clat,
                clng=clng,
                severity=severity_score,
                sources=sources,
                description=f"{event_type} near {location_desc}. {severity_rationale}",
            )
            alert_id = self._write_alert(
                cluster_id, alert_text, severity_score, event_type,
                clat, clng, sources,
            )

            # Publish to Redis
            self._publish_alert(alert_id, alert_text, severity_score, event_type, clat, clng, sources)

            results.append(AlertPayload(
                cluster_id=cluster_id,
                unified_post_id=unified_post_id,
                crisis_id=crisis_id,
                alert_id=alert_id,
                event_type=event_type,
                location_description=location_desc,
                severity=severity_score,
                alert_text=alert_text,
                verification=verification,
                sources=sources,
                lat=clat,
                lng=clng,
            ))

        return results

    def answer_query(self, question: str, lat: float, lng: float) -> tuple[str, list[dict]]:
        """
        Run the digest chain to answer a natural-language question about
        active crisis clusters near the given coordinates.
        Returns (digest_text, clusters_list).
        """
        if not self._chains:
            return "AI pipeline unavailable.", []

        # Fetch recent clusters near the location
        try:
            clusters = list(self.main_db.clusters.find(
                {"status": "active"},
                limit=10,
            ))
        except Exception as exc:
            logger.warning("cluster fetch error: %s", exc)
            clusters = []

        clusters_text = json.dumps(
            [self._cluster_to_dict(c) for c in clusters],
            default=str, indent=2,
        )
        location_str = f"{lat:.4f}, {lng:.4f}"

        digest_chain = self._chains.get("digest")
        if not digest_chain:
            return "Digest chain unavailable.", clusters

        try:
            digest = digest_chain.invoke({
                "location":      location_str,
                "clusters_text": clusters_text,
                "question":      question,
            })
        except Exception as exc:
            logger.error("digest chain error: %s", exc)
            return "Error generating digest.", clusters

        return digest, clusters

    # ── Chain builders ────────────────────────────────────────────────────────

    def _build_chains(self) -> dict:
        from langchain_core.prompts import ChatPromptTemplate
        from langchain_core.output_parsers import StrOutputParser

        parser = StrOutputParser()

        cluster_prompt  = ChatPromptTemplate.from_messages([
            ("system", CLUSTER_SYSTEM),
            ("human",  CLUSTER_HUMAN),
        ])
        severity_prompt = ChatPromptTemplate.from_messages([
            ("system", SEVERITY_SYSTEM),
            ("human",  SEVERITY_HUMAN),
        ])
        alert_prompt = ChatPromptTemplate.from_messages([
            ("system", ALERT_SYSTEM),
            ("human",  ALERT_HUMAN),
        ])
        digest_prompt = ChatPromptTemplate.from_messages([
            ("system", DIGEST_SYSTEM),
            ("human",  DIGEST_HUMAN),
        ])

        return {
            "cluster":  cluster_prompt  | self._llm | parser,
            "severity": severity_prompt | self._llm | parser,
            "alert":    alert_prompt    | self._llm | parser,
            "digest":   digest_prompt   | self._llm | parser,
        }

    # ── Chain runners ─────────────────────────────────────────────────────────

    def _run_cluster_chain(self, n: int, location: str, posts_text: str) -> list[dict]:
        try:
            raw = self._chains["cluster"].invoke({
                "n":          n,
                "location":   location,
                "posts_text": posts_text,
            })
            return self._parse_json_list(raw)
        except Exception as exc:
            logger.error("cluster_chain error: %s", exc)
            return []

    def _run_severity_chain(
        self,
        cluster_posts: list[dict],
        event_type: str,
        location: str,
        official_signal: bool,
    ) -> dict:
        n             = len(cluster_posts)
        posts_text    = self._posts_to_text(cluster_posts)
        contributor_count = len({p.get("user", "") for p in cluster_posts if p.get("user")})
        image_count   = sum(1 for p in cluster_posts if p.get("image_urls"))
        try:
            raw = self._chains["severity"].invoke({
                "n":                    n,
                "event_type":           event_type,
                "location":             location,
                "posts_text":           posts_text,
                "contributor_count":    contributor_count,
                "official_corroboration": official_signal,
                "image_corroborated":   image_count > 0,
            })
            return self._parse_json_obj(raw)
        except Exception as exc:
            logger.error("severity_chain error: %s", exc)
            return {"severity_score": 1, "rationale": "Severity assessment failed."}

    def _run_alert_chain(
        self,
        severity: int,
        event_type: str,
        location: str,
        platform_list: str,
    ) -> str:
        try:
            return self._chains["alert"].invoke({
                "severity":      severity,
                "event_type":    event_type,
                "location":      location,
                "platform_list": platform_list,
            })
        except Exception as exc:
            logger.error("alert_chain error: %s", exc)
            return f"Emergency alert: {event_type} reported near {location}. Sources: {platform_list}"

    # ── MongoDB writes ────────────────────────────────────────────────────────

    def _write_cluster(
        self,
        posts: list[dict],
        event_type: str,
        severity: int,
        severity_rationale: str,
        estimated_time,
        sources: list[str],
        official_corroboration: bool,
        lat: float,
        lng: float,
        verification: VerificationResult,
    ) -> str:
        now = datetime.now(timezone.utc)
        doc = {
            "centroid":               {"type": "Point", "coordinates": [lng, lat]},
            "affected_area":          {"type": "Polygon", "coordinates": [[]]},
            "crisis_type":            event_type,
            "severity":               severity,
            "severity_rationale":     severity_rationale,
            "summary":                f"{event_type} cluster with {len(posts)} posts",
            "post_count":             len(posts),
            "contributor_count":      len({p.get("user", "") for p in posts if p.get("user")}),
            "post_ids":               [],
            "sources":                sources,
            "official_corroboration": official_corroboration,
            "location_confidence":    verification.confidence_score,
            "start_time":             now,
            "last_updated":           now,
            "status":                 "active",
        }
        result = self.main_db.clusters.insert_one(doc)
        return str(result.inserted_id)

    def _write_unified_post(
        self,
        cluster_id: str,
        cluster_posts: list[dict],
        event_type: str,
        location_desc: str,
        severity: int,
        severity_rationale: str,
        sources: list[str],
        official_signal: bool,
        clat: float,
        clng: float,
        verification: VerificationResult,
    ) -> str:
        now = datetime.now(timezone.utc)
        post_ids = [p.get("post_id", str(p.get("_id", ""))) for p in cluster_posts]
        contributor_count = len({p.get("user", "") for p in cluster_posts if p.get("user")})
        summary = f"{event_type} near {location_desc}. {severity_rationale}"
        doc = {
            "cluster_id":             ObjectId(cluster_id),
            "event_type":             event_type,
            "summary":                summary,
            "location":               {"type": "Point", "coordinates": [clng, clat]},
            "lat":                    clat,
            "lng":                    clng,
            "severity":               severity,
            "confidence_score":       verification.confidence_score,
            "sources":                sources,
            "contributor_count":      contributor_count,
            "official_corroboration": official_signal,
            "post_ids":               post_ids,
            "verified":               verification.verified,
            "created_at":             now,
            "updated_at":             now,
        }
        result = self.main_db.unified_posts.insert_one(doc)
        return str(result.inserted_id)

    def _write_crisis(
        self,
        unified_post_id: str,
        event_type: str,
        clat: float,
        clng: float,
        severity: int,
        sources: list[str],
        description: str,
    ) -> str:
        now = datetime.now(timezone.utc)
        doc = {
            "unified_post_id": ObjectId(unified_post_id),
            "event":           event_type,
            "location":        {"type": "Point", "coordinates": [clng, clat]},
            "lat":             clat,
            "lng":             clng,
            "severity":        severity,
            "confirmed":       True,
            "sources":         sources,
            "description":     description,
            "image_urls":      [],
            "start_time":      now,
            "last_updated":    now,
        }
        result = self.main_db.crises.insert_one(doc)
        return str(result.inserted_id)

    def _write_alert(
        self,
        cluster_id: str,
        alert_text: str,
        severity: int,
        crisis_type: str,
        lat: float,
        lng: float,
        sources: list[str],
    ) -> str:
        now = datetime.now(timezone.utc)
        doc = {
            "cluster_id":       ObjectId(cluster_id),
            "alert_text":       alert_text,
            "severity":         severity,
            "crisis_type":      crisis_type,
            "centroid":         {"type": "Point", "coordinates": [lng, lat]},
            "published_at":     now,
            "source_platforms": sources,
            "notified_users":   [],
        }
        result = self.main_db.alerts.insert_one(doc)
        return str(result.inserted_id)

    # ── Redis publish ─────────────────────────────────────────────────────────

    def _publish_alert(
        self,
        alert_id: str,
        alert_text: str,
        severity: int,
        crisis_type: str,
        lat: float,
        lng: float,
        sources: list[str],
    ) -> None:
        payload = json.dumps({
            "id":          alert_id,
            "text":        alert_text,
            "severity":    severity,
            "crisis_type": crisis_type,
            "lat":         lat,
            "lng":         lng,
            "sources":     sources,
            "published_at": datetime.now(timezone.utc).isoformat(),
        })
        try:
            self._redis.publish("alerts:live", payload)
        except Exception as exc:
            logger.warning("Redis publish error (non-critical): %s", exc)

    # ── Utilities ─────────────────────────────────────────────────────────────

    @staticmethod
    def _posts_to_text(posts: list[dict]) -> str:
        lines = []
        for i, p in enumerate(posts[:50], 1):
            src  = p.get("source", "unknown")
            pid  = p.get("post_id", "?")
            text = p.get("text", p.get("cleaned_text", ""))[:300]
            lines.append(f"[{i}] [{src}:{pid}] {text}")
        return "\n".join(lines)

    @staticmethod
    def _cluster_to_dict(c: dict) -> dict:
        """Convert a MongoDB cluster document to a JSON-serializable dict."""
        return {
            "id":          str(c.get("_id", "")),
            "crisis_type": c.get("crisis_type", ""),
            "severity":    c.get("severity", 0),
            "summary":     c.get("summary", ""),
            "sources":     c.get("sources", []),
            "status":      c.get("status", ""),
        }

    @staticmethod
    def _parse_json_list(raw: str) -> list[dict]:
        """Extract a JSON array from LLM output, tolerating surrounding prose."""
        match = re.search(r"\[.*\]", raw, re.DOTALL)
        if not match:
            return []
        try:
            return json.loads(match.group(0))
        except json.JSONDecodeError:
            return []

    @staticmethod
    def _parse_json_obj(raw: str) -> dict:
        """Extract a JSON object from LLM output."""
        match = re.search(r"\{.*\}", raw, re.DOTALL)
        if not match:
            return {}
        try:
            return json.loads(match.group(0))
        except json.JSONDecodeError:
            return {}
