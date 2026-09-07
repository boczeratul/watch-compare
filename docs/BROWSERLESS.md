# Rendering Chrono24 through Browserless

Chrono24 answers plain HTTP clients with 403, so its adapter fetches pages through a headless
browser service. The crawler speaks the **Browserless v2 REST API** (`POST /content`), which is
offered both as a hosted service (browserless.io) and as a container you can run on Cloud Run.

Review Chrono24's terms of use before enabling this. A licensed data feed is the proper
production path; rendering is for evaluation and low volumes.

## Option A — browserless.io (hosted)

1. Get an API key from https://account.browserless.io and pick the region closest to
   Chrono24's edge (EU is usually best): `https://production-lon.browserless.io`,
   `https://production-ams.browserless.io` or `https://production-sfo.browserless.io`.

2. Try it from your machine first. This is the exact request the crawler sends:

   ```bash
   TOKEN=your-api-key
   curl -s -X POST "https://production-lon.browserless.io/content?token=$TOKEN" \
     -H 'Content-Type: application/json' \
     -d '{"url":"https://www.chrono24.com/rolex/index.htm?pageSize=120&showpage=1&sortorder=5","gotoOptions":{"waitUntil":"networkidle2","timeout":60000},"rejectResourceTypes":["image","media","font"]}' \
     -o chrono24.html
   grep -c -- '--id' chrono24.html      # > 0 means listing cards are present
   grep -ci 'access denied\|captcha\|are you a robot' chrono24.html   # > 0 means still blocked
   ```

   If the page is blocked, add browserless.io's residential proxy / stealth options to the query
   string (billed extra): `&proxy=residential&proxyCountry=de&stealth=true`. The crawler passes the
   same options through `CRAWL_RENDER_EXTRA_QUERY`.

3. Run the crawler locally as a dry run against the real site:

   ```bash
   cd backend
   export CRAWL_RENDER_SERVICE_URL=https://production-lon.browserless.io
   export CRAWL_RENDER_SERVICE_TOKEN=$TOKEN
   export CRAWL_RENDER_EXTRA_QUERY='proxy=residential&proxyCountry=de'   # only if needed
   make crawl-dry ARGS="-sources=chrono24"
   ```

   The dry run prints the first 25 parsed listings. If it prints none but step 2 returned cards,
   Chrono24's markup differs from the selectors in `internal/crawler/chrono24` — save the file from
   step 2 as `internal/crawler/chrono24/testdata/list.html` and adjust `ParseList` against it.

4. Deploy: store the key in Secret Manager and set the URL on the crawler job.

   ```bash
   printf '%s' "$TOKEN" | gcloud secrets create BROWSERLESS_TOKEN --replication-policy=automatic --data-file=- \
     || printf '%s' "$TOKEN" | gcloud secrets versions add BROWSERLESS_TOKEN --data-file=-

   # immediate effect on the existing job …
   gcloud run jobs update watch-compare-crawler --region asia-east1 \
     --update-secrets=CRAWL_RENDER_SERVICE_TOKEN=BROWSERLESS_TOKEN:latest \
     --update-env-vars=CRAWL_RENDER_SERVICE_URL=https://production-lon.browserless.io,^@^CRAWL_RENDER_EXTRA_QUERY=proxy=residential&proxyCountry=de

   # … and permanently via the Cloud Build trigger substitutions, so redeploys keep it
   gcloud builds triggers update watch-compare-backend \
     --update-substitutions=_RENDER_SERVICE_URL=https://production-lon.browserless.io,_RENDER_EXTRA_QUERY='proxy=residential&proxyCountry=de'
   ```

   `cloudbuild.yaml` already mounts `BROWSERLESS_TOKEN` as `CRAWL_RENDER_SERVICE_TOKEN`; the secret
   must exist (the bootstrap script creates a placeholder) or the job deploy fails.

5. Execute the job once and check `GET /api/v1/crawls`: chrono24 should move from `skipped` to
   `ok` with a non-zero `listingsSeen`.

### Budget

Each Chrono24 list page is one rendered page (one Browserless unit, plus proxy bandwidth if
enabled) and carries 120 listings sorted newest-first. The nightly budget is bounded by two
settings on the crawler job:

| Variable | Default | Effect |
|----------|---------|--------|
| `CHRONO24_BRANDS` | `rolex,omega,iwc,audemarspiguet,patekphilippe` | Brands crawled (Chrono24 slugs; our own slugs such as `patek-philippe` are accepted too) |
| `CHRONO24_MAX_PAGES` | `3` | Pages per brand per night (never more than `CRAWL_MAX_PAGES`) |

