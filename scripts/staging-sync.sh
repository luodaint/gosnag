#!/usr/bin/env bash
set -euo pipefail

# Sync production database to staging
#
# Usage:
#   ./scripts/staging-sync.sh                  # full sync (schema + data)
#   ./scripts/staging-sync.sh --schema-only    # schema only, no data
#   ./scripts/staging-sync.sh --dump-only      # just create the dump file
#
# Prerequisites:
#   - psql and pg_dump installed (brew install postgresql@16)
#   - Production DB accessible (VPN/SSM tunnel if needed)
#   - Staging docker-compose running: docker compose -f docker-compose.staging.yml up -d

DUMP_FILE="/tmp/gosnag-prod-dump.sql"

# Production DB (RDS via gosnag schema in outline DB)
PROD_HOST="${PROD_DB_HOST:-internal-apps.cluster-cueyann8pjad.eu-west-1.rds.amazonaws.com}"
PROD_PORT="${PROD_DB_PORT:-5432}"
PROD_USER="${PROD_DB_USER:-outliner}"
PROD_DB="${PROD_DB_NAME:-outline}"
PROD_SCHEMA="${PROD_DB_SCHEMA:-gosnag}"

# Staging DB (local docker)
STAGING_HOST="${STAGING_DB_HOST:-localhost}"
STAGING_PORT="${STAGING_DB_PORT:-5433}"
STAGING_USER="${STAGING_DB_USER:-gosnag}"
STAGING_DB="${STAGING_DB_NAME:-gosnag}"
STAGING_PASS="${STAGING_DB_PASS:-gosnag}"

SCHEMA_ONLY=false
DUMP_ONLY=false

for arg in "$@"; do
  case "$arg" in
    --schema-only) SCHEMA_ONLY=true ;;
    --dump-only)   DUMP_ONLY=true ;;
  esac
done

echo "==> Dumping production database..."
echo "    Host: $PROD_HOST"
echo "    Schema: $PROD_SCHEMA"

DUMP_ARGS=(-h "$PROD_HOST" -p "$PROD_PORT" -U "$PROD_USER" -d "$PROD_DB" -n "$PROD_SCHEMA" --no-owner --no-privileges)

if [ "$SCHEMA_ONLY" = true ]; then
  DUMP_ARGS+=(--schema-only)
  echo "    Mode: schema only"
else
  echo "    Mode: full (schema + data)"
fi

pg_dump "${DUMP_ARGS[@]}" > "$DUMP_FILE"
echo "    Dump saved to: $DUMP_FILE ($(du -h "$DUMP_FILE" | cut -f1))"

if [ "$DUMP_ONLY" = true ]; then
  echo "==> Done (dump only)."
  exit 0
fi

echo ""
echo "==> Importing into staging..."
echo "    Host: $STAGING_HOST:$STAGING_PORT"
echo "    Database: $STAGING_DB"

export PGPASSWORD="$STAGING_PASS"

# Drop and recreate all tables (clean import)
echo "    Dropping existing schema..."
psql -h "$STAGING_HOST" -p "$STAGING_PORT" -U "$STAGING_USER" -d "$STAGING_DB" -q <<SQL
-- Create the gosnag schema if it doesn't exist, then drop everything in it
CREATE SCHEMA IF NOT EXISTS $PROD_SCHEMA;
DROP SCHEMA $PROD_SCHEMA CASCADE;
CREATE SCHEMA $PROD_SCHEMA;
SQL

echo "    Restoring dump..."
psql -h "$STAGING_HOST" -p "$STAGING_PORT" -U "$STAGING_USER" -d "$STAGING_DB" -q -f "$DUMP_FILE"

# Set search_path so the app finds the tables
psql -h "$STAGING_HOST" -p "$STAGING_PORT" -U "$STAGING_USER" -d "$STAGING_DB" -q <<SQL
ALTER DATABASE $STAGING_DB SET search_path TO $PROD_SCHEMA, public;
SQL

echo ""
echo "==> Verifying..."
TABLE_COUNT=$(psql -h "$STAGING_HOST" -p "$STAGING_PORT" -U "$STAGING_USER" -d "$STAGING_DB" -tAc \
  "SELECT count(*) FROM information_schema.tables WHERE table_schema = '$PROD_SCHEMA' AND table_type = 'BASE TABLE'")
echo "    Tables: $TABLE_COUNT"

if [ "$SCHEMA_ONLY" = false ]; then
  PROJECT_COUNT=$(psql -h "$STAGING_HOST" -p "$STAGING_PORT" -U "$STAGING_USER" -d "$STAGING_DB" -tAc \
    "SELECT count(*) FROM $PROD_SCHEMA.projects" 2>/dev/null || echo "0")
  ISSUE_COUNT=$(psql -h "$STAGING_HOST" -p "$STAGING_PORT" -U "$STAGING_USER" -d "$STAGING_DB" -tAc \
    "SELECT count(*) FROM $PROD_SCHEMA.issues" 2>/dev/null || echo "0")
  echo "    Projects: $PROJECT_COUNT"
  echo "    Issues: $ISSUE_COUNT"
fi

unset PGPASSWORD

echo ""
echo "==> Done! Staging available at http://localhost:8081"
