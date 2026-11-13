#!/usr/bin/env bash
# Cognitive complexity and nesting are review signals, not line budgets.
# New offenders fail; grandfathered functions may only shrink.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

threshold=15
nesting_ceiling=8
baseline="scripts/complexity-baseline.tsv"

# Generated templ and sqlc output is machine-written and excluded; tests measure
# fixtures, not the code under review.
mapfile -t files < <(git ls-files -- '*.go' | grep -v '_templ.go$' | grep -v '/sqlc/' | grep -v '_test.go$')
if [[ ${#files[@]} -eq 0 ]]; then
  printf 'check-complexity: no Go files found\n' >&2
  exit 1
fi

if [[ ! -f "$baseline" ]]; then
  printf 'check-complexity: missing %s\n' "$baseline" >&2
  exit 1
fi

current="$(go tool gocognit -over "$threshold" \
  -f '{{.Complexity}}	{{.PkgName}}	{{.FuncName}}	{{.Pos}}' \
  "${files[@]}" 2>/dev/null | grep -v '^gocognit:' || true)"

fail=0
report=""

# Pass 1: is this function already known, and did it grow?
while IFS=$'\t' read -r score pkg fn pos; do
  [[ -z "${score:-}" ]] && continue
  allowed="$(awk -F'\t' -v p="$pkg" -v f="$fn" '$1 == p && $2 == f { print $3 }' "$baseline")"
  if [[ -z "$allowed" ]]; then
    report+="$(printf 'new   %3s      %s %s  %s' "$score" "$pkg" "$fn" "$pos")"$'\n'
    fail=1
  elif (( score > allowed )); then
    report+="$(printf 'grew  %3s > %-3s  %s %s  %s' "$score" "$allowed" "$pkg" "$fn" "$pos")"$'\n'
    fail=1
  fi
done <<< "$current"

# Pass 2: the baseline only ratchets down. Report entries that are now clean.
current_keys="$(printf '%s\n' "$current" | awk -F'\t' 'NF >= 3 { print $2 "\t" $3 }')"
stale=""
while IFS=$'\t' read -r pkg fn allowed; do
  [[ -z "${pkg:-}" || "$pkg" == \#* ]] && continue
  if ! grep -qxF "$pkg"$'\t'"$fn" <<< "$current_keys"; then
    stale+="  $pkg $fn  (was $allowed, now under $threshold)"$'\n'
  fi
done < "$baseline"

if [[ -n "$report" ]]; then
  printf '%s\n' \
    "Cognitive complexity over $threshold (see Pillars §8, Function Shape):" \
    "$report" \
    "Split by level of abstraction, not by line count. Reduce nesting instead of" \
    "extracting a helper that takes the caller's state as parameters." >&2
fi

if [[ -n "$stale" ]]; then
  printf '%s\n' "Baseline entries that can be removed from $baseline:" "$stale"
fi

# Deep nesting cannot be fixed by extraction, so it gets its own guard.
nesting="$(go tool nestif --min "$nesting_ceiling" "${files[@]}" 2>&1 || true)"
if [[ -n "$nesting" ]]; then
  printf '%s\n' "Nesting weight over $nesting_ceiling:" "$nesting" >&2
  fail=1
fi

exit "$fail"
