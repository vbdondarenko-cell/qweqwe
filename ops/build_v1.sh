#!/usr/bin/env bash
set -euo pipefail

ROOT="/opt/linkup"
SRC="$ROOT/src"
ARTIFACT_ROOT="$ROOT/artifacts"
CANDIDATE_ROOT="$ARTIFACT_ROOT/candidates"
ANDROID_HOME="${ANDROID_HOME:-/opt/android-sdk}"
BUILD_ENV_FILE="${LINKUP_BUILD_ENV_FILE:-/etc/linkup/build.env}"
export ANDROID_HOME
export ANDROID_SDK_ROOT="$ANDROID_HOME"
export PATH="/usr/local/go/bin:$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$PATH"

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

# Build/release configuration is server-only. Nothing from this file is echoed,
# copied into Git, or written to artifact metadata. Firebase Android client
# identifiers are exposed to Gradle by ORG_GRADLE_PROJECT_* variables; signing
# passwords and keystore paths remain only in the build process environment.
if [ -e "$BUILD_ENV_FILE" ]; then
  [ -r "$BUILD_ENV_FILE" ] || fail "build environment file is not readable: $BUILD_ENV_FILE"
  set -a
  # shellcheck disable=SC1090
  . "$BUILD_ENV_FILE"
  set +a
fi

cd "$SRC"
git fetch --prune origin main
git checkout main
git reset --hard origin/main
commit="$(git rev-parse HEAD)"

mkdir -p "$CANDIDATE_ROOT"
# Only one unpromoted candidate is retained. Stable releases live elsewhere and
# are never touched by a new build or a failed build.
find "$CANDIDATE_ROOT" -mindepth 1 -maxdepth 1 -type d ! -name "$commit" -exec rm -rf -- {} +
rm -rf -- "$CANDIDATE_ROOT/$commit"
tmp_candidate="$CANDIDATE_ROOT/.${commit}.tmp.$$"
mkdir -p "$tmp_candidate"

cleanup_failed_candidate() {
  rm -rf -- "$tmp_candidate"
}
trap cleanup_failed_candidate ERR INT TERM

cd "$SRC/backend"
go mod verify
go test ./... -count=1
go vet ./...
go test -race ./... -count=1
go build -trimpath -o "$tmp_candidate/linkup-api" ./cmd/api

cd "$SRC/android"
chmod +x ./gradlew
./gradlew --no-daemon --stacktrace :app:testDebugUnitTest :app:lintDebug :app:assembleDebug

debug_apk="LinkUp-v1.0-$commit-debug.apk"
cp app/build/outputs/apk/debug/app-debug.apk "$tmp_candidate/$debug_apk"

if [ "${LINKUP_BUILD_RELEASE:-0}" = "1" ]; then
  : "${ORG_GRADLE_PROJECT_LINKUP_API_BASE_URL:?missing release API URL}"
  : "${ORG_GRADLE_PROJECT_LINKUP_PRIVACY_URL:?missing privacy URL}"
  : "${ORG_GRADLE_PROJECT_LINKUP_TERMS_URL:?missing terms URL}"
  : "${ORG_GRADLE_PROJECT_LINKUP_RESET_HOST:?missing reset host}"
  : "${ORG_GRADLE_PROJECT_LINKUP_FIREBASE_API_KEY:?missing Firebase Android API key}"
  : "${ORG_GRADLE_PROJECT_LINKUP_FIREBASE_APP_ID:?missing Firebase Android app id}"
  : "${ORG_GRADLE_PROJECT_LINKUP_FIREBASE_PROJECT_ID:?missing Firebase project id}"
  : "${ORG_GRADLE_PROJECT_LINKUP_FIREBASE_SENDER_ID:?missing Firebase sender id}"
  : "${ORG_GRADLE_PROJECT_LINKUP_KEYSTORE_FILE:?missing keystore file}"
  : "${ORG_GRADLE_PROJECT_LINKUP_KEYSTORE_PASSWORD:?missing keystore password}"
  : "${ORG_GRADLE_PROJECT_LINKUP_KEY_ALIAS:?missing key alias}"
  : "${ORG_GRADLE_PROJECT_LINKUP_KEY_PASSWORD:?missing key password}"

  ./gradlew --no-daemon --stacktrace \
    :app:testReleaseUnitTest :app:lintRelease :app:assembleRelease :app:bundleRelease

  cp app/build/outputs/apk/release/app-release.apk \
    "$tmp_candidate/LinkUp-v1.0-$commit-release.apk"
  cp app/build/outputs/bundle/release/app-release.aab \
    "$tmp_candidate/LinkUp-v1.0-$commit-release.aab"
fi

cd "$tmp_candidate"
{
  sha256sum linkup-api "$debug_apk"
  if [ "${LINKUP_BUILD_RELEASE:-0}" = "1" ]; then
    sha256sum "LinkUp-v1.0-$commit-release.apk" "LinkUp-v1.0-$commit-release.aab"
  fi
} > SHA256SUMS.txt
sha256sum -c SHA256SUMS.txt

apksigner_bin=""
if command -v apksigner >/dev/null 2>&1; then
  apksigner_bin="$(command -v apksigner)"
elif [ -d "$ANDROID_HOME/build-tools" ]; then
  apksigner_bin="$(find "$ANDROID_HOME/build-tools" -mindepth 2 -maxdepth 2 -type f -name apksigner -print | sort -V | tail -1)"
fi
[ -n "$apksigner_bin" ] || fail "apksigner not found"

{
  echo "debug_apk=$debug_apk"
  "$apksigner_bin" verify --verbose --print-certs "$debug_apk"
  if [ "${LINKUP_BUILD_RELEASE:-0}" = "1" ]; then
    release_apk="LinkUp-v1.0-$commit-release.apk"
    release_aab="LinkUp-v1.0-$commit-release.aab"
    echo "release_apk=$release_apk"
    "$apksigner_bin" verify --verbose --print-certs "$release_apk"
    echo "release_aab=$release_aab"
    jarsigner -verify -strict "$release_aab"
  fi
} > SIGNATURE_VERIFICATION.txt 2>&1

cat > BUILD_METADATA.txt <<EOF
commit=$commit
built_at_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)
release=${LINKUP_BUILD_RELEASE:-0}
state=candidate
EOF

cd "$CANDIDATE_ROOT"
mv -- "$tmp_candidate" "$commit"
trap - ERR INT TERM

printf 'candidate_commit=%s\ncandidate=%s\nnext_step=ops/promote_v1.sh %s\n' \
  "$commit" "$CANDIDATE_ROOT/$commit" "$commit"
