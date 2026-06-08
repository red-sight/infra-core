#!/bin/sh
set -e

CONFIG=/run/infra/admin-app.json
DIST=/usr/share/nginx/html

if [ -f "$CONFIG" ]; then
  APP_ID=$(jq -r '.appId' "$CONFIG")
  ENDPOINT=$(jq -r '.endpoint' "$CONFIG")
  API_RESOURCE=$(jq -r '.apiResource' "$CONFIG")
  printf 'window.__INFRA_ADMIN_CONFIG__={logtoAppId:"%s",logtoEndpoint:"%s",logtoApiResource:"%s"};\n' \
    "$APP_ID" "$ENDPOINT" "$API_RESOURCE" \
    > "$DIST/config.js"
  echo "[admin] config.js generated (appId=$APP_ID)"
else
  echo "[admin] $CONFIG not found — config.js not generated"
  printf '' > "$DIST/config.js"
fi

exec nginx -g 'daemon off;'
