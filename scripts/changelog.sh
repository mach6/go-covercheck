#!/usr/bin/env bash
# Maintains CHANGELOG.md (keep a changelog format). The notes are written by hand under "## [Unreleased]".
#
#   changelog.sh notes
#       print the content of the [Unreleased] section, the release notes of the next release.
#       fails when the section is empty.
#   changelog.sh release <tag> [<date>]
#       roll the [Unreleased] section into a section for <tag> dated <date> (default: today, UTC), leave a new
#       empty [Unreleased] section, and update the compare links at the bottom.
#
# CHANGELOG sets the file to work on (default: CHANGELOG.md at the root of the repository).

set -euo pipefail

CHANGELOG="${CHANGELOG:-$(dirname "$0")/../CHANGELOG.md}"

usage() {
	sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//' >&2
	exit 2
}

# the [Unreleased] section without its heading, leading and trailing blank lines removed
notes() {
	local out
	[[ -f "$CHANGELOG" ]] || { echo "$CHANGELOG not found" >&2; return 1; }
	grep -q '^## \[Unreleased\]' "$CHANGELOG" || { echo "$CHANGELOG has no [Unreleased] section" >&2; return 1; }
	out=$(awk '
		/^## \[Unreleased\]/ { found = 1; next }
		found && /^## \[/    { exit }
		found && /^\[[^]]+\]: / { exit }
		found {
			if ($0 ~ /^[[:space:]]*$/) { blanks = blanks "\n"; next }
			if (started) { printf "%s", blanks }
			blanks = ""; started = 1; print
		}
	' "$CHANGELOG")
	if [[ -z "$out" ]]; then
		echo "the [Unreleased] section of $CHANGELOG is empty" >&2
		return 1
	fi
	printf '%s\n' "$out"
}

release() {
	local tag="${1:-}" date="${2:-$(date -u +%F)}"
	[[ -n "$tag" ]] || usage

	notes > /dev/null
	if grep -q "^## \[$tag\]" "$CHANGELOG"; then
		echo "$CHANGELOG already has a section for $tag" >&2
		return 1
	fi

	# the version that was released before, and the base url of the compare links
	local previous base
	previous=$(awk '/^## \[Unreleased\]/ { found = 1; next }
		found && /^## \[/ { sub(/^## \[/, ""); sub(/\].*/, ""); print; exit }' "$CHANGELOG")
	base=$(sed -n 's#^\[Unreleased\]: \(.*\)/compare/.*#\1#p' "$CHANGELOG")
	if [[ -z "$previous" || -z "$base" ]]; then
		echo "$CHANGELOG has no previous version or no [Unreleased] compare link" >&2
		return 1
	fi

	local tmp
	tmp=$(mktemp)
	awk -v v="$tag" -v d="$date" -v prev="$previous" -v base="$base" '
		/^## \[Unreleased\]/ && !heading {
			print; print ""; print "## [" v "] - " d
			heading = 1; next
		}
		/^\[Unreleased\]: / {
			print "[Unreleased]: " base "/compare/" v "...HEAD"
			print "[" v "]: " base "/compare/" prev "..." v
			next
		}
		{ print }
	' "$CHANGELOG" > "$tmp"
	cat "$tmp" > "$CHANGELOG"
	rm -f "$tmp"
}

case "${1:-}" in
	notes)   notes ;;
	release) shift; release "$@" ;;
	*)       usage ;;
esac
