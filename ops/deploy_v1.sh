#!/usr/bin/env bash
set -euo pipefail

ROOT="/opt/linkup"
SRC="$ROOT/src"
ARTIFACT_ROOT="$ROOT/artifacts"
ENV_FILE="/etc/linkup/linkup.env"
SECRET_ENV_FILE="/etc/linkup/linkup-api-secrets.env"
SERVICE="linkup-api.service"
STABLE_LINK="$ARTIFACT_ROOT/stable"

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

[ -L "$STABLE_LINK" ] || fail "no promoted stable artifact; run ops/promote_v1.sh first"
stable_target="$(readlink "$STABLE_LINK")"
case "$stable_target" in
  releases/*) ;;
  *) fail "invalid stable artifact link: $stable_target" ;;
esac
stable_commit="${stable_target#releases/}"
commit="${1:-$stable_commit}"
[ "$commit" = "$stable_commit" ] || fail "refusing to deploy non-stable commit $commit (stable is $stable_commit)"

artifact_dir="$ARTIFACT_ROOT/releases/$commit"
artifact="$artifact_dir/linkup-api"
live="$ROOT/bin/linkup-api"
previous="$ROOT/bin/linkup-api.previous"
next="$ROOT/bin/.linkup-api.$commit.next"

[ -f "$artifact" ] || fail "missing promoted runtime artifact: $artifact"
[ -f "$artifact_dir/SHA256SUMS.txt" ] || fail "missing artifact checksum ledger"
[ -f "$artifact_dir/BUILD_METADATA.txt" ] || fail "missing artifact metadata"
grep -qx "commit=$commit" "$artifact_dir/BUILD_METADATA.txt" || fail "artifact metadata commit mismatch"
grep -qx 'state=stable' "$artifact_dir/BUILD_METADATA.txt" || fail "artifact is not promoted stable"
sudo test -f "$ENV_FILE" || fail "missing runtime environment file"
if ! sudo grep -Eq '^DATABASE_URL=postgres(ql)?://' "$ENV_FILE" 2>/dev/null && \
   ! sudo grep -Eq '^DATABASE_URL=postgres(ql)?://' "$SECRET_ENV_FILE" 2>/dev/null; then
  fail "DATABASE_URL is not configured in runtime environment"
fi

(
  cd "$artifact_dir"
  sha256sum -c SHA256SUMS.txt
)

expected="$(awk '$2 == "linkup-api" {print $1; exit}' "$artifact_dir/SHA256SUMS.txt")"
actual="$(sha256sum "$artifact" | awk '{print $1}')"
[ -n "$expected" ] && [ "$actual" = "$expected" ] || fail "runtime artifact checksum mismatch"

sudo install -d -o root -g root -m 0755 "$ROOT/bin"
sudo install -o root -g root -m 0755 "$artifact" "$next"
was_active=0
if sudo systemctl is-active --quiet "$SERVICE"; then
  was_active=1
fi
if sudo test -f "$live"; then
  sudo install -o root -g root -m 0755 "$live" "$previous"
fi
sudo mv -f "$next" "$live"
sudo install -m 0644 "$SRC/ops/linkup-api.service" "/etc/systemd/system/$SERVICE"
sudo systemctl daemon-reload

rollback() {
  sudo systemctl stop "$SERVICE" 2>/dev/null || true
  if sudo test -f "$previous"; then
    sudo mv -f "$previous" "$live"
    if [ "$was_active" -eq 1 ]; then
      sudo systemctl restart "$SERVICE" || true
    fi
  fi
}
trap rollback ERR

sudo systemctl restart "$SERVICE"
healthy=0
for _ in $(seq 1 20); do
  if curl --fail --silent --show-error http://127.0.0.1:8080/livez >/dev/null && \
     curl --fail --silent --show-error http://127.0.0.1:8080/healthz >/dev/null; then
    healthy=1
    break
  fi
  sleep 1
done
[ "$healthy" -eq 1 ] || fail "LinkUp API failed local health checks"
sudo systemctl enable "$SERVICE" >/dev/null
trap - ERR
sudo rm -f "$previous"

printf 'deployed_commit=%s\nservice=%s\n' "$commit" "$(sudo systemctl is-active "$SERVICE")"
