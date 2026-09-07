#!/usr/bin/env bash
set -euo pipefail

REPO_URL="https://github.com/vbdondarenko-cell/qweqwe.git"
ROOT="/opt/linkup"
SRC="$ROOT/src"
ANDROID_HOME="/opt/android-sdk"
GO_VERSION="1.27.1"
ANDROID_CMDLINE_REV="15859902"

sudo apt-get update
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y \
  ca-certificates curl git jq unzip zip openjdk-17-jdk-headless

arch="$(dpkg --print-architecture)"
case "$arch" in
  amd64) go_arch="amd64" ;;
  arm64) go_arch="arm64" ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac

go_file="go${GO_VERSION}.linux-${go_arch}.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

curl -fsSL 'https://go.dev/dl/?mode=json&include=all' -o "$tmp/go.json"
go_sha="$(jq -r --arg v "go${GO_VERSION}" --arg f "$go_file" '.[] | select(.version==$v) | .files[] | select(.filename==$f) | .sha256' "$tmp/go.json" | head -1)"
test -n "$go_sha" && test "$go_sha" != "null"
curl -fL "https://go.dev/dl/${go_file}" -o "$tmp/$go_file"
printf '%s  %s\n' "$go_sha" "$tmp/$go_file" | sha256sum -c -
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "$tmp/$go_file"
sudo ln -sf /usr/local/go/bin/go /usr/local/bin/go
sudo ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt

sudo install -d -o ubuntu -g ubuntu -m 0755 "$ANDROID_HOME/cmdline-tools"
curl -fL "https://dl.google.com/android/repository/commandlinetools-linux-${ANDROID_CMDLINE_REV}_latest.zip" -o "$tmp/cmdline.zip"
unzip -q "$tmp/cmdline.zip" -d "$tmp/android-cli"
rm -rf "$ANDROID_HOME/cmdline-tools/latest"
mv "$tmp/android-cli/cmdline-tools" "$ANDROID_HOME/cmdline-tools/latest"

export ANDROID_HOME
export ANDROID_SDK_ROOT="$ANDROID_HOME"
export PATH="/usr/local/go/bin:$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$PATH"
yes | sdkmanager --licenses >/dev/null
sdkmanager 'platform-tools' 'platforms;android-37' 'build-tools;36.0.0'

sudo install -d -o ubuntu -g ubuntu -m 0755 "$ROOT" "$ROOT/bin" "$ROOT/artifacts"
if [ -d "$SRC/.git" ]; then
  git -C "$SRC" fetch --prune origin main
  git -C "$SRC" checkout main
  git -C "$SRC" reset --hard origin/main
else
  git clone --branch main --single-branch "$REPO_URL" "$SRC"
fi

printf 'sdk.dir=%s\n' "$ANDROID_HOME" > "$SRC/android/local.properties"

if ! id linkup >/dev/null 2>&1; then
  sudo useradd --system --home /nonexistent --shell /usr/sbin/nologin linkup
fi
sudo install -d -o root -g linkup -m 0750 /etc/linkup
sudo install -m 0644 "$SRC/ops/linkup-api.service" /etc/systemd/system/linkup-api.service
sudo systemctl daemon-reload

java -version
go version
"$ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager" --version
git -C "$SRC" rev-parse HEAD
