#!/usr/bin/env sh
# Fails when any Go file differs from `just format-go` output, so the verify
# gate and the formatter cannot drift apart. Fix with `just format-go`.
set -eu

failed=0

check() {
  name="$1"
  shift
  output="$("$@")" || exit 1
  if [ -n "$output" ]; then
    printf '%s\n' "$output"
    printf '%s: run "just format-go"\n' "$name" >&2
    failed=1
  fi
}

check goimports go tool goimports -l .
check gofumpt go tool gofumpt -l .
check golines go tool golines -l --ignore-generated .

exit "$failed"
