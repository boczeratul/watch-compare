# CI/CD & Deployment

Two independent pipelines, both triggered by pushes to `main`:

| Component | Pipeline | Target |
|-----------|----------|--------|
| `frontend/` | Vercel Git integration (preview per PR, production on `main`) | Vercel |
| `backend/`  | Cloud Build trigger → `backend/cloudbuild.yaml` | Artifact Registry, Cloud Run service `watch-compare-api`, Cloud Run job `watch-compare-crawler` |
| both | GitHub Actions `ci.yml` (lint, typecheck, tests) | PR checks |

Cloud Scheduler runs the crawler job every day at **02:00 Asia/Taipei**.

---

## 1. GCP one-time setup

Everything below is scripted in [`infra/gcp/setup.sh`](../infra/gcp/setup.sh). Read it, set the
variables at the top, then run it once from a machine with `gcloud` authenticated as a project owner.
What it does:

1. Enables APIs: Cloud Run, Cloud Build, Artifact Registry, Cloud SQL Admin, Secret Manager,
   Cloud Scheduler, IAM.
2. Creates an Artifact Registry Docker repo `watch-compare` in `$REGION`.
3. Creates a Cloud SQL **PostgreSQL 16** instance (`db-g1-small` is plenty; enable automatic
   backups), database `watch`, user `watch` with a generated password.
4. Stores secrets in Secret Manager: `DATABASE_URL` (Unix-socket DSN for Cloud SQL), `EBAY_CLIENT_ID`,
   `EBAY_CLIENT_SECRET` (placeholders you fill in later).
5. Creates two service accounts:
   * `watch-compare-runtime` — used by the API service and the crawler job: `roles/cloudsql.client`,
     `roles/secretmanager.secretAccessor`, `roles/logging.logWriter`.
   * `watch-compare-cloudbuild` — used by the Cloud Build trigger: `roles/run.admin`,
     `roles/artifactregistry.writer`, `roles/iam.serviceAccountUser` (on the runtime SA),
     `roles/logging.logWriter`, `roles/cloudsql.client`.
   * `watch-compare-scheduler` — used by Cloud Scheduler: `roles/run.invoker` on the job.
6. Creates the Cloud Build trigger connected to your GitHub repo (`backend/**` path filter, branch
   `^main$`, config `backend/cloudbuild.yaml`, substitutions listed in the file header).
7. Creates the Cloud Scheduler job `watch-compare-nightly-crawl` (`0 2 * * *`, `Asia/Taipei`) that
   calls the Cloud Run Jobs API `…/jobs/watch-compare-crawler:run` with an OAuth token from the
   scheduler SA.

The first Cloud Build run creates the Cloud Run service and job; the scheduler job will start
succeeding after that.

### Extensions

Migration 0002 runs `CREATE EXTENSION IF NOT EXISTS pg_trgm`. On Cloud SQL this needs a user with
the `cloudsqlsuperuser` role — users created with `gcloud sql users create` (as in `setup.sh`)
have it. pg_trgm is on Cloud SQL's supported-extension list.

### Database URL format

The API and crawler connect through the Cloud SQL connector socket that Cloud Run mounts when
`--set-cloudsql-instances` is set:

```
postgres://watch:<password>@/watch?host=/cloudsql/<PROJECT>:<REGION>:<INSTANCE>&sslmode=disable
```

Locally use a TCP DSN (`postgres://watch:watch@localhost:5432/watch?sslmode=disable`) or the
Cloud SQL Auth Proxy.

---

## 2. Backend pipeline (`backend/cloudbuild.yaml`)

Steps, in order:

1. **test** — `go vet` + `go test -race ./...` in the official `golang:1.25` image. Any failure
   stops the build before an image is produced.
2. **build** — multi-stage Docker build (`golang:1.25-alpine` → `distroless/static`, non-root).
   The image is tagged with `$SHORT_SHA` and `latest`; `--cache-from latest` keeps builds fast.
3. **push** — to `REGION-docker.pkg.dev/PROJECT/watch-compare/backend`.
4. **migrate** — deploys a tiny Cloud Run job `watch-compare-api-migrate` from the same image with
   `--command=/app/migrate --args=up` and executes it with `--wait`. Migrations are embedded in the
   binary, idempotent, and guarded by an advisory lock. The API also runs them on boot unless
   `AUTO_MIGRATE=false` (which production sets, because step 4 already did it).
5. **deploy-api** — `gcloud run deploy` with Cloud SQL attached, secrets mounted as env vars,
   `min-instances=0`, `max-instances=10`, `concurrency=80`. Traffic moves to the new revision only
   after it passes startup checks; the previous revision stays available for rollback.
