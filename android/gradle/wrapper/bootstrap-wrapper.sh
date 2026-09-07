#!/bin/sh
set -eu

WRAPPER_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
TARGET="$WRAPPER_DIR/gradle-wrapper.jar"
EXPECTED_SHA256="497c8c2a7e5031f6aa847f88104aa80a93532ec32ee17bdb8d1d2f67a194a9c7"
URL="https://raw.githubusercontent.com/gradle/gradle/v9.6.0/gradle/wrapper/gradle-wrapper.jar"

if [ -f "$TARGET" ]; then
    exit 0
fi

TMP="$TARGET.tmp.$$"
trap 'rm -f "$TMP"' EXIT HUP INT TERM

if command -v curl >/dev/null 2>&1; then
    curl --fail --location --silent --show-error --proto '=https' --tlsv1.2 --output "$TMP" "$URL"
elif command -v wget >/dev/null 2>&1; then
    wget --https-only --secure-protocol=TLSv1_2 --output-document="$TMP" "$URL"
else
    echo "ERROR: curl or wget is required to bootstrap the verified Gradle wrapper JAR." >&2
    exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "$TMP" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
    ACTUAL=$(shasum -a 256 "$TMP" | awk '{print $1}')
elif command -v openssl >/dev/null 2>&1; then
    ACTUAL=$(openssl dgst -sha256 "$TMP" | awk '{print $NF}')
else
    echo "ERROR: sha256sum, shasum, or openssl is required to verify the Gradle wrapper JAR." >&2
    exit 1
fi

if [ "$ACTUAL" != "$EXPECTED_SHA256" ]; then
    echo "ERROR: Gradle wrapper JAR checksum mismatch." >&2
    echo "Expected: $EXPECTED_SHA256" >&2
    echo "Actual:   $ACTUAL" >&2
    exit 1
fi

mv "$TMP" "$TARGET"
trap - EXIT HUP INT TERM
