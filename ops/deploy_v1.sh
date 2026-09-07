#!/usr/bin/env bash
set -euo pipefail

ROOT="/opt/linkup"
SRC="$ROOT/src"
ENV_FILE="/etc/linkup/linkup.env"
SERVICE="linkup-api.service"
commit="${1:-$(git -C "$SRC" rev-parse HEAD)}"
artifact_dir="$ROOT/artifacts/$commit"
artifact="$artifact_dir/linkup-api"
live="$ROOT/bin/linkup-api"
previous="$ROOT/bin/linkup-api.previous"
next="$ROOT/bin/.linkup-api.$commit.next"

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

[ -f "$artifact" ] || fail "missing built runtime artifact: $artifact"
[ -f "$artifact_dir/SHA256SUMS.txt" ] || fail "missing artifact checksum ledger"
sudo test -f "$ENV_FILE" || fail "missing runtime environment file"
sudo grep -Eq '^DATABASE_URL=postgres(ql)?://' "$ENV_FILE" || fail "DATABASE_URL is not configured"

expected="$(awk '$2 ~ /\/linkup-api$/ {print $1; exit}' "$artifact_dir/SHA256SUMS.txt")"
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
