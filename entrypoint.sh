#!/usr/bin/env bash
# entrypoint.sh — Start Python pipeline sidecar + Go API concurrently.
# Both processes are monitored; if either exits unexpectedly the container stops.

set -euo pipefail

# ── Ensure Go binary is executable ───────────────────────────────────────────
: "${PORT:=8080}"
: "${PYTHON_SIDECAR_PORT:=8081}"

# ── Start Python pipeline sidecar in background ───────────────────────────────
echo "crisisecho: starting Python sidecar on port ${PYTHON_SIDECAR_PORT}..."
PYTHON_SIDECAR_PORT="${PYTHON_SIDECAR_PORT}" \
  python /app/internal/ai/pipeline.py &
PYTHON_PID=$!

# Give the sidecar a moment to start before Go tries to connect
sleep 3

# ── Start Go API ──────────────────────────────────────────────────────────────
echo "crisisecho: starting Go API on port ${PORT}..."
PYTHON_SIDECAR_URL="http://localhost:${PYTHON_SIDECAR_PORT}" \
  /app/crisisecho &
GO_PID=$!

# ── Monitor both processes ────────────────────────────────────────────────────
# If either process dies, kill the other and exit with its code.
wait_and_check() {
  while true; do
    if ! kill -0 "${PYTHON_PID}" 2>/dev/null; then
      echo "crisisecho: Python sidecar (pid ${PYTHON_PID}) exited — stopping container"
      kill "${GO_PID}" 2>/dev/null || true
      exit 1
    fi
    if ! kill -0 "${GO_PID}" 2>/dev/null; then
      echo "crisisecho: Go API (pid ${GO_PID}) exited — stopping container"
      kill "${PYTHON_PID}" 2>/dev/null || true
      exit 1
    fi
    sleep 5
  done
}

trap 'kill "${PYTHON_PID}" "${GO_PID}" 2>/dev/null; exit 0' SIGTERM SIGINT

wait_and_check
