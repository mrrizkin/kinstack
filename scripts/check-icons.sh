#!/usr/bin/env bash
# Icon names live in resources/js/lib/icons.ts so one edit can migrate the set.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

pattern='i-(bi|material-symbols|lucide|heroicons|mdi|ph|tabler|simple-icons)[:-]'

matches="$(
  { rg -n --no-heading "$pattern" resources/js vite.config.ts \
      --glob '*.vue' --glob '*.ts' --glob '!*.test.ts' \
      --glob '!resources/js/lib/icons.ts' || true; }
)"

if [[ -n "$matches" ]]; then
  printf '%s\n' "Icon names are hardcoded outside the icon registry:" "$matches" "" \
    "Import { icons } from \"@/lib/icons\" and use a semantic key instead." >&2
  exit 1
fi
