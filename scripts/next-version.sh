#!/bin/sh
# Release tags follow the UTC ISO week: v<YEAR>.<WEEK>.<PATCH>, where PATCH
# counts releases inside that week. Run `git fetch --tags` first on a clone that
# has not seen the release tags, otherwise the counter restarts at 0.
set -eu

prefix="v$(date -u +'%Y.%V')"

last=$(git tag -l "${prefix}.*" |
	sed -E "s/^v[0-9]{4}\.[0-9]{2}\.([0-9]+)$/\1/" |
	sort -n | tail -1)

if [ -n "$last" ]; then
	patch=$((last + 1))
else
	patch=0
fi

version="${prefix}.${patch}"

case "${1:-}" in
"")
	echo "$version"
	;;
--create)
	git tag -a "$version" -m "$version"
	echo "$version"
	echo "push it with: git push origin $version" >&2
	;;
*)
	echo "usage: $0 [--create]" >&2
	exit 2
	;;
esac
