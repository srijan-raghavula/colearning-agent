#!/usr/bin/env bash
# Boots the API, posts a tutor turn, kills the server, restarts and verifies
# the learning state survived the restart via the durable snapshot (Phase 2).
set -euo pipefail

PORT=${PORT:-18099}
STORE_DIR=$(mktemp -d /tmp/opencode/learning-smoke.XXXXXX)
trap 'fuser -k "${PORT}"/tcp 2>/dev/null || true; rm -rf "$STORE_DIR"' EXIT

BASE="http://localhost:${PORT}"

boot() {
  LEARNING_STORE_PATH="$STORE_DIR/learning-state.json" PORT="$PORT" \
    nohup go run ./src/cmd/api > "$1" 2>&1 &
  for _ in $(seq 1 40); do
    sleep 0.25
    curl -sf -o /dev/null "${BASE}/healthz" && return 0
  done
  echo "server failed to start"; cat "$1"; exit 1
}

echo "▶ boot 1 (seed + learn turn)"
boot "$STORE_DIR/boot1.log"
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST \
  "$BASE/api/v1/student/sessions/session-s001-linear/turns" \
  -H 'Content-Type: application/json' -H 'X-Role: student' \
  -d '{"message":"Smoke check: I will verify by substituting the answer back."}')
[[ "$code" == "200" ]] || { echo "turn POST failed (HTTP $code)"; exit 1; }
fuser -k "${PORT}"/tcp 2>/dev/null || true
sleep 1

echo "▶ boot 2 (restore from snapshot)"
boot "$STORE_DIR/boot2.log"
grep -q "restored learning state" "$STORE_DIR/boot2.log" \
  || { echo "server did not log a successful restore"; cat "$STORE_DIR/boot2.log"; exit 1; }

count=$(curl -s "$BASE/api/v1/student/dashboard" -H 'X-Role: student' | \
  python3 -c 'import json,sys; d=json.load(sys.stdin); print(len(d.get("turns") or []))')
echo "▶ dashboard turns after restore: $count"
[[ "$count" -ge 4 ]] || { echo "expected ≥4 turns after restore (seeds + smoke turn), got $count"; exit 1; }

echo "✅ smoke ok: learning state survived a restart"
