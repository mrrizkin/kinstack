#!/usr/bin/env bash
# Shape report — informational only, never fails the build.
#
# Prints files and functions that are drifting away from the structural rules in
# the refactor plan. Treat the output as a hint, not a gate.
set -uo pipefail

vue_limit="${VUE_LINE_LIMIT:-300}"
go_limit="${GO_FUNC_LINE_LIMIT:-70}"

fail=0
warn() {
  printf '• %s\n' "$1"
  fail=1
}

section() {
  printf '\n\033[1m== %s ==\033[0m\n' "$1"
}

section "Vue files over ${vue_limit} lines"
if command -v find >/dev/null 2>&1; then
  while IFS= read -r entry; do
    [ -n "$entry" ] || continue
    warn "$entry"
  done < <(find resources/js -name '*.vue' -type f -exec wc -l {} + 2>/dev/null | awk -v limit="$vue_limit" '$1 > limit && $2 != "total" { print $1 "\t" $2 }' | sort -rn)
fi

section "Go functions over ${go_limit} lines"
while IFS= read -r entry; do
  [ -n "$entry" ] || continue
  warn "$entry"
done < <(find . -name '*.go' -type f -not -path './vendor/*' -print0 2>/dev/null |
  xargs -0 awk -v limit="$go_limit" '
    function flush() {
      if (name != "" && end != 0) {
        lines = end - start + 1
        if (lines > limit) {
          printf "%d\t%s:%d\t%s\n", lines, file, start, name
        }
      }
      name = ""
      start = 0
      end = 0
      depth = 0
      pending = 0
    }
    FNR == 1 { flush(); file = FILENAME }
    name == "" && /^func / {
      name = $0
      start = FNR
      depth = 0
      pending = 1
    }
    name != "" {
      line = $0
      opens = gsub(/\{/, "{", line)
      closes = gsub(/\}/, "}", line)
      if (pending) {
        pending = 0
        depth = opens - closes
        if (depth > 0) end = FNR
        if (depth == 0 && opens > 0) flush()
        next
      }
      depth += opens - closes
      if (opens > 0) end = FNR
      if (depth <= 0 && end > 0) flush()
    }
    END { flush() }
  ' | sort -rn)

section "reactive() usage (prefer ref/computed)"
while IFS= read -r entry; do
  [ -n "$entry" ] || continue
  warn "$entry"
done < <(rg -n 'reactive\(' resources/js --glob '*.ts' --glob '*.vue' || true)

section "TableColumn<any>"
while IFS= read -r entry; do
  [ -n "$entry" ] || continue
  warn "$entry"
done < <(rg -n 'TableColumn<any>' resources/js || true)

section "Bare 'as' casts"
while IFS= read -r entry; do
  [ -n "$entry" ] || continue
  warn "$entry"
done < <(rg -n --pcre2 '\bas\s+(?:[A-Z]|\{|\[|\(|unknown\b|Record\b|Array\b|Readonly\b|Partial\b)' resources/js/pages resources/js/lib resources/js/layouts --glob '*.ts' --glob '*.vue' | rg -v '\bas const\b' || true)

if [[ "$fail" -eq 0 ]]; then
  printf '\nNo shape warnings.\n'
fi

exit 0
