# syntax=docker/dockerfile:1
# Build from the repository root (context = "."), e.g.:
#   docker build -f devops-status-fe/Dockerfile -t devops-status-frontend .

FROM node:22-alpine AS build
WORKDIR /app

COPY devops-status-fe/package.json devops-status-fe/package-lock.json ./
RUN npm ci

COPY devops-status-fe/ .
RUN npm run build

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production
ENV HOST=0.0.0.0
ENV PORT=3000
COPY --from=build /app/.output ./.output
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
