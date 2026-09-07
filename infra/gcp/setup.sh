#!/usr/bin/env bash
# One-time GCP bootstrap for WatchCompare. Idempotent where gcloud allows it.
# Usage: PROJECT_ID=watch-compare-507905 GITHUB_OWNER=boczeratul GITHUB_REPO=watch-compare ./infra/gcp/setup.sh
set -euo pipefail

: "${PROJECT_ID:?set PROJECT_ID}"
: "${GITHUB_OWNER:?set GITHUB_OWNER}"
: "${GITHUB_REPO:?set GITHUB_REPO}"
REGION="${REGION:-asia-east1}"
AR_REPO="${AR_REPO:-watch-compare}"
SQL_INSTANCE="${SQL_INSTANCE:-watch-compare-pg}"
SQL_TIER="${SQL_TIER:-db-perf-optimized-N-2}"
API_SERVICE="${API_SERVICE:-watch-compare-api}"
CRAWLER_JOB="${CRAWLER_JOB:-watch-compare-crawler}"
CORS_ORIGINS="${CORS_ORIGINS:-https://watch-compare.vercel.app}"

PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')
RUNTIME_SA="watch-compare-runtime@${PROJECT_ID}.iam.gserviceaccount.com"
BUILD_SA="watch-compare-cloudbuild@${PROJECT_ID}.iam.gserviceaccount.com"
SCHED_SA="watch-compare-scheduler@${PROJECT_ID}.iam.gserviceaccount.com"

gcloud config set project "$PROJECT_ID" >/dev/null

echo "▶ Enabling APIs"
gcloud services enable run.googleapis.com cloudbuild.googleapis.com artifactregistry.googleapis.com \
  sqladmin.googleapis.com secretmanager.googleapis.com cloudscheduler.googleapis.com iam.googleapis.com \
  cloudresourcemanager.googleapis.com logging.googleapis.com

echo "▶ Artifact Registry"
gcloud artifacts repositories describe "$AR_REPO" --location="$REGION" >/dev/null 2>&1 || \
  gcloud artifacts repositories create "$AR_REPO" --repository-format=docker --location="$REGION" \
    --description="WatchCompare backend images"

echo "▶ Cloud SQL (PostgreSQL 16)"
if ! gcloud sql instances describe "$SQL_INSTANCE" >/dev/null 2>&1; then
  gcloud sql instances create "$SQL_INSTANCE" --database-version=POSTGRES_16 --tier="$SQL_TIER" \
    --region="$REGION" --storage-auto-increase --backup-start-time=18:00 --availability-type=zonal \
    --database-flags=max_connections=100
fi
gcloud sql databases describe watch --instance="$SQL_INSTANCE" >/dev/null 2>&1 || \
  gcloud sql databases create watch --instance="$SQL_INSTANCE"
# URL-safe password (no / + = so it can be embedded in the DSN without encoding).
DB_PASSWORD="${DB_PASSWORD:-$(openssl rand -base64 24 | tr -d '/+=')}"
if gcloud sql users list --instance="$SQL_INSTANCE" --format='value(name)' | grep -qx watch; then
  # Re-run: always (re)apply the password so the user and the secret below cannot drift apart.
  gcloud sql users set-password watch --instance="$SQL_INSTANCE" --password="$DB_PASSWORD"
else
  gcloud sql users create watch --instance="$SQL_INSTANCE" --password="$DB_PASSWORD"
fi
CONNECTION_NAME=$(gcloud sql instances describe "$SQL_INSTANCE" --format='value(connectionName)')

echo "▶ Service accounts"
for sa in watch-compare-runtime watch-compare-cloudbuild watch-compare-scheduler; do
  gcloud iam service-accounts describe "${sa}@${PROJECT_ID}.iam.gserviceaccount.com" >/dev/null 2>&1 || \
    gcloud iam service-accounts create "$sa" --display-name="$sa"
done
for role in roles/cloudsql.client roles/secretmanager.secretAccessor roles/logging.logWriter; do
  gcloud projects add-iam-policy-binding "$PROJECT_ID" --member="serviceAccount:$RUNTIME_SA" --role="$role" --quiet >/dev/null
