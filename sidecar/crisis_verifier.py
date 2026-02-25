"""
crisis_verifier.py — Cross-source crisis verification logic.

Determines whether a cluster of posts constitutes a verified crisis
based on source diversity, user diversity, official signals, and
image corroboration.
"""

import logging
from dataclasses import dataclass

from image_verifier import ImageCorroborationResult

logger = logging.getLogger(__name__)

# Thresholds for the primary evidence path
MIN_DISTINCT_SOURCES = 2
MIN_DISTINCT_USERS   = 3
# Thresholds for the image corroboration path
MIN_USERS_WITH_IMAGE = 2


@dataclass
class VerificationResult:
    verified: bool
    confidence_score: float       # 0.0 – 1.0
    evidence_summary: str


class CrisisVerifier:
    """
    Pure-logic verifier: takes a list of cluster posts plus image and
    official-signal context, and returns a VerificationResult.
    """

    def verify(
        self,
        cluster_posts: list[dict],
        image_corroboration: ImageCorroborationResult,
        official_signal: bool = False,
    ) -> VerificationResult:
        """
        Verification rules:
          PASS if:
            (distinct_sources >= 2 AND distinct_users >= 3)
            OR official_signal == True
            OR (image_corroboration.corroborated AND distinct_users >= 2)

        confidence_score is a weighted sum of the available evidence.
        """
        distinct_sources = len({p.get("source", "") for p in cluster_posts if p.get("source")})
        distinct_users   = len({p.get("user", "")   for p in cluster_posts if p.get("user")})

        # Evaluate each evidence path
        social_path  = distinct_sources >= MIN_DISTINCT_SOURCES and distinct_users >= MIN_DISTINCT_USERS
        official_path = official_signal
        image_path   = image_corroboration.corroborated and distinct_users >= MIN_USERS_WITH_IMAGE

        verified = social_path or official_path or image_path

        # Confidence calculation — additive evidence weights
        confidence = 0.0
        evidence_parts: list[str] = []

        if social_path:
            confidence += 0.5
            evidence_parts.append(
                f"social corroboration ({distinct_sources} sources, {distinct_users} users)"
            )
        if official_path:
            confidence += 0.4
            evidence_parts.append("official signal")
        if image_path:
            # Partial if social_path already counted; avoid > 1.0
            confidence = min(1.0, confidence + 0.2)
            evidence_parts.append(
                f"image corroboration ({image_corroboration.match_count} matching posts)"
            )

        # Partial credit for near-threshold evidence
        if not verified:
            if distinct_sources >= 1 and distinct_users >= 2:
                confidence = 0.25
                evidence_parts.append("partial social evidence")

        evidence_summary = "; ".join(evidence_parts) if evidence_parts else "insufficient evidence"

        return VerificationResult(
            verified=verified,
            confidence_score=round(min(1.0, confidence), 3),
            evidence_summary=evidence_summary,
        )
