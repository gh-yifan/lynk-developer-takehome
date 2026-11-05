#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
GO_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
cd "$GO_DIR"
RESULTS_DIR=${RESULTS_DIR:-$SCRIPT_DIR/results}
API_ADDR=${API_ADDR:-127.0.0.1:8080}
case "$API_ADDR" in
    http://*|https://*) API_URL=$API_ADDR ;;
    :*) API_URL="http://127.0.0.1$API_ADDR" ;;
    *) API_URL="http://$API_ADDR" ;;
esac
mkdir -p "$RESULTS_DIR"

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 QUEUE_SIZE" >&2
    echo "Start the implementation server separately; this script only sends k6 requests." >&2
    exit 2
fi

QUEUE_SIZE=$1
case "$QUEUE_SIZE" in
    ''|*[!0-9]*) echo "Queue size must be a non-negative integer: $QUEUE_SIZE" >&2; exit 2 ;;
esac

if ! command -v k6 >/dev/null 2>&1; then
    echo "k6 is required. Install it from https://grafana.com/docs/k6/latest/set-up/install-k6/" >&2
    exit 1
fi
if ! command -v curl >/dev/null 2>&1; then
    echo "curl is required to check API readiness." >&2
    exit 1
fi

ready=false
attempt=0
while [ "$attempt" -lt 30 ]; do
    if curl --silent --fail "$API_URL/commands" >/dev/null; then
        ready=true
        break
    fi
    attempt=$((attempt + 1))
    sleep 1
done
if [ "$ready" != true ]; then
    echo "No API responded at $API_URL/commands; start the selected implementation first." >&2
    exit 1
fi

echo "Benchmarking existing API at $API_URL with baseline queue size $QUEUE_SIZE"
k6 run \
    -e "BASE_URL=$API_URL" \
    -e "QUEUE_SIZE=$QUEUE_SIZE" \
    -e "K6_PROFILE=${K6_PROFILE:-throughput}" \
    -e "THROUGHPUT_VUS=${THROUGHPUT_VUS:-20}" \
    -e "THROUGHPUT_DURATION=${THROUGHPUT_DURATION:-60s}" \
    --out "json=$RESULTS_DIR/results-$QUEUE_SIZE.ndjson" \
    --summary-export "$RESULTS_DIR/summary-$QUEUE_SIZE.json" \
    "$SCRIPT_DIR/commands.js"

echo "k6 samples: $RESULTS_DIR/results-$QUEUE_SIZE.ndjson"
echo "k6 summary: $RESULTS_DIR/summary-$QUEUE_SIZE.json"
