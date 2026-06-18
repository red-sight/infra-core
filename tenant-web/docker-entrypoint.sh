#!/bin/sh
set -e

CONFIG=/run/infra/tenant-app.json

if [ -f "$CONFIG" ]; then
  export VITE_LOGTO_APP_ID=$(node -p "require('$CONFIG').appId")
  export VITE_LOGTO_ENDPOINT=$(node -p "require('$CONFIG').endpoint")
  export VITE_LOGTO_API_RESOURCE=$(node -p "require('$CONFIG').apiResource")
  echo "[tenant-web] Logto config loaded: appId=$VITE_LOGTO_APP_ID"
else
  echo "[tenant-web] $CONFIG not found — falling back to VITE_* env vars"
fi

exec "$@"
