#!/usr/bin/env bash
# Generated SQL is committed so `go build` works without running a generator.
# This gate regenerates and fails when the committed output is stale, so the
# queries reviewed in a pull request are the queries that ship.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

generated=(
  db/schema.sql
  'internal/features/*/sqlc'
  'internal/platform/*/sqlc'
)

go run ./cmd/schema
go tool sqlc generate

# Compared against HEAD, so this gate answers "does the committed generator
# output match the committed queries". Run it after committing, as CI does.
if ! git diff --quiet HEAD -- "${generated[@]}"; then
  git diff --stat HEAD -- "${generated[@]}" >&2
  printf 'check-sqlc: generated SQL is stale; run "just sqlc-generate" and commit the result\n' >&2
  exit 1
fi

untracked="$(git ls-files --others --exclude-standard -- "${generated[@]}")"
if [[ -n "$untracked" ]]; then
  printf '%s\n' "$untracked" >&2
  printf 'check-sqlc: generated SQL is untracked; commit it\n' >&2
  exit 1
fi

printf 'sqlc output is current\n'
