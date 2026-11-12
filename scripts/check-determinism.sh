#!/usr/bin/env bash
# Direct time and randomness in feature code (PILLARS.md §4). Every hit must be
# either removed or grandfathered in scripts/determinism-baseline.tsv.
#
# A hit carrying //kinstack:allow-nondeterminism on the line above, the line
# itself, or the line below is exempt: golines moves comments when it wraps a
# call. The pragma marks the one place that supplies a production default; the
# seam itself takes the clock as an argument, which is what makes a slice
# replayable.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

baseline="scripts/determinism-baseline.tsv"
pattern='time\.Now\(|time\.Since\(|time\.Sleep\(|time\.After\(|time\.Tick|math/rand'
pragma='kinstack:allow-nondeterminism'

[[ -f "$baseline" ]] || { printf 'check-determinism: missing %s\n' "$baseline" >&2; exit 1; }

# Tracked and untracked-but-not-ignored: a new file must fail before it is
# committed, not after.
mapfile -t files < <(
  git ls-files --cached --others --exclude-standard -- \
    'internal/features' 'internal/platform' 'internal/pkg' 'internal/bootstrap' 'config' 'cmd' |
    grep '\.go$' | grep -v '_test\.go$'
)
if [[ ${#files[@]} -eq 0 ]]; then
  printf 'check-determinism: no feature files found\n' >&2
  exit 1
fi

fail=0
report=""
current_keys=""

for file in "${files[@]}"; do
  hits="$(grep -nE "$pattern" "$file" | while IFS=: read -r line _; do
    window="$(sed -n "$((line > 1 ? line - 1 : 1)),$((line + 1))p" "$file")"
    grep -qF "$pragma" <<< "$window" || printf '%s\n' "$(sed -n "${line}p" "$file" | sed "s/^/$line:/")"
  done || true)"
  [[ -z "$hits" ]] && continue
  count="$(printf '%s\n' "$hits" | wc -l | tr -d ' ')"
  allowed="$(awk -F'\t' -v p="$file" '$1 == p { print $2 }' "$baseline")"
  current_keys+="$file"$'\t'"$count"$'\n'
  if [[ -z "$allowed" ]]; then
    report+="$file: $count hit(s), not in the baseline"$'\n'
    report+="$(printf '%s\n' "$hits" | sed 's/^/    /')"$'\n'
    fail=1
  elif (( count > allowed )); then
    report+="$file: $count hit(s), baseline allows $allowed"$'\n'
    report+="$(printf '%s\n' "$hits" | sed 's/^/    /')"$'\n'
    fail=1
  fi
done

lowered=""
while IFS=$'\t' read -r file allowed; do
  [[ -z "${file:-}" || "$file" == \#* ]] && continue
  count="$(awk -F'\t' -v p="$file" '$1 == p { print $2 }' <<< "$current_keys")"
  count="${count:-0}"
  if (( count < allowed )); then
    lowered+="  $file: $allowed -> $count"$'\n'
  fi
done < "$baseline"

if [[ -n "$report" ]]; then
  printf '%s\n' \
    "Direct time or randomness in feature code (PILLARS.md §4):" \
    "$report" \
    "Inject the clock: take now func() time.Time or *dstsim.Clock." >&2
fi

[[ -n "$lowered" ]] && printf '%s\n' \
  "Baseline entries that can be lowered in $baseline:" \
  "$lowered"

exit "$fail"
