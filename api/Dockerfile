# All targets are built from the REPO ROOT context so the web bundle can be
# embedded for preview deploys:
#   docker build -f api/Dockerfile --target prod .

# --- dev: hot reload for local compose (source is volume-mounted) ----------
FROM golang:1.22-alpine AS dev
WORKDIR /app
RUN go install github.com/air-verse/air@v1.52.3
COPY api/go.mod api/go.sum* ./
RUN go mod download
CMD ["air", "-c", ".air.toml"]

# --- web-build: vite bundle for single-container (preview) deploys ---------
FROM node:26-alpine AS web-build
ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0
RUN corepack enable
WORKDIR /web
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# --- build -----------------------------------------------------------------
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY api/go.mod api/go.sum* ./
RUN go mod download
COPY api/ ./
# Embed the built SPA for preview deploys; harmless in production where
# SERVE_STATIC is unset.
COPY --from=web-build /web/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/api ./cmd/api

# --- prod ------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS prod
COPY --from=build /bin/api /bin/api
EXPOSE 8080
ENTRYPOINT ["/bin/api"]
