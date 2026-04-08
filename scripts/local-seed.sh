#!/usr/bin/env bash
set -euo pipefail

DSN="${1:-}"
COUNT="${2:-200}"

if [ -z "$DSN" ]; then
  echo "Usage: $0 <DSN> [count]"
  echo "  DSN:   project DSN from settings (e.g. http://key@localhost:8099/5)"
  echo "  count: number of events to send (default: 200)"
  exit 1
fi

# Parse DSN: http://PUBLIC_KEY@HOST/PROJECT_ID
PROTO=$(echo "$DSN" | sed -E 's|^(https?)://.*|\1|')
KEY=$(echo "$DSN" | sed -E 's|^https?://([^@]+)@.*|\1|')
HOST=$(echo "$DSN" | sed -E 's|^https?://[^@]+@([^/]+).*|\1|')
PROJECT_ID=$(echo "$DSN" | sed -E 's|.*/([^/]+)$|\1|')
BASE="${PROTO}://${HOST}"

echo "Seeding ${COUNT} events to project ${PROJECT_ID} at ${BASE}"
echo "Public key: ${KEY}"

# Error types to pick from
ERRORS=(
  "TypeError:Cannot read properties of undefined (reading 'map')"
  "ReferenceError:process is not defined"
  "SyntaxError:Unexpected token < in JSON at position 0"
  "Error:ECONNREFUSED 127.0.0.1:5432"
  "TypeError:fetch failed"
  "RangeError:Maximum call stack size exceeded"
  "Error:Request failed with status code 500"
  "TypeError:Cannot convert undefined or null to object"
  "Error:ETIMEDOUT"
  "Error:connect ECONNRESET"
  "ValueError:invalid literal for int() with base 10"
  "KeyError:'user_id'"
  "AttributeError:'NoneType' object has no attribute 'get'"
  "RuntimeError:dictionary changed size during iteration"
  "IOError:No such file or directory: '/tmp/cache.json'"
  "PermissionError:[Errno 13] Permission denied: '/var/log/app.log'"
  "ZeroDivisionError:division by zero"
  "IndexError:list index out of range"
  "ConnectionError:Connection refused"
  "TimeoutError:The read operation timed out"
)

LEVELS=("error" "error" "error" "error" "warning" "fatal")

URLS=(
  "/api/v1/users"
  "/api/v1/orders"
  "/api/v1/products"
  "/api/v1/payments"
  "/api/v1/auth/login"
  "/api/v1/webhooks"
  "/api/v1/reports"
  "/api/v1/notifications"
  "/dashboard"
  "/settings"
)

METHODS=("GET" "GET" "GET" "POST" "POST" "PUT" "DELETE")

PLATFORMS=("javascript" "python" "node" "go")

RELEASES=("1.0.0" "1.0.1" "1.1.0" "1.2.0" "2.0.0-beta")

ENVS=("production" "staging" "development")

SERVERS=("web-01" "web-02" "worker-01" "api-01" "api-02")

OK=0
FAIL=0

for i in $(seq 1 "$COUNT"); do
  ERR="${ERRORS[$((RANDOM % ${#ERRORS[@]}))]}"
  ERR_TYPE="${ERR%%:*}"
  ERR_MSG="${ERR#*:}"
  LEVEL="${LEVELS[$((RANDOM % ${#LEVELS[@]}))]}"
  URL="${URLS[$((RANDOM % ${#URLS[@]}))]}"
  METHOD="${METHODS[$((RANDOM % ${#METHODS[@]}))]}"
  PLATFORM="${PLATFORMS[$((RANDOM % ${#PLATFORMS[@]}))]}"
  RELEASE="${RELEASES[$((RANDOM % ${#RELEASES[@]}))]}"
  ENV="${ENVS[$((RANDOM % ${#ENVS[@]}))]}"
  SERVER="${SERVERS[$((RANDOM % ${#SERVERS[@]}))]}"

  # Vary timestamp across last 7 days
  OFFSET=$((RANDOM % 604800))
  TS=$(date -u -v-${OFFSET}S +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "-${OFFSET} seconds" +"%Y-%m-%dT%H:%M:%SZ")

  PAYLOAD=$(cat <<EOJSON
{
  "event_id": "$(uuidgen | tr '[:upper:]' '[:lower:]' | tr -d '-')",
  "timestamp": "${TS}",
  "level": "${LEVEL}",
  "platform": "${PLATFORM}",
  "release": "${RELEASE}",
  "environment": "${ENV}",
  "server_name": "${SERVER}",
  "transaction": "${METHOD} ${URL}",
  "message": "${ERR_MSG}",
  "exception": {
    "values": [{
      "type": "${ERR_TYPE}",
      "value": "${ERR_MSG}",
      "stacktrace": {
        "frames": [
          {"filename": "app/handlers/${URL##*/}.js", "lineno": $((RANDOM % 500 + 1)), "function": "handle${URL##*/^}"},
          {"filename": "app/middleware/auth.js", "lineno": $((RANDOM % 200 + 1)), "function": "authenticate"},
          {"filename": "node_modules/express/lib/router.js", "lineno": $((RANDOM % 300 + 1)), "function": "processParams"}
        ]
      }
    }]
  },
  "request": {
    "method": "${METHOD}",
    "url": "https://app.example.com${URL}",
    "headers": {"User-Agent": "Mozilla/5.0", "Content-Type": "application/json"}
  },
  "tags": {"browser": "Chrome", "os": "macOS"},
  "user": {"id": "user-$((RANDOM % 50))", "email": "user$((RANDOM % 50))@example.com"}
}
EOJSON
)

  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${BASE}/api/${PROJECT_ID}/store/" \
    -H "Content-Type: application/json" \
    -H "X-Sentry-Auth: Sentry sentry_key=${KEY}" \
    -d "$PAYLOAD")

  if [ "$STATUS" = "200" ]; then
    OK=$((OK + 1))
  else
    FAIL=$((FAIL + 1))
  fi

  # Progress every 20
  if [ $((i % 20)) -eq 0 ]; then
    echo "  ${i}/${COUNT} sent (ok: ${OK}, fail: ${FAIL})"
  fi
done

echo ""
echo "Done: ${OK} ok, ${FAIL} failed out of ${COUNT}"
