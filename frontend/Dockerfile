# ---------- build stage ----------
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci
COPY . .
# Empty: nginx proxies /api/ to the backend on the same origin, so relative paths work everywhere this is deployed.
ARG PUBLIC_API_URL=""
ENV PUBLIC_API_URL=$PUBLIC_API_URL
RUN npm run build
RUN npm run build.server

# ---------- runtime stage ----------
FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/dist /app/dist
COPY --from=builder /app/server /app/server
COPY --from=builder /app/node_modules /app/node_modules
COPY --from=builder /app/package.json /app/package.json
ENV PORT=3000
EXPOSE 3000
CMD ["node", "server/entry.node-server.js"]
