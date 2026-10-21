#!/usr/bin/env sh
# Fails when any Go file differs from `just format-go` output, so the verify
# gate and the formatter cannot drift apart. Fix with `just format-go`.
#
# templ's generated *_templ.go files are skipped because the generator owns
# their formatting; reformatting them only creates churn when templ runs.
set -eu

failed=0

report() {
  name="$1"
  output="$2"
  if [ -n "$output" ]; then
    printf '%s\n' "$output"
    printf '%s: run "just format-go"\n' "$name" >&2
    failed=1
  fi
}

go_files() {
  git ls-files -z --cached --others --exclude-standard -- '*.go' ':!*_templ.go'
}

report goimports "$(go_files | xargs -0 -r go tool goimports -l)"
report gofumpt "$(go_files | xargs -0 -r go tool gofumpt -l)"
report golines "$(go tool golines -l --ignore-generated .)"

exit "$failed"
