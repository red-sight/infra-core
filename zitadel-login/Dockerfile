# Forked ZITADEL Login UI v2 — see zitadel-login/README.md.
#
# Self-contained multi-stage build: unlike upstream (which builds the Next.js
# standalone bundle on the host via nx and only packages the result), this image
# runs the whole pipeline — proto codegen -> @zitadel/client -> next build.
# Build context is the workspace root (zitadel-login/).
#
# Build-time network is required for: the npm registry, the Buf Schema Registry
# (proto/buf.lock deps) and fonts.googleapis.com (next/font).

FROM node:24-alpine AS builder
RUN corepack enable
WORKDIR /workspace

# Manifests first so dependency installation caches independently of sources.
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY packages/zitadel-client/package.json packages/zitadel-client/
COPY packages/zitadel-proto/package.json packages/zitadel-proto/
COPY apps/login/package.json apps/login/
RUN pnpm install --frozen-lockfile

COPY . .
RUN pnpm run build

FROM node:24-alpine AS runtime
WORKDIR /app
RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs
# If /.env-file/.env is mounted into the container, its variables are made available to the server before it starts up.
RUN mkdir -p /.env-file && touch /.env-file/.env && chown -R nextjs:nodejs /.env-file

COPY --from=builder --chown=nextjs:nodejs /workspace/apps/login/.next/standalone ./

USER nextjs
ENV HOSTNAME="::" \
    PORT="3000" \
    NODE_ENV="production" \
    NODE_OPTIONS="--use-openssl-ca --require /app/load-ssl-cert-dir.cjs" \
    SSL_CERT_FILE="/etc/ssl/certs/ca-certificates.crt" \
    ZITADEL_TLS_ENABLED="false" \
    OTEL_SERVICE_NAME="zitadel-login" \
    OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf"

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/node", "/app/healthcheck.mjs", "/ui/v2/login/ready"]
ENTRYPOINT ["/app/entrypoint.sh", "node", "apps/login/server.js"]
