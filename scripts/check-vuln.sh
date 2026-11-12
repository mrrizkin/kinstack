#!/usr/bin/env bash
# Reachable-vulnerability gate. govulncheck has no exclude flag, so the
# allow-list lives here: every entry must have a recorded deviation in
# PILLARS.md, and an advisory that is not listed fails the gate.
set -uo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

allowed=(
  # No patched crewjam/saml release exists; the panic is recovered in
  # parseResponseSafely. See PILLARS.md # Deviations.
  GO-2026-6909
)

output="$(go run golang.org/x/vuln/cmd/govulncheck@latest ./... 2>&1)"
status=$?

if (( status == 0 )); then
  printf 'check-vuln: no reachable vulnerabilities\n'
  exit 0
fi

if ! printf '%s\n' "$output" | grep -q 'Your code is affected'; then
  # Not a finding: the scan itself failed (database, network, build).
  printf '%s\n' "$output"
  exit 1
fi

unexpected=0
while IFS= read -r id; do
  [[ -z "$id" ]] && continue
  if ! printf '%s\n' "${allowed[@]}" | grep -qx "$id"; then
    printf 'check-vuln: %s is reachable with no recorded deviation\n' "$id"
    unexpected=1
  fi
done < <(printf '%s\n' "$output" |
  grep -oE 'Vulnerability #[0-9]+: GO-[0-9]{4}-[0-9]+' |
  sed 's/.*: //' | sort -u)

if (( unexpected )); then
  printf '%s\n' "$output"
  exit 1
fi
printf 'check-vuln: only recorded deviations are reachable\n'
