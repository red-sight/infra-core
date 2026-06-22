#!/bin/sh
set -e

CONFIG=/run/infra/tenant-app.json

if [ -f "$CONFIG" ]; then
  export VITE_OIDC_ISSUER=$(node -p "require('$CONFIG').issuer")
  export VITE_OIDC_CLIENT_ID=$(node -p "require('$CONFIG').clientId")
  export VITE_OIDC_PROJECT_ID=$(node -p "require('$CONFIG').projectId")
  echo "[tenant-web] OIDC config loaded: clientId=$VITE_OIDC_CLIENT_ID"
else
  echo "[tenant-web] $CONFIG not found — falling back to VITE_* env vars"
fi

exec "$@"
