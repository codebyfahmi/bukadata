# BukaData Marketplace Intelligence API

Backend API server Go untuk ingest capture marketplace, membentuk snapshot analytics, dan menyajikan dashboard shop/product untuk Chrome extension.

## Stack

- Go `net/http`
- Supabase Postgres
- Supabase Auth
- Optional Supabase Storage untuk payload raw besar
- Target deploy: Vercel serverless (`api/index.go`)

## Struktur

- `api/index.go`: entrypoint Vercel
- `internal/app`: bootstrap handler + routing
- `internal/auth`: validasi bearer token ke Supabase Auth
- `internal/analytics`: source election, product extraction, dashboard shaping
- `internal/config`: env loading
- `internal/dashboard`: pembacaan latest dashboard snapshot
- `internal/httpx`: util response/error
- `internal/ingest`: ingest pipeline dan analytics recompute
- `internal/jobs`: pembacaan ingest/analytics jobs
- `internal/model`: request/response DTO
- `internal/observability`: request logging dan request ID
- `internal/repository`: seluruh query SQL dan persistence primitives
- `internal/service`: HTTP boundary dan orchestration tipis
- `internal/storage`: penyimpanan raw payload besar ke Supabase Storage
- `internal/workspace`: resolusi header/query workspace slug
- `sql/schema.sql`: schema awal Postgres
- `sql/migrations`: migration versioned awal

## Environment

Minimal:

- `DATABASE_URL`
- `SUPABASE_URL`
- `SUPABASE_ANON_KEY`
- `SUPABASE_SERVICE_ROLE_KEY`
- `APP_ENV`
- `APP_BASE_URL`

Opsional:

- `SUPABASE_JWT_SECRET`
- `RAW_CAPTURE_BUCKET`
- `LOG_LEVEL`
- `RAW_CAPTURE_INLINE_MAX_BYTES`
- `SERVER_READ_TIMEOUT`
- `SERVER_WRITE_TIMEOUT`

Catatan:

- `DATABASE_URL` diperlukan karena backend ini menulis langsung ke Postgres untuk transaksi snapshot latest.
- `RAW_CAPTURE_BUCKET` dipakai bila payload raw besar perlu dialihkan ke Supabase Storage.

## Menjalankan Lokal

```powershell
go mod tidy
go test ./...
```

## Deploy Vercel

Vercel akan menggunakan `api/index.go` sebagai serverless entrypoint. Semua route aplikasi diarahkan lewat handler ini.

## Database

Jalankan [sql/schema.sql](sql/schema.sql) di Supabase SQL editor untuk bootstrap schema dan indeks awal.
