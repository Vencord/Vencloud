FROM node:lts-slim

RUN npm install --global pnpm

WORKDIR /app

ADD package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

ADD . ./
RUN pnpm compile

CMD ["node", "dist/index.js"]
