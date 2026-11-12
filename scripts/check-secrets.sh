#!/usr/bin/env bash
# Secret scanning (PILLARS.md §5).
#
# The scan runs on the committed tree, not on the working directory: .env holds
# real local credentials, is ignored on purpose, and is not what this gate
# protects. A secret that reaches a commit reaches the remote, and that is the
# mistake this catches.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

if ! command -v gitleaks >/dev/null 2>&1; then
  printf '%s\n' "check-secrets: gitleaks not found." \
    "  brew install gitleaks  # or" \
    "  curl -sSL https://github.com/gitleaks/gitleaks/releases/download/v8.30.1/gitleaks_8.30.1_linux_x64.tar.gz | tar -xz -C ~/.local/bin gitleaks" >&2
  exit 1
fi

scan="$(mktemp -d)"
trap 'rm -rf "$scan"' EXIT

git archive HEAD | tar -x -C "$scan"

gitleaks dir "$scan" \
  --config "$root/.gitleaks.toml" \
  --no-banner \
  --redact \
  --max-target-megabytes 2 \
  --exit-code 1
