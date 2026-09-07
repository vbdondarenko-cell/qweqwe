#!/usr/bin/env bash
set -euo pipefail

ROOT="/opt/linkup"
ARTIFACT_ROOT="$ROOT/artifacts"
CANDIDATE_ROOT="$ARTIFACT_ROOT/candidates"
RELEASE_ROOT="$ARTIFACT_ROOT/releases"
STABLE_LINK="$ARTIFACT_ROOT/stable"
ANDROID_HOME="${ANDROID_HOME:-/opt/android-sdk}"

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
[ -f "$candidate/SHA256SUMS.txt" ] || fail "candidate checksum ledger missing"
[ -f "$candidate/BUILD_METADATA.txt" ] || fail "candidate metadata missing"

grep -qx "commit=$commit" "$candidate/BUILD_METADATA.txt" || fail "candidate metadata commit mismatch"
grep -qx 'state=candidate' "$candidate/BUILD_METADATA.txt" || fail "candidate is not in candidate state"

cd "$candidate"
sha256sum -c SHA256SUMS.txt

apksigner_bin=""
if command -v apksigner >/dev/null 2>&1; then
  apksigner_bin="$(command -v apksigner)"
elif [ -d "$ANDROID_HOME/build-tools" ]; then
  apksigner_bin="$(find "$ANDROID_HOME/build-tools" -mindepth 2 -maxdepth 2 -type f -name apksigner -print | sort -V | tail -1)"
fi
[ -n "$apksigner_bin" ] || fail "apksigner not found"

shopt -s nullglob
apks=(LinkUp-v1.0-*-$commit-debug.apk LinkUp-v1.0-$commit-release.apk)
# The first glob is kept for compatibility with older candidate names; avoid
# duplicate verification if both patterns resolve to the same file.
declare -A seen_apk=()
for apk in "${apks[@]}"; do
  [ -f "$apk" ] || continue
  [ -z "${seen_apk[$apk]:-}" ] || continue
  "$apksigner_bin" verify --verbose --print-certs "$apk" >/dev/null
  seen_apk[$apk]=1
done
for aab in LinkUp-v1.0-$commit-release.aab; do
  [ -f "$aab" ] || continue
  jarsigner -verify -strict "$aab" >/dev/null
 done
shopt -u nullglob

mkdir -p "$RELEASE_ROOT"
release="$RELEASE_ROOT/$commit"
rm -rf -- "$release"

cat > "$candidate/BUILD_METADATA.txt" <<EOF
commit=$commit
promoted_at_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)
state=stable
EOF

mv -- "$candidate" "$release"

if [ -e "$STABLE_LINK" ] && [ ! -L "$STABLE_LINK" ]; then
  fail "$STABLE_LINK exists and is not a symlink"
fi
next_link="$ARTIFACT_ROOT/.stable.next.$$"
rm -f -- "$next_link"
ln -s "releases/$commit" "$next_link"
mv -Tf -- "$next_link" "$STABLE_LINK"

resolved="$(readlink "$STABLE_LINK")"
[ "$resolved" = "releases/$commit" ] || fail "stable link verification failed"

# Promotion is complete. Keep only the one stable release and no stale
# candidates, minimizing disk usage without risking the previous stable during
# candidate build/verification.
find "$RELEASE_ROOT" -mindepth 1 -maxdepth 1 -type d ! -name "$commit" -exec rm -rf -- {} +
find "$CANDIDATE_ROOT" -mindepth 1 -maxdepth 1 -type d -exec rm -rf -- {} +

printf 'stable_commit=%s\nstable=%s\n' "$commit" "$STABLE_LINK"
