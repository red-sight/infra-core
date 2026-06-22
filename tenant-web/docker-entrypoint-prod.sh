#!/bin/sh
set -e

CONFIG=/run/infra/tenant-app.json
DIST=/usr/share/nginx/html

if [ -f "$CONFIG" ]; then
  ISSUER=$(jq -r '.issuer' "$CONFIG")
  CLIENT_ID=$(jq -r '.clientId' "$CONFIG")
  PROJECT_ID=$(jq -r '.projectId' "$CONFIG")
  printf 'window.__INFRA_TENANT_CONFIG__={issuer:"%s",clientId:"%s",projectId:"%s"};\n' \
    "$ISSUER" "$CLIENT_ID" "$PROJECT_ID" \
    > "$DIST/config.js"
  echo "[tenant-web] config.js generated (clientId=$CLIENT_ID)"
else
  echo "[tenant-web] $CONFIG not found — config.js not generated"
  printf '' > "$DIST/config.js"
fi

exec nginx -g 'daemon off;'
