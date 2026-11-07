# syntax=docker/dockerfile:1

FROM node:22-alpine AS web-builder
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26.5-alpine AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=web-builder /src/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/chess-server ./cmd/server

FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app
WORKDIR /app
COPY --from=go-builder --chown=app:app /out/chess-server /app/chess-server
COPY --from=go-builder --chown=app:app /src/migrations /app/migrations
COPY --from=web-builder --chown=app:app /src/web/dist /app/web/dist

ENV HTTP_ADDR=:8080 \
    WEB_DIR=/app/web/dist

USER app
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/chess-server"]