6. **deploy-crawler** — `gcloud run jobs deploy` pointing the job at the same image
   (`--command=/app/crawler`, `--task-timeout=5h`, `--max-retries=1`).

### Trigger substitutions

| Key | Example |
|-----|---------|
| `_REGION` | `asia-east1` |
| `_AR_REPO` | `watch-compare` |
| `_API_SERVICE` | `watch-compare-api` |
| `_CRAWLER_JOB` | `watch-compare-crawler` |
| `_CLOUDSQL` | `my-project:asia-east1:watch-compare-pg` |
| `_RUNTIME_SA` | `watch-compare-runtime@my-project.iam.gserviceaccount.com` |
| `_CORS_ORIGINS` | `https://watch-compare.vercel.app,https://watchcompare.example` |

### Manual operations

```bash
# Run the crawler now (same as the nightly trigger)
gcloud run jobs execute watch-compare-crawler --region asia-east1

# Run a single source with a page cap
gcloud run jobs execute watch-compare-crawler --region asia-east1 \
  --args="-sources=jackroad,-max-pages=5"

# Tail logs
gcloud run services logs tail watch-compare-api --region asia-east1
gcloud logging read 'resource.type="cloud_run_job" AND resource.labels.job_name="watch-compare-crawler"' --limit 100

# Roll back the API to the previous revision
gcloud run services update-traffic watch-compare-api --region asia-east1 --to-revisions PREVIOUS_REVISION=100

# Roll back the last migration
gcloud run jobs execute watch-compare-api-migrate --region asia-east1 --args=down --wait
```

### Why these choices

* **One image, three entrypoints** keeps the API and crawler on the same commit and halves build
  time; Cloud Run jobs just override the command.
* **Distroless static, non-root** image: no shell, ~15 MB, fewer CVEs.
* **Migrations as a job step** rather than on API boot in production: a failed migration fails the
  build instead of crash-looping the service.
* **Cloud Run Job + Cloud Scheduler** instead of a long-running crawler container: you pay only for
  the nightly hour, retries and timeouts are handled by the platform, and logs are per execution.
* **Secrets via Secret Manager env mounts**, never in build substitutions.

---

## 3. Frontend pipeline (Vercel)

1. In Vercel, **Add New → Project**, import the GitHub repository.
2. Set **Root Directory** to `frontend`. Framework preset: Next.js (auto-detected). Build command
   `pnpm build`, install command `pnpm install --frozen-lockfile` (defaults work).
3. Environment variables (Production and Preview):

   | Name | Value |
   |------|-------|
   | `API_URL` | `https://watch-compare-api-xxxxx-de.a.run.app` (Cloud Run URL; server-only) |
   | `NEXT_PUBLIC_SITE_URL` | `https://watch-compare.vercel.app` (or your domain) |

   Preview deployments can point at the same production API (it is read-only) or at a staging
   Cloud Run service.
4. Push to `main` → production deployment. Every PR → preview URL with its own comment on GitHub.
5. Add the production domain and the `*.vercel.app` preview origin(s) to `_CORS_ORIGINS` on the
   Cloud Build trigger (the API only serves the configured origins in the browser; server-side
   RSC fetches are unaffected by CORS).

`frontend/vercel.json` pins the function region to `hnd1` (Tokyo) so RSC → Cloud Run (asia-east1)
round-trips stay short; change both if you deploy the backend elsewhere.

### Using the Vercel CLI instead of Git integration

```bash
cd frontend
vercel link                    # once
vercel env pull .env.local     # sync env vars
vercel                          # preview
vercel --prod                   # production
```

---

## 4. GitHub Actions (`.github/workflows/ci.yml`)

Runs on every PR and push:

* `backend`: `go vet`, `go test -race`, `gofmt` check.
* `frontend`: `pnpm install --frozen-lockfile`, `eslint`, `tsc --noEmit`, `next build`
  (with a dummy `API_URL`; pages degrade gracefully when the API is unreachable).

Deployments are *not* done from GitHub Actions — Vercel and Cloud Build own them — so no cloud
credentials live in GitHub secrets.

---

## 5. Observability & cost

* Structured JSON logs with Cloud Logging severities (`zerolog`), request IDs, latency.
* `/readyz` pings the database; Cloud Run uses it as the startup probe.
* `GET /api/v1/crawls` shows the last 50 runs with counts and errors — wire it to an alert
  (e.g. Cloud Monitoring log-based alert on `severity=ERROR AND job="crawler"`).
* Typical monthly cost at hobby scale: Cloud SQL `db-g1-small` ≈ $25–30, Cloud Run < $5,
  Artifact Registry < $1, Vercel Hobby $0.
