#!/usr/bin/env bash
# Release the notes under "## [Unreleased]" in CHANGELOG.md as a section for a tag.
#
#   scripts/changelog.sh <tag> [date]
#
# The text under [Unreleased] becomes "## [<tag>] - <date>", [Unreleased] is left empty, and the
# [Unreleased] and [<tag>] compare links at the bottom of the file are updated. When [Unreleased]
# is empty, the notes are generated from the commits with git-chglog instead.
#
# date defaults to the date of the tag, or today. CHANGELOG sets the file to change (CHANGELOG.md).
set -euo pipefail

tag="${1:?usage: changelog.sh <tag> [date]}"
file="${CHANGELOG:-CHANGELOG.md}"
date="${2:-$(git for-each-ref --format='%(creatordate:short)' "refs/tags/${tag}" 2>/dev/null | head -n1)}"
date="${date:-$(date -u +%F)}"

fail() {
	echo "changelog: $*" >&2
	exit 1
}

[ -f "${file}" ] || fail "${file} not found"
grep -q '^## \[Unreleased\]' "${file}" || fail "${file} has no [Unreleased] section"

if grep -q "^## \[${tag}\]" "${file}"; then
	echo "changelog: ${file} already has a section for ${tag}; nothing to do"
	exit 0
fi

# trim_blank drops the blank lines at the start and the end of its input.
trim_blank() {
	awk 'NF { found = 1 } found { lines[++n] = $0 }
		END { while (n > 0 && lines[n] ~ /^[[:space:]]*$/) n--; for (i = 1; i <= n; i++) print lines[i] }'
}

# The curated notes: everything under [Unreleased], up to the next section or the links at the end.
body="$(awk '
	/^## \[Unreleased\]/ { in_block = 1; next }
	in_block && /^## \[/ { exit }
	in_block && /^\[[^]]+\]: / { exit }
	in_block { print }
' "${file}" | trim_blank)"

# Nothing curated: generate the notes for the tag from the commits.
if [ -z "${body}" ]; then
	command -v git-chglog >/dev/null 2>&1 || fail "[Unreleased] is empty and git-chglog is not installed"
	body="$(git-chglog "${tag}" | awk -v tag="${tag}" '
		$0 ~ "^## \\[?" tag "\\]? - " { in_block = 1; next }
		in_block && /^## / { exit }
		in_block && /^\[[^]]+\]: / { exit }
		in_block { print }
	' | trim_blank)"
	[ -n "${body}" ] || fail "[Unreleased] is empty and git-chglog found no changes for ${tag}"
fi

# The compare links: [Unreleased]: <base>/compare/<previous>...HEAD
base=""
prev=""
link="$(grep -m1 '^\[Unreleased\]: ' "${file}" || true)"
if [ -n "${link}" ]; then
	url="${link#\[Unreleased\]: }"
	base="${url%%/compare/*}"
	prev="${url#*/compare/}"
	prev="${prev%...HEAD}"
fi

tmp="$(mktemp)"
trap 'rm -f "${tmp}" "${tmp}.body"' EXIT
printf '%s\n' "${body}" >"${tmp}.body"

awk -v tag="${tag}" -v date="${date}" -v bodyfile="${tmp}.body" -v base="${base}" -v prev="${prev}" '
	function emit_body(   line) {
		while ((getline line < bodyfile) > 0) print line
		close(bodyfile)
	}
	/^## \[Unreleased\]/ && !released {
		print
		print ""
		print "## [" tag "] - " date
		print ""
		emit_body()
		print ""
		skipping = 1
		released = 1
		next
	}
	skipping && /^## \[/ { skipping = 0 }
	skipping && /^\[[^]]+\]: / { skipping = 0 }
	skipping { next }
	/^\[Unreleased\]: / && base != "" {
		print "[Unreleased]: " base "/compare/" tag "...HEAD"
		print "[" tag "]: " base "/compare/" prev "..." tag
		next
	}
	{ print }
' "${file}" >"${tmp}"

cat "${tmp}" >"${file}"
echo "changelog: released the notes in ${file} as ${tag} (${date})"
