# Build context is the backend module directory (go.mod, ./cmd/api, embedded migrations).
#   docker build -t couple-daily-api -f backend/Dockerfile backend

FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# migrations/*.sql are compiled in via //go:embed in package migrations.
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.21

RUN apk add --no-cache ca-certificates curl \
    && adduser -D -H -u 65532 -s /sbin/nologin app

COPY --from=build /out/api /usr/local/bin/api

USER app

ENV HTTP_ADDR=:8080
EXPOSE 8080

# GET /healthz must return HTTP 200. This assumes HTTP_ADDR stays :8080.
HEALTHCHECK --interval=30s --timeout=3s --start-period=20s --retries=3 \
    CMD sh -c 'code=$(curl -sS -o /dev/null -w "%{http_code}" http://127.0.0.1:8080/healthz); test "$code" = 200'

ENTRYPOINT ["/usr/local/bin/api"]
