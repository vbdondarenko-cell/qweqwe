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

if [ -e "$STABLE_LINK" ] && [ ! -L "$STABLE_LINK" ]; then
  fail "$STABLE_LINK exists and is not a symlink"
fi
if [ -L "$STABLE_LINK" ] && [ "$(readlink "$STABLE_LINK")" = "releases/$commit" ]; then
  fail "commit $commit is already stable"
fi

cd "$candidate"
sha256sum -c SHA256SUMS.txt

apksigner_bin=""
if command -v apksigner >/dev/null 2>&1; then
  apksigner_bin="$(command -v apksigner)"
elif [ -d "$ANDROID_HOME/build-tools" ]; then
  apksigner_bin="$(find "$ANDROID_HOME/build-tools" -mindepth 2 -maxdepth 2 -type f -name apksigner -print | sort -V | tail -1)"
fi
[ -n "$apksigner_bin" ] || fail "apksigner not found"

debug_apk="LinkUp-v1.0-$commit-debug.apk"
[ -f "$debug_apk" ] || fail "debug APK missing"
"$apksigner_bin" verify --verbose --print-certs "$debug_apk" >/dev/null

release_apk="LinkUp-v1.0-$commit-release.apk"
release_aab="LinkUp-v1.0-$commit-release.aab"
if [ -f "$release_apk" ] || [ -f "$release_aab" ]; then
  [ -f "$release_apk" ] && [ -f "$release_aab" ] || fail "release APK/AAB pair is incomplete"
  "$apksigner_bin" verify --verbose --print-certs "$release_apk" >/dev/null
  jarsigner -verify -strict "$release_aab" >/dev/null
fi

mkdir -p "$RELEASE_ROOT"
release="$RELEASE_ROOT/$commit"
[ ! -e "$release" ] || fail "release directory already exists: $release"

# Moving within /opt/linkup keeps the verified candidate intact until this
# point. The previous stable symlink is not changed until the new release is
# fully staged.
mv -- "$candidate" "$release"
promoted=0
rollback_stage() {
  if [ "$promoted" -eq 0 ] && [ -d "$release" ] && [ ! -e "$candidate" ]; then
    mv -- "$release" "$candidate" || true
  fi
}
trap rollback_stage ERR INT TERM

cat > "$release/BUILD_METADATA.txt" <<EOF
commit=$commit
promoted_at_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)
state=stable
EOF

next_link="$ARTIFACT_ROOT/.stable.next.$$"
rm -f -- "$next_link"
ln -s "releases/$commit" "$next_link"
mv -Tf -- "$next_link" "$STABLE_LINK"

resolved="$(readlink "$STABLE_LINK")"
[ "$resolved" = "releases/$commit" ] || fail "stable link verification failed"
promoted=1
trap - ERR INT TERM

# Only after the atomic stable-link switch is verified may the previous stable
# artifact be deleted. This keeps disk use bounded to stable + candidate while
# preserving rollback safety during the build/promotion phase.
find "$RELEASE_ROOT" -mindepth 1 -maxdepth 1 -type d ! -name "$commit" -exec rm -rf -- {} +
find "$CANDIDATE_ROOT" -mindepth 1 -maxdepth 1 -type d -exec rm -rf -- {} +

printf 'stable_commit=%s\nstable=%s\n' "$commit" "$STABLE_LINK"
