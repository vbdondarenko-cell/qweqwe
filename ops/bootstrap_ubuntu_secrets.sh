#!/usr/bin/env bash
set -euo pipefail

if [ "${EUID}" -ne 0 ]; then
  exec sudo -E "$0" "$@"
fi

PROJECT_REF="oavnrlwsfiiehluubwjk"
API_ROLE="linkup_api"
DB_HOST="${LINKUP_SUPABASE_DB_HOST:-aws-1-eu-west-1.pooler.supabase.com}"
DB_PORT="${LINKUP_SUPABASE_DB_PORT:-6543}"
DB_NAME="${LINKUP_SUPABASE_DB_NAME:-postgres}"
ADMIN_USER="${LINKUP_SUPABASE_ADMIN_USER:-postgres.${PROJECT_REF}}"
SECRET_ENV="/etc/linkup/linkup-api-secrets.env"
BUILD_ENV="/etc/linkup/build.env"
FIREBASE_SA="/etc/linkup/firebase-service-account.json"
JKS_FILE="/etc/linkup/linkup-release-key.jks"
JKS_ALIAS="linkup-key-alias"

fail() { echo "ERROR: $*" >&2; exit 1; }

command -v openssl >/dev/null || fail "openssl is required"
command -v psql >/dev/null || fail "psql is required"
command -v keytool >/dev/null || fail "keytool is required"
[ -s "$BUILD_ENV" ] || fail "$BUILD_ENV is missing"
[ -s "$FIREBASE_SA" ] || fail "$FIREBASE_SA is missing"

if [ -n "${LINKUP_SUPABASE_ADMIN_PASSWORD:-}" ]; then
  ADMIN_PASS="$LINKUP_SUPABASE_ADMIN_PASSWORD"
else
  if [ ! -t 0 ]; then
    fail "interactive terminal required for Supabase postgres password"
  fi
  read -rsp "Supabase postgres password: " ADMIN_PASS
  echo
fi
[ -n "$ADMIN_PASS" ] || fail "Supabase postgres password is empty"

RUNTIME_PASS="$(openssl rand -hex 32)"
export PGHOST="$DB_HOST" PGPORT="$DB_PORT" PGDATABASE="$DB_NAME" PGUSER="$ADMIN_USER" PGPASSWORD="$ADMIN_PASS" PGSSLMODE=require
psql -v ON_ERROR_STOP=1 -v runtime_pass="$RUNTIME_PASS" -v api_role="$API_ROLE" >/dev/null <<'SQL'
SELECT format(
  'ALTER ROLE %I WITH LOGIN PASSWORD %L CONNECTION LIMIT 20',
  :'api_role',
  :'runtime_pass'
) \gexec
SQL
unset ADMIN_PASS LINKUP_SUPABASE_ADMIN_PASSWORD PGPASSWORD

export PGUSER="${API_ROLE}.${PROJECT_REF}" PGPASSWORD="$RUNTIME_PASS"
psql -v ON_ERROR_STOP=1 -Atqc 'select current_user' >/dev/null
unset PGPASSWORD PGUSER

echo "DATABASE LOGIN: OK"

PUSH_KEY="$(openssl rand -base64 32 | tr -d '\n')"
RUNTIME_URL="postgresql://${API_ROLE}.${PROJECT_REF}:${RUNTIME_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=require"
umask 077
TMP_SECRET="$(mktemp)"
cat > "$TMP_SECRET" <<EOF
DATABASE_URL=${RUNTIME_URL}
LINKUP_PUSH_TOKEN_KEY_ID=production-v1
LINKUP_PUSH_TOKEN_KEY_BASE64=${PUSH_KEY}
LINKUP_FIREBASE_PROJECT_ID=linkup-4b782
LINKUP_FIREBASE_SERVICE_ACCOUNT_FILE=${FIREBASE_SA}
EOF
install -o root -g linkup -m 640 "$TMP_SECRET" "$SECRET_ENV"
rm -f "$TMP_SECRET"
unset RUNTIME_URL RUNTIME_PASS PUSH_KEY

echo "RUNTIME SECRETS: OK"

if [ ! -s "$JKS_FILE" ]; then
  JKS_PASS="$(openssl rand -hex 32)"
  export LINKUP_JKS_PASS="$JKS_PASS"
  TMP_JKS="$(mktemp --suffix=.jks)"
  rm -f "$TMP_JKS"
  keytool -genkeypair -noprompt \
    -alias "$JKS_ALIAS" \
    -keyalg RSA -keysize 3072 -validity 36500 \
    -dname "CN=LinkUp, O=LinkUp, C=UA" \
    -keystore "$TMP_JKS" -storetype PKCS12 \
    -storepass:env LINKUP_JKS_PASS -keypass:env LINKUP_JKS_PASS >/dev/null 2>&1
  install -o root -g ubuntu -m 640 "$TMP_JKS" "$JKS_FILE"
  rm -f "$TMP_JKS"

  TMP_BUILD="$(mktemp)"
  grep -vE '^ORG_GRADLE_PROJECT_LINKUP_(KEYSTORE_FILE|KEYSTORE_PASSWORD|KEY_ALIAS|KEY_PASSWORD)=' "$BUILD_ENV" > "$TMP_BUILD" || true
  cat >> "$TMP_BUILD" <<EOF
ORG_GRADLE_PROJECT_LINKUP_KEYSTORE_FILE=${JKS_FILE}
ORG_GRADLE_PROJECT_LINKUP_KEYSTORE_PASSWORD=${JKS_PASS}
ORG_GRADLE_PROJECT_LINKUP_KEY_ALIAS=${JKS_ALIAS}
ORG_GRADLE_PROJECT_LINKUP_KEY_PASSWORD=${JKS_PASS}
EOF
  install -o root -g ubuntu -m 640 "$TMP_BUILD" "$BUILD_ENV"
  rm -f "$TMP_BUILD"
  unset LINKUP_JKS_PASS JKS_PASS
fi

echo "ANDROID SIGNING: OK"

systemctl daemon-reload
printf 'firebase_service_account=OK\ndatabase_login=OK\npush_encryption=OK\nrelease_jks=OK\n'
