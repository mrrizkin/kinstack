#!/usr/bin/env bash
# Generate local secrets for .env so the stack never runs on the values that
# ship in .env.example (those are rejected in production).
set -euo pipefail

env_file="${1:-.env}"
example_file="${2:-.env.example}"

if ! command -v openssl >/dev/null 2>&1; then
	echo "setup-secrets: openssl is required" >&2
	exit 1
fi

if [[ ! -f "$env_file" ]]; then
	if [[ ! -f "$example_file" ]]; then
		echo "setup-secrets: $example_file not found" >&2
		exit 1
	fi
	cp "$example_file" "$env_file"
	echo "setup-secrets: created $env_file from $example_file"
fi

# URL-safe, 43 characters, no quoting needed in .env or compose files.
generate() {
	openssl rand -base64 48 | tr -d '\n=' | tr '/+' '_-'
}

current() {
	local key="$1"
	sed -n "s/^${key}=//p" "$env_file" | head -n 1
}

set_value() {
	local key="$1" value="$2"
	local escaped="${value//\\/\\\\}"
	escaped="${escaped//&/\\&}"
	escaped="${escaped//#/\\#}"
	if grep -q "^${key}=" "$env_file"; then
		sed -i "s#^${key}=.*#${key}=${escaped}#" "$env_file"
	else
		printf '%s=%s\n' "$key" "$value" >>"$env_file"
	fi
}

# Only replace values that are empty or still match an example default so
# running this twice never rotates working credentials.
should_replace() {
	case "${1}" in
	"" | kinstack | kinstack-secret | kinstack-local-password | superadmin | rustfsadmin | \
		local-* | replace-* | change-me* | your-* | *minimal-32*)
		return 0
		;;
	esac
	return 1
}

rotate() {
	local key="$1" value
	value="$(current "$key")"
	if should_replace "$value"; then
		set_value "$key" "$(generate)"
		echo "setup-secrets: rotated $key"
	fi
}

old_password="$(current POSTGRES_PASSWORD)"
rotate POSTGRES_PASSWORD
new_password="$(current POSTGRES_PASSWORD)"
if [[ -n "$new_password" && "$old_password" != "$new_password" ]]; then
	sed -i "s#^DATABASE_URL=\(postgres\(ql\)\?://[^:/@]*\):[^@]*@#DATABASE_URL=\1:${new_password}@#" "$env_file"
	echo "setup-secrets: synced the database password into DATABASE_URL"
fi

rotate VALKEY_PASSWORD
rotate SECRET_KEY
rotate RUSTFS_ACCESS_KEY
rotate RUSTFS_SECRET_KEY

echo "setup-secrets: done. Review $env_file before committing anything."
