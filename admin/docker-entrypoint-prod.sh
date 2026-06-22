#!/bin/sh
set -e

CONFIG=/run/infra/admin-app.json
DIST=/usr/share/nginx/html

if [ -f "$CONFIG" ]; then
  ISSUER=$(jq -r '.issuer' "$CONFIG")
  CLIENT_ID=$(jq -r '.clientId' "$CONFIG")
  PROJECT_ID=$(jq -r '.projectId' "$CONFIG")
  printf 'window.__INFRA_ADMIN_CONFIG__={issuer:"%s",clientId:"%s",projectId:"%s"};\n' \
    "$ISSUER" "$CLIENT_ID" "$PROJECT_ID" \
    > "$DIST/config.js"
  echo "[admin] config.js generated (clientId=$CLIENT_ID)"
else
  echo "[admin] $CONFIG not found — config.js not generated"
  printf '' > "$DIST/config.js"
fi

exec nginx -g 'daemon off;'