The defaults are 5 brands × 3 pages = **15 renders a night, about 450 a month**, which fits in
browserless.io's free tier and still refreshes the newest 360 listings per brand daily. Raise
`CHRONO24_MAX_PAGES` once you know the render hit rate; every extra page per brand adds 150
renders a month.

## Option B — self-hosted Browserless on Cloud Run

Cheaper at volume and keeps traffic inside your project, but Chrono24 sees a Google Cloud IP,
which is more likely to be blocked than a residential proxy; you can point Browserless at a proxy
with `CRAWL_RENDER_LAUNCH_JSON='{"stealth":true,"args":["--proxy-server=http://host:port"]}'`.

```bash
PROJECT_ID=my-proj REGION=asia-east1
RUNTIME_SA=watch-compare-runtime@$PROJECT_ID.iam.gserviceaccount.com
TOKEN=$(openssl rand -hex 24)

# 1. Cloud Run cannot pull from ghcr.io directly: mirror it through an Artifact Registry remote repo
gcloud artifacts repositories create ghcr --repository-format=docker --location=$REGION \
  --mode=remote-repository --remote-docker-repo=https://ghcr.io
IMAGE=$REGION-docker.pkg.dev/$PROJECT_ID/ghcr/browserless/chromium:latest

# 2. Deploy privately; Chromium needs real CPU and memory
gcloud run deploy browserless --image=$IMAGE --region=$REGION \
  --port=3000 --cpu=2 --memory=4Gi --no-cpu-throttling --execution-environment=gen2 \
  --concurrency=5 --timeout=120 --min-instances=0 --max-instances=2 \
  --set-env-vars=TOKEN=$TOKEN,MAX_CONCURRENT_SESSIONS=5,TIMEOUT=60000,QUEUED=10 \
  --no-allow-unauthenticated

# 3. Only the crawler's service account may call it
gcloud run services add-iam-policy-binding browserless --region=$REGION \
  --member=serviceAccount:$RUNTIME_SA --role=roles/run.invoker

# 4. Store the token and point the crawler at the service
URL=$(gcloud run services describe browserless --region=$REGION --format='value(status.url)')
printf '%s' "$TOKEN" | gcloud secrets create BROWSERLESS_TOKEN --replication-policy=automatic --data-file=-
gcloud run jobs update watch-compare-crawler --region=$REGION \
  --update-secrets=CRAWL_RENDER_SERVICE_TOKEN=BROWSERLESS_TOKEN:latest \
  --update-env-vars=CRAWL_RENDER_SERVICE_URL=$URL
```

Because the URL ends in `.run.app`, the crawler automatically attaches a Google identity token
(from the Cloud Run metadata server) in addition to `?token=`, which is what makes the private
service reachable. Test from your workstation with your own identity:

```bash
curl -s -X POST "$URL/content?token=$TOKEN" \
  -H "Authorization: Bearer $(gcloud auth print-identity-token)" \
  -H 'Content-Type: application/json' -d '{"url":"https://example.com"}' | head -c 300
```

Cost note: a 2 vCPU / 4 GiB instance only bills while rendering; with `min-instances=0` a nightly
crawl of a few hundred pages costs cents. Set `--max-instances` to bound the worst case.

## Settings reference

| Variable | Meaning |
|----------|---------|
| `CRAWL_RENDER_SERVICE_URL` | Base URL of the Browserless service. Empty = Chrono24 is skipped. |
| `CRAWL_RENDER_SERVICE_TOKEN` | Browserless API key / container `TOKEN`, sent as `?token=`. |
| `CRAWL_RENDER_EXTRA_QUERY` | Extra query options, e.g. `proxy=residential&proxyCountry=de&stealth=true` (browserless.io). |
| `CRAWL_RENDER_LAUNCH_JSON` | Chromium launch options for self-hosted Browserless, e.g. `{"stealth":true,"args":["--proxy-server=…"]}`. |
| `CRAWL_RENDER_MODE` | `browserless` (default) or `get` for a custom endpoint answering `GET ?url=`. |
| `CRAWL_RENDER_USE_IDTOKEN` | Attach a Google identity token; defaults to true for `*.run.app` URLs. |
| `CRAWL_RATE_LIMIT_MS` | Applied per *target* host, so rendered fetches are as polite as plain ones. |
