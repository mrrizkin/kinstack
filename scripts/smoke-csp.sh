#!/usr/bin/env bash
# Smoke-test the CSP document shell with the real application binary.
# Requires PostgreSQL/Valkey/`.env` and a production build in public/build.
set -euo pipefail

cd "$(dirname "$0")/.."

if [[ -f public/hot ]]; then
  echo "public/hot exists; stop the Vite dev server before this smoke test." >&2
  exit 1
fi
if [[ ! -f public/build/manifest.json ]]; then
  echo "public/build/manifest.json missing; run: bun x vite build" >&2
  exit 1
fi

port="${SMOKE_PORT:-3120}"
binary="$(mktemp)"
body="$(mktemp)"
headers="$(mktemp)"
log="$(mktemp)"
server_pid=""

cleanup() {
  if [[ -n "$server_pid" ]]; then
    kill -- "-$server_pid" 2>/dev/null || kill "$server_pid" 2>/dev/null || true
  fi
  rm -f "$binary" "$body" "$headers" "$log"
}
trap cleanup EXIT

go build -trimpath -o "$binary" ./cmd/kinstack
setsid env APP_PORT="$port" CSP_ENABLED=true CSP_REPORT_ONLY=false \
  LOG_LEVEL=error LOG_DIR=/tmp/kinstack-smoke-logs "$binary" >"$log" 2>&1 &
server_pid=$!

for _ in $(seq 1 40); do
  if curl -fsS "http://127.0.0.1:$port/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done
if ! curl -fsS "http://127.0.0.1:$port/health" >/dev/null; then
  echo "application did not become healthy:" >&2
  cat "$log" >&2
  exit 1
fi

curl -fsS -D "$headers" -o "$body" "http://127.0.0.1:$port/login"
nonce="$(tr -d '\r' <"$headers" | sed -n "s/.*'nonce-\([^']*\)'.*/\1/p" | head -1)"
if [[ -z "$nonce" ]]; then
  echo "CSP header has no nonce:" >&2
  tr -d '\r' <"$headers" >&2
  exit 1
fi
if ! grep -q "report-uri /csp-report" "$headers"; then
  echo "CSP header has no report-uri:" >&2
  tr -d '\r' <"$headers" >&2
  exit 1
fi
if grep -q 'nonce=""' "$body"; then
  echo "rendered HTML contains an empty nonce" >&2
  exit 1
fi
script_nonces="$(grep -o 'nonce="[^"]*"' "$body" | sed 's/nonce="//;s/"//' | sort -u)"
if [[ "$script_nonces" != "$nonce" ]]; then
  echo "script nonce mismatch; header=$nonce html=$script_nonces" >&2
  exit 1
fi
report_status="$(curl -sS -o /dev/null -w "%{http_code}" -X POST \
  -H "Content-Type: application/csp-report" \
  --data '{"csp-report":{"violated-directive":"script-src"}}' \
  "http://127.0.0.1:$port/csp-report")"
if [[ "$report_status" != "204" ]]; then
  echo "CSP report endpoint returned $report_status, want 204" >&2
  exit 1
fi
echo "CSP smoke passed: nonce=$nonce report=204"