done
for role in roles/run.admin roles/artifactregistry.writer roles/logging.logWriter roles/cloudsql.client roles/secretmanager.secretAccessor; do
  gcloud projects add-iam-policy-binding "$PROJECT_ID" --member="serviceAccount:$BUILD_SA" --role="$role" --quiet >/dev/null
done
gcloud iam service-accounts add-iam-policy-binding "$RUNTIME_SA" --member="serviceAccount:$BUILD_SA" --role=roles/iam.serviceAccountUser --quiet >/dev/null

echo "▶ Secrets"
upsert_secret() { # name value  (printf '%s': no trailing newline, which would break the password)
  if gcloud secrets describe "$1" >/dev/null 2>&1; then
    printf '%s' "$2" | gcloud secrets versions add "$1" --data-file=- >/dev/null
  else
    printf '%s' "$2" | gcloud secrets create "$1" --replication-policy=automatic --data-file=- >/dev/null
  fi
}
DATABASE_URL_VALUE="postgres://watch:${DB_PASSWORD}@/watch?host=/cloudsql/${CONNECTION_NAME}&sslmode=disable"
upsert_secret DATABASE_URL "$DATABASE_URL_VALUE"
echo "   DATABASE_URL secret updated (password applied to Cloud SQL user 'watch' in the same run)."
gcloud secrets describe EBAY_CLIENT_ID >/dev/null 2>&1 || upsert_secret EBAY_CLIENT_ID "replace-me"
gcloud secrets describe EBAY_CLIENT_SECRET >/dev/null 2>&1 || upsert_secret EBAY_CLIENT_SECRET "replace-me"
gcloud secrets describe BROWSERLESS_TOKEN >/dev/null 2>&1 || upsert_secret BROWSERLESS_TOKEN "replace-me"

echo "▶ Cloud Build trigger (connect the GitHub repo in the console first if prompted)"
gcloud builds triggers describe watch-compare-backend >/dev/null 2>&1 || \
  gcloud builds triggers create github --name=watch-compare-backend \
    --repo-owner="$GITHUB_OWNER" --repo-name="$GITHUB_REPO" --branch-pattern='^main$' \
    --included-files='backend/**' --build-config=backend/cloudbuild.yaml \
    --service-account="projects/${PROJECT_ID}/serviceAccounts/${BUILD_SA}" \
    --substitutions="_REGION=${REGION},_AR_REPO=${AR_REPO},_API_SERVICE=${API_SERVICE},_CRAWLER_JOB=${CRAWLER_JOB},_CLOUDSQL=${CONNECTION_NAME},_RUNTIME_SA=${RUNTIME_SA},_CORS_ORIGINS=${CORS_ORIGINS}"

echo "▶ Cloud Scheduler: nightly crawl at 02:00 Asia/Taipei"
JOB_RUN_URL="https://${REGION}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${PROJECT_ID}/jobs/${CRAWLER_JOB}:run"
if gcloud scheduler jobs describe watch-compare-nightly-crawl --location="$REGION" >/dev/null 2>&1; then
  gcloud scheduler jobs update http watch-compare-nightly-crawl --location="$REGION" \
    --schedule="0 2 * * *" --time-zone="Asia/Taipei" --uri="$JOB_RUN_URL" --http-method=POST \
    --oauth-service-account-email="$SCHED_SA" --attempt-deadline=30m
else
  gcloud scheduler jobs create http watch-compare-nightly-crawl --location="$REGION" \
    --schedule="0 2 * * *" --time-zone="Asia/Taipei" --uri="$JOB_RUN_URL" --http-method=POST \
    --oauth-service-account-email="$SCHED_SA" --attempt-deadline=30m
fi
# The scheduler SA needs run.invoker on the job; the job exists only after the first Cloud Build run.
cat <<MSG

Done. Next steps:
  1. Push to main (or run the trigger) to build & deploy: gcloud builds triggers run watch-compare-backend --branch=main
  2. After the first deploy, grant the scheduler permission to run the job:
       gcloud run jobs add-iam-policy-binding ${CRAWLER_JOB} --region=${REGION} \\
         --member=serviceAccount:${SCHED_SA} --role=roles/run.invoker
  3. Put real eBay keys into Secret Manager (EBAY_CLIENT_ID / EBAY_CLIENT_SECRET).
  4. Copy the API URL into Vercel as API_URL:
       gcloud run services describe ${API_SERVICE} --region=${REGION} --format='value(status.url)'
MSG
