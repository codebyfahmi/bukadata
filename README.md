# BukaData Marketplace Intelligence API

A Go backend for turning marketplace capture data into queryable shop and product analytics. It accepts normalized capture batches, stores immutable snapshots, and exposes workspace-scoped dashboard and product endpoints for a companion client.

This repository is a portfolio implementation. It contains no production credentials, customer data, marketplace session data, or proprietary capture payloads.

## Highlights

- Multi-workspace data model with membership-aware access checks
- Snapshot-based analytics for shops and products
- Batch ingestion with job tracking and analytics refreshes
- Supabase Auth bearer-token validation
- Postgres persistence through `pgx`
- Optional Supabase Storage for large raw-capture payloads
- Vercel serverless entry point, plus a local Go HTTP server

## Architecture

```text
Client capture -> ingestion API -> Postgres snapshots -> dashboard/product API
                            \-> optional object storage for large raw payloads
```

The HTTP layer stays intentionally thin. Domain services coordinate ingestion, snapshot refreshes, workspace resolution, and read models; repository code owns SQL persistence.

## Requirements

- Go 1.24+
- PostgreSQL (Supabase Postgres is supported)
- A Supabase project for Auth; Storage is optional

## Run locally

1. Copy `.env.example` to `.env` and fill in values from your own Supabase project.
2. Run `sql/schema.sql` in the Supabase SQL editor to create the schema.
3. Start the API:

```powershell
go test ./...
go run ./cmd/server
```

The local server listens on `http://localhost:8080` by default. `GET /healthz` returns a basic health response.

## Environment variables

| Variable | Required | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `SUPABASE_URL` | Yes | Supabase project URL |
| `SUPABASE_ANON_KEY` | Yes | Supabase anonymous key used for Auth requests |
| `SUPABASE_SERVICE_ROLE_KEY` | Yes | Server-only Supabase key |
| `APP_ENV` | No | Environment name; defaults to `development` |
| `APP_BASE_URL` | No | Public base URL |
| `SUPABASE_JWT_SECRET` | No | JWT validation secret when needed |
| `RAW_CAPTURE_BUCKET` | No | Storage bucket for large raw payloads |
| `RAW_CAPTURE_INLINE_MAX_BYTES` | No | Inline payload threshold; defaults to `65536` |

Never commit a populated `.env` file or real capture payloads.

## API surface

All application routes except authentication require an `Authorization: Bearer <token>` header.

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/healthz` | Health check |
| `POST` | `/v1/extension/auth/login` | Authenticate through Supabase |
| `POST` | `/v1/extension/auth/refresh` | Refresh an access token |
| `POST` | `/v1/extension/ingest/captures/batch` | Ingest marketplace captures |
| `GET` | `/v1/extension/dashboard/shop` | Read the latest shop dashboard |
| `GET` | `/v1/extension/products` | Read product analytics |
| `POST` | `/v1/extension/analytics/refresh` | Recompute analytics |
| `GET` | `/v1/extension/jobs/{id}` | Read ingest or analytics job status |

## Deployment

`api/index.go` is the Vercel serverless entry point. The included `vercel.json` forwards application routes to it. Configure all required environment variables in the deployment platform; do not place them in source control.

## Privacy and responsible use

Marketplace captures may include sensitive request headers or user-provided data. This project is designed for data you are authorized to collect and process. Sanitize samples, restrict access to raw payloads, and comply with the applicable platform terms and privacy requirements.

## License

Released under the [MIT License](LICENSE).
