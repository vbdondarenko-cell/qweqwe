#!/usr/bin/env bash
set -euo pipefail

ROOT="/opt/linkup"
SRC="$ROOT/src"
ANDROID_HOME="${ANDROID_HOME:-/opt/android-sdk}"
export ANDROID_HOME
export ANDROID_SDK_ROOT="$ANDROID_HOME"
export PATH="/usr/local/go/bin:$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$PATH"

cd "$SRC"
git fetch --prune origin main
git checkout main
git reset --hard origin/main
commit="$(git rev-parse HEAD)"

cd "$SRC/backend"
go mod verify
go test ./... -count=1
go vet ./...
go test -race ./... -count=1
go build -trimpath -o "$ROOT/bin/linkup-api" ./cmd/api

cd "$SRC/android"
chmod +x ./gradlew
./gradlew --no-daemon --stacktrace :app:testDebugUnitTest :app:lintDebug :app:assembleDebug

mkdir -p "$ROOT/artifacts/$commit"
cp app/build/outputs/apk/debug/app-debug.apk "$ROOT/artifacts/$commit/LinkUp-v1.0-$commit-debug.apk"
sha256sum "$ROOT/bin/linkup-api" "$ROOT/artifacts/$commit/LinkUp-v1.0-$commit-debug.apk" > "$ROOT/artifacts/$commit/SHA256SUMS.txt"

if [ "${LINKUP_BUILD_RELEASE:-0}" = "1" ]; then
  : "${ORG_GRADLE_PROJECT_LINKUP_API_BASE_URL:?missing release API URL}"
  : "${ORG_GRADLE_PROJECT_LINKUP_PRIVACY_URL:?missing privacy URL}"
  : "${ORG_GRADLE_PROJECT_LINKUP_TERMS_URL:?missing terms URL}"
  : "${ORG_GRADLE_PROJECT_LINKUP_RESET_HOST:?missing reset host}"
  : "${ORG_GRADLE_PROJECT_LINKUP_KEYSTORE_FILE:?missing keystore file}"
  : "${ORG_GRADLE_PROJECT_LINKUP_KEYSTORE_PASSWORD:?missing keystore password}"
  : "${ORG_GRADLE_PROJECT_LINKUP_KEY_ALIAS:?missing key alias}"
  : "${ORG_GRADLE_PROJECT_LINKUP_KEY_PASSWORD:?missing key password}"
  ./gradlew --no-daemon --stacktrace :app:testReleaseUnitTest :app:lintRelease :app:assembleRelease :app:bundleRelease
  cp app/build/outputs/apk/release/app-release.apk "$ROOT/artifacts/$commit/LinkUp-v1.0-$commit-release.apk"
  cp app/build/outputs/bundle/release/app-release.aab "$ROOT/artifacts/$commit/LinkUp-v1.0-$commit-release.aab"
  sha256sum "$ROOT/artifacts/$commit/LinkUp-v1.0-$commit-release.apk" "$ROOT/artifacts/$commit/LinkUp-v1.0-$commit-release.aab" >> "$ROOT/artifacts/$commit/SHA256SUMS.txt"
fi

printf 'commit=%s\nartifacts=%s\n' "$commit" "$ROOT/artifacts/$commit"
