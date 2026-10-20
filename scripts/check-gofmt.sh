#!/usr/bin/env sh
# Fails when any Go file is not gofmt formatted. Fix with `just format-go`.
set -eu

unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
  printf '%s\n' "$unformatted"
  echo "gofmt: run 'just format-go'" >&2
  exit 1
fi
