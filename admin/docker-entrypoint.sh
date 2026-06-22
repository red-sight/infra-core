#!/bin/sh
set -e

CONFIG=/run/infra/admin-app.json

if [ -f "$CONFIG" ]; then
  export VITE_OIDC_ISSUER=$(node -p "require('$CONFIG').issuer")
  export VITE_OIDC_CLIENT_ID=$(node -p "require('$CONFIG').clientId")
  export VITE_OIDC_PROJECT_ID=$(node -p "require('$CONFIG').projectId")
  echo "[admin] OIDC config loaded: clientId=$VITE_OIDC_CLIENT_ID"
else
  echo "[admin] $CONFIG not found — falling back to VITE_* env vars"
fi

exec "$@"
