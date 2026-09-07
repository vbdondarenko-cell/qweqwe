#!/usr/bin/env bash
set -euo pipefail

ROOT="/opt/linkup"
CANDIDATE_ROOT="$ROOT/artifacts/candidates"

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

commit="${1:-}"
[ -n "$commit" ] || fail "usage: $0 <candidate-commit>"
case "$commit" in
  *[!0-9a-f]*|'') fail "invalid commit SHA" ;;
esac
[ "${#commit}" -ge 7 ] || fail "commit SHA is too short"

candidate="$CANDIDATE_ROOT/$commit"
[ -d "$candidate" ] || fail "candidate not found: $candidate"

rm -rf -- "$candidate"
printf 'discarded_candidate=%s\n' "$commit"
