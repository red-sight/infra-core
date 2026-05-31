FROM node:22-alpine AS base
WORKDIR /app
RUN chown node:node /app
USER node
COPY --chown=node:node package*.json ./
RUN npm ci

FROM base AS dev
CMD ["npm", "run", "start:dev"]

FROM base AS build
COPY --chown=node:node . .
RUN npm run build

FROM node:22-alpine AS prod
WORKDIR /app
RUN chown node:node /app
USER node
COPY --chown=node:node package*.json ./
RUN npm ci --omit=dev
COPY --from=build --chown=node:node /app/dist ./dist
ENTRYPOINT ["node", "dist/main"]
