FROM node:22-alpine AS dev
ARG UID=1000
ARG GID=1000
RUN addgroup -g ${GID} -S devgroup 2>/dev/null || true && \
    adduser -D -u ${UID} -G devgroup -S devuser 2>/dev/null || true
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh
WORKDIR /app
RUN chown ${UID}:${GID} /app
USER ${UID}:${GID}
COPY --chown=${UID}:${GID} package*.json ./
RUN npm install
EXPOSE 5173
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["npm", "run", "dev"]

FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install --frozen-lockfile
COPY . .
RUN npm run build

FROM nginx:alpine AS prod
RUN apk add --no-cache jq
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html
COPY docker-entrypoint-prod.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh
EXPOSE 80
ENTRYPOINT ["docker-entrypoint.sh"]
