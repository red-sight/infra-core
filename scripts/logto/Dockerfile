FROM node:22-alpine
WORKDIR /app
COPY package.json .
RUN npm install --omit=dev
COPY init.js .
COPY logto.config.yaml .
CMD ["node", "init.js"]
