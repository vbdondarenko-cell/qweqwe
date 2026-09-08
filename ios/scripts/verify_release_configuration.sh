#!/bin/sh
set -eu

if [ "${CONFIGURATION:-}" != "Release" ]; then
  exit 0
fi

fail() {
  echo "error: LinkUp iOS release configuration: $1" >&2
  exit 1
}

require_https() {
  name="$1"
  value="$2"
  case "$value" in
    https://*) ;;
    *) fail "$name must be a non-empty https:// URL" ;;
  esac
  case "$value" in
    *' '*|*'@'*|*'#'*) fail "$name contains an unsafe URL component" ;;
  esac
}

api="${LINKUP_API_BASE_URL:-}"
reset="${LINKUP_RECOVERY_RESET_URL:-}"
associated="${LINKUP_RECOVERY_ASSOCIATED_DOMAIN:-}"
privacy="${LINKUP_PRIVACY_URL:-}"
terms="${LINKUP_TERMS_URL:-}"

require_https LINKUP_API_BASE_URL "$api"
require_https LINKUP_RECOVERY_RESET_URL "$reset"
require_https LINKUP_PRIVACY_URL "$privacy"
require_https LINKUP_TERMS_URL "$terms"

for pair in "LINKUP_API_BASE_URL=$api" "LINKUP_RECOVERY_RESET_URL=$reset" "LINKUP_PRIVACY_URL=$privacy" "LINKUP_TERMS_URL=$terms"; do
  name=${pair%%=*}
  value=${pair#*=}
  authority=${value#https://}
  authority=${authority%%/*}
  authority=${authority%%\?*}
  host=${authority%%:*}
  case "$host" in
    *.invalid) fail "$name must not use a reserved .invalid host for Release" ;;
  esac
done

case "$reset" in
  *'?'*) fail "LINKUP_RECOVERY_RESET_URL must not contain a query" ;;
esac

case "$associated" in
  applinks:*) ;;
  *) fail "LINKUP_RECOVERY_ASSOCIATED_DOMAIN must use applinks:<host>" ;;
esac
associated_host=${associated#applinks:}
[ -n "$associated_host" ] || fail "LINKUP_RECOVERY_ASSOCIATED_DOMAIN host is empty"
[ "$associated_host" != "example.invalid" ] || fail "reserved example.invalid recovery domain cannot be used for Release"
case "$associated_host" in
  *'/'*|*':'*|*'?'*|*'#'*|*'@'*|*' '*) fail "LINKUP_RECOVERY_ASSOCIATED_DOMAIN must contain a bare host" ;;
esac

reset_authority=${reset#https://}
reset_authority=${reset_authority%%/*}
reset_authority=${reset_authority%%\?*}
[ "$reset_authority" = "$associated_host" ] || fail "reset URL host must exactly match the Associated Domains host"

api_authority=${api#https://}
case "$api_authority" in
  */*)
    api_path=${api_authority#*/}
    [ -z "$api_path" ] || fail "LINKUP_API_BASE_URL must be an origin without a base path"
    ;;
esac
case "$api_authority" in
  *'?'*|*'#'*) fail "LINKUP_API_BASE_URL must not contain query or fragment" ;;
esac

exit 0
