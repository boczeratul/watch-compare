# WatchCompare

Find the best deal for a watch across marketplaces. A nightly crawler indexes listings from
**Chrono24, eBay, Watchnian, Jackroad, Commit Ginza, 7hours, Brand Shop LIPS, ALLU, Housekihiroba, BEST ISHIDA, Hourstack and RD Watch** into PostgreSQL; a Next.js frontend lets you
search, filter and sort them Chrono24-style, in your own currency and language.

```
┌────────────┐   02:00 Asia/Taipei   ┌───────────────────┐        ┌──────────────┐
│ Cloud      │ ────────────────────▶ │ Cloud Run Job     │ ─────▶ │ Cloud SQL    │
│ Scheduler  │                       │ backend/cmd/crawler│        │ PostgreSQL 16│
└────────────┘                       └───────────────────┘        └──────┬───────┘
                                                                         │
┌────────────┐   fetch (RSC, ISR)    ┌───────────────────┐               │
│ Vercel     │ ────────────────────▶ │ Cloud Run Service │ ◀─────────────┘
│ Next.js 16 │                       │ backend/cmd/api   │
└────────────┘                       └───────────────────┘
```

| Part | Stack | Deploy |
|------|-------|--------|
| `backend/` | Go 1.27, chi, pgx, goquery, zerolog | Cloud Build → Artifact Registry → Cloud Run (service + job) |
| `frontend/` | Next.js 16 (App Router, RSC), React 19, Tailwind v4, next-intl | Vercel (Git integration) |
| Database | PostgreSQL 16 (Cloud SQL) with full-text search + facets | migrations embedded in the Go binary |

Docs: [Architecture](docs/ARCHITECTURE.md) · [Crawlers](docs/CRAWLERS.md) · [CI/CD & Deployment](docs/DEPLOYMENT.md)

## Quick start (local)

Prerequisites: Go 1.27+, Node 24+, pnpm, and a PostgreSQL 16 (Docker: `cd backend && docker compose up -d`).

```bash
# 1. Backend API (applies migrations on boot)
cd backend
cp .env.example .env            # edit DATABASE_URL if needed
set -a; source .env; set +a
make run                        # http://localhost:8080/healthz

# 2. Populate the index (a couple of pages per site is enough to try the UI)
make crawl ARGS="-sources=jackroad,hourstack,watchnian,rdwatch,commitwatch,sevenhours,lips,allu,housekihiroba,ishida -max-pages=2"

# 3. Frontend
cd ../frontend
pnpm install
cp .env.example .env.local      # API_URL=http://localhost:8080
pnpm dev                        # http://localhost:3000
```

Try a dry run of a single crawler without touching the database:

```bash
cd backend && make crawl-dry ARGS="-sources=jackroad"
```

## API at a glance

All endpoints are `GET`, JSON, CORS-enabled for the configured origins, and cached for `CACHE_TTL`.

| Endpoint | Purpose |
|----------|---------|
| `/api/v1/listings` | Search. Params: `q, brand, model, ref, source, condition, movement, gender, country, dial, price_min, price_max, currency, year_min, year_max, diameter_min, diameter_max, box, papers, sort, page, per_page`; facets include brands, sources, conditions, movements, countries, dial colours and years |
| `/api/v1/listings/{id}` | One listing |
| `/api/v1/listings/{id}/similar` | Same reference on other marketplaces, cheapest first |
| `/api/v1/listings/{id}/price-history` | Observed price points |
| `/api/v1/brands` · `/api/v1/sources` · `/api/v1/rates` · `/api/v1/stats` · `/api/v1/crawls` | Reference data & ops |
| `/healthz` · `/readyz` | Liveness / DB readiness |

`sort` accepts `relevance, newest, oldest, price_asc, price_desc, year_desc, year_asc, size_asc, size_desc`.
`price_min`/`price_max` are interpreted in `currency` (default USD) and converted server-side; all
cross-marketplace sorting uses `price_usd`, which is computed at crawl time with that day's rates.

## Testing

```bash
cd backend && make test                      # unit + fixture-based parser tests
TEST_EMBEDDED_PG=1 go test ./internal/repository/ -run TestRepositoryEndToEnd   # real Postgres via embedded-postgres
cd frontend && pnpm lint && pnpm typecheck && pnpm build
```

## Legal note

Only index sites whose terms allow it. eBay is accessed through its official Browse API. Chrono24
blocks unattended requests; the adapter is provided for use with a licensed data feed or an
authorized rendering proxy (see [docs/CRAWLERS.md](docs/CRAWLERS.md)). Images are never stored, only
linked to their original location.
