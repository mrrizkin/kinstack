#!/usr/bin/env bash
# Request-scoped logging (PILLARS.md §7). Every log line written while a request
# is in flight names that request. The binding is a function, not a habit:
# `log.ForRequest(logger, c.RequestID())` stamps the field once, so a new call
# site cannot forget it.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

fail=0
report=""

# 1. No hand-rolled request_id pairs. The field is the binder's job.
mapfile -t files < <(
  git ls-files --cached --others --exclude-standard -- 'internal/features/*.go' 'internal/bootstrap/*.go' |
    grep -v '_test\.go$' | grep -v '_templ\.go$'
)
for file in "${files[@]}"; do
  hits="$(grep -n '"request_id"' "$file" | grep -v 'json:"request_id"' || true)"
  if [[ -n "$hits" ]]; then
    report+="$file: request_id passed by hand, use log.ForRequest"$'\n'
    report+="$(printf '%s\n' "$hits" | sed 's/^/    /')"$'\n'
    fail=1
  fi
done

# 2. Every structured log call in a feature handler goes through the binder.
# Bootstrap logs (startup, shutdown, background workers) have no request and are
# out of scope by construction.
mapfile -t handlers < <(
  git ls-files --cached --others --exclude-standard -- 'internal/features/*/handler*.go' |
    grep -v '_test\.go$' | grep -v '_templ\.go$'
)
for file in "${handlers[@]}"; do
  [[ -f "$file" ]] || continue
  hits="$(grep -nE '\.(Errorw|Warnw|Infow|Debugw)\(' "$file" | grep -v 'ForRequest(' || true)"
  if [[ -n "$hits" ]]; then
    report+="$file: log call without a request-scoped logger"$'\n'
    report+="$(printf '%s\n' "$hits" | sed 's/^/    /')"$'\n'
    fail=1
  fi
done

if [[ -n "$report" ]]; then
  printf '%s\n' "Request-scoped logging (PILLARS.md §7):" "$report" >&2
fi

exit "$fail"
