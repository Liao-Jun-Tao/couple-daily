# Deploy the API

The container serves plain HTTP on port **8080**. HTTPS/TLS termination and any reverse proxy are **out of scope**: the process only calls `ListenAndServe`. Terminate TLS on the host or load balancer in front of this container (including an Oracle Cloud Always Free VM). This image does not configure certificates, security lists, or a proxy.

`backend/docker-compose.yml` starts Postgres, Redis, and MinIO only. It does not start the API.

Data is Postgres (`DATABASE_URL`). There is no SQLite mode. Object bytes go to an S3-compatible API (MinIO in local compose). The API presigns URLs; it does not store uploads on its own disk.

## Build

From the repository root (context is `backend/`, where `go.mod` and `./cmd/api` live; SQL migrations are embedded at compile time):

```bash
docker build -t couple-daily-api -f backend/Dockerfile backend
```

## Run

Use a private env file. Copy `backend/.env.example` and replace every placeholder. Do not commit it, and do not publish `APP_ENV=local` or the example secrets.

```bash
docker run --rm --name couple-daily-api \
  --env-file /path/to/api.env \
  -p 8080:8080 \
  couple-daily-api
```

`HTTP_ADDR` defaults to `:8080`. The image sets that value. The image `HEALTHCHECK` calls `GET http://127.0.0.1:8080/healthz` and requires HTTP **200**. If you change the listen port, the check no longer matches.

```bash
curl -fsS -D - http://127.0.0.1:8080/healthz
```

Expect `HTTP/1.1 200` and `{"status":"ok"}`. `/healthz` does not probe Postgres.

`localhost` inside the container is the container, not the VM. On the compose network the dependency hostnames are `postgres`, `redis`, and `minio` (S3 API port `9000`).

## Environment

`config.Load` reads the variables below. An empty value is treated as unset and replaced by the default in `backend/internal/config/config.go` (the same defaults as `backend/.env.example`).

`APP_ENV=local`, `dev`, or `development` may boot with `JWT_SECRET=change-me` and the MinIO example keys. Any other value, including an unset `APP_ENV`, refuses those secrets. `ENV` is used only when `APP_ENV` is empty. On a public host, leave `APP_ENV` unset or set it to something else (for example `production`).

**Required to boot on a public host** (non-local `APP_ENV`):

- `JWT_SECRET` — at least 32 characters, not empty, and not `change-me`. The default is `change-me`, which is refused.
- `S3_ENDPOINT`, `S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` — object storage is enabled when the endpoint or the bucket is non-empty. Both defaults are non-empty, and blank env vars fall back to those defaults, so storage stays enabled. All four must be set. Outside local/dev, the access key and secret must not be the example pair `minio` / `minio12345`.

**Required for the process to stay up:**

- `DATABASE_URL` — Postgres connection string. Startup applies embedded migrations and exits if Postgres is unreachable. The default `postgres://couple:couple@localhost:5432/couple_daily?sslmode=disable` is the compose database on the host loopback, which is the wrong address inside this container.

**Optional:**

- `HTTP_ADDR` — default `:8080`.
- `APP_ENV` — see the boot rule above. Do not set `local` on a public host.
- `REDIS_URL` — default `redis://localhost:6379/0`. If Redis does not answer, the process still listens. Refresh tokens are stored in Postgres (Redis only caches them). Reminder dedupe falls back to memory in this process. Login throttling is in-process, not Redis.
- `S3_REGION` — default `us-east-1`.
- `S3_USE_PATH_STYLE` — default `true` (MinIO). Set `false` for typical AWS virtual-hosted URLs.
- `JWT_ACCESS_TTL` — default `1h` (Go duration).
- `JWT_REFRESH_TTL` — default `720h`.
- `APNS_KEY_ID`, `APNS_TEAM_ID`, `APNS_BUNDLE_ID`, `APNS_KEY_PATH` — all four must be non-empty to send APNs. Otherwise delivery is a log stub (`push stub: provider=apns`) and `POST /devices` still registers tokens. Mount the `.p8` file at `APNS_KEY_PATH`; the image does not contain it.
- `APNS_USE_SANDBOX` — default `false`. Used only when APNs is configured.
- `FCM_PROJECT_ID`, `FCM_CREDENTIALS_PATH` — both must be set to send FCM HTTP v1. Otherwise delivery is a log stub (`push stub: provider=fcm`). Mount the service-account JSON; the image does not contain it.

No other environment variables are read.
