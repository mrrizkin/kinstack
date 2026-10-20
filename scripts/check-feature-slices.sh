#!/usr/bin/env bash
set -euo pipefail

root="${1:-internal/features}"
fail=0

if ! command -v rg >/dev/null 2>&1; then
  printf 'check-feature-slices: ripgrep (rg) is required but not installed\n' >&2
  exit 1
fi

check_empty() {
  local label="$1"
  local output="$2"
  if [[ -n "$output" ]]; then
    printf '%s\n' "$output" >&2
    fail=1
  fi
}

check_empty "repository_fake files are forbidden:" \
  "$(find "$root" -name 'repository_fake*.go' -print)"
check_empty "FakeRepository identifiers are forbidden in feature code:" \
  "$(rg -n '\bFakeRepository\b|NewFakeRepository|fakeRepository|fakeRepo' "$root" --glob '*.go' || true)"
check_empty "camelCase JSON tags are forbidden in production feature code:" \
  "$(rg -n 'json:\"[^\"]*[A-Z][^\"]*\"' "$root" --glob '*.go' --glob '!**/*_test.go' || true)"
check_empty "database tags are forbidden in DTO files:" \
  "$(rg -n 'db:\"' "$root" --glob 'dto.go' || true)"

# Every feature slice must ship the canonical files and a handler with an
# unexpected-error logging path.
while IFS= read -r handler; do
  dir="$(dirname "$handler")"
  feature="$(basename "$dir")"
  required=("$feature.go" dto.go repository.go service.go handler.go repository_pg.go repository_mem.go service_test.go)
  for required_file in "${required[@]}"; do
    if [[ ! -f "$dir/$required_file" ]]; then
      printf '%s: missing %s\n' "$dir" "$required_file" >&2
      fail=1
    fi
  done
  if ! rg -q 'Errorw\(' "$handler"; then
    printf '%s: no unexpected-error logger path\n' "$handler" >&2
    fail=1
  fi
done < <(find "$root" -name handler.go -print | sort)

if (( fail != 0 )); then
  exit 1
fi
printf 'feature slice checks passed\n'
