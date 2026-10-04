# AGENTS.md — api-recaptcha

Small Go + Gin REST API (`module api-recaptcha`, Go 1.22) that validates reCAPTCHA Enterprise tokens. Single binary, no DB.

## Entrypoints & layout

- `cmd/server/main.go` — only entrypoint. Wires env → `service.NewRecaptchaService` → `handler.NewVerifyHandler`, registers routes, graceful shutdown (10s).
- `internal/handler/` — `verify.go` (`POST /api/v1/recaptcha/verify`, auth'd), `health.go` (`GET /health`, `GET /ready`, no auth).
- `internal/service/recaptcha.go` — `Assessor` interface + `RecaptchaService`. Real Google calls live only here.
- `internal/middleware/` — `apikey.go` (auth), `ratelimit.go` (per-IP token bucket), `cors.go`.
- `internal/errors/errors.go` — `AppError{Code, Message, HTTPStatus, Internal}`. Handlers must return `Message`/`Code` only; log `Internal` server-side.
- `internal/logger/logger.go` — structured `log/slog` JSON logger, level via `LOG_LEVEL`.

## Setup & run

- Copy env first: `make setup-env` (or `cp .env.example .env`), then edit credentials. `.env` is gitignored; `godotenv.Load()` falls back to system env if missing.
- Required at startup or the process `os.Exit(1)`: `APP_API_KEY`, `GOOGLE_RECAPTCHA_API_KEY`, `GOOGLE_RECAPTCHA_PROJECT_ID`. `GOOGLE_RECAPTCHA_SITE_KEY` falls back to a hardcoded default (warns; not for production). `PORT` defaults to `8080`.
- `make run` / `make build && ./bin/api-recaptcha` / `make dev` (needs `air`, not vendored).
- Docker: `make docker-build docker-run` (uses `--env-file .env`); compose healthcheck hits `/health`.

## Verify

- `make test` (`go test -v ./...`) — unit tests only, no external services/credentials needed (handlers mock the `Assessor` interface).
- Single package: `go test ./internal/handler/ -run TestVerifyHandler -v`; `go test -cover ./...` for coverage.
- `make fmt`; `make lint` requires external `golangci-lint` (not in `go.mod`).

## Conventions — do not regress

- API-key auth: client header is `X-API-Key`; compare with `crypto/subtle.ConstantTimeCompare`, never `==`. Missing → 401, wrong → 403 (see `middleware/apikey.go`).
- Google key transport: send via `X-goog-api-key` request header in `service/recaptcha.go`, never as URL query param (avoids leaking in logs).
- Input limits in `Assessor.Assess`: token required, max 2000 chars; action max 100 chars, trimmed; `expectedAction` omitted when empty.
- Error shape: `{ "error": "<user-safe>", "code": "<ERROR_CODE>" }`. Never expose `err.Error()` / internal details to clients.
- Middleware order on `/api/v1`: `RateLimit()` then `APIKeyAuth()`. Health endpoints stay outside the auth group.
- Rate limiter config via `RATE_LIMIT_REQUESTS` (default 100) / `RATE_LIMIT_WINDOW_SECONDS` (default 60); constructed once in `main` with deferred `Stop()`.
- HTTP timeouts: server 15s read/write, 60s idle; outbound Google client 10s. Keep them.
