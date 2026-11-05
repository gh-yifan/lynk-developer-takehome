#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
GO_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
TARGET_GO_DIR=${TARGET_GO_DIR:-}
if [ -z "$TARGET_GO_DIR" ]; then
    echo "Set TARGET_GO_DIR to the implementation worktree's go/ directory." >&2
    exit 2
fi
case "$TARGET_GO_DIR" in
    /*) ;;
    *) TARGET_GO_DIR="$GO_DIR/$TARGET_GO_DIR" ;;
esac
TARGET_GO_DIR=$(CDPATH= cd -- "$TARGET_GO_DIR" && pwd)

QUEUE_SIZE=${1:-10000}
LABEL=${2:-$(git -C "$TARGET_GO_DIR" branch --show-current | tr '/ ' '--')}
OUTPUT_ROOT=${3:-$SCRIPT_DIR/profiles}
PPROF_ADDR=${PPROF_ADDR:-127.0.0.1:6060}
API_ADDR=${API_ADDR:-127.0.0.1:8080}
CPU_SECONDS=${CPU_SECONDS:-60}
API_URL="http://$API_ADDR"
PPROF_URL="http://$PPROF_ADDR"
OVERLAY_SOURCE="$TARGET_GO_DIR/zz_perf_pprof.go"

case "$QUEUE_SIZE" in
    ''|*[!0-9]*) echo "Queue size must be a non-negative integer: $QUEUE_SIZE" >&2; exit 2 ;;
esac
case "$LABEL" in
    ''|*[!A-Za-z0-9_.-]*) echo "Label may contain only letters, numbers, dot, underscore, and dash: $LABEL" >&2; exit 2 ;;
esac
if [ -e "$OVERLAY_SOURCE" ]; then
    echo "Refusing to overlay existing file: $OVERLAY_SOURCE" >&2
    exit 1
fi
if ! command -v k6 >/dev/null 2>&1 || ! command -v curl >/dev/null 2>&1; then
    echo "k6 and curl are required." >&2
    exit 1
fi

STAMP=$(date +%Y%m%d-%H%M%S)
OUT_DIR="$OUTPUT_ROOT/$LABEL-$QUEUE_SIZE-$STAMP"
mkdir -p "$OUT_DIR"
TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/command-queue-pprof.XXXXXX")
SERVER_PID=
K6_PID=
CPU_PID=

cleanup() {
    if [ -n "$CPU_PID" ]; then kill "$CPU_PID" 2>/dev/null || true; wait "$CPU_PID" 2>/dev/null || true; fi
    if [ -n "$K6_PID" ]; then kill "$K6_PID" 2>/dev/null || true; wait "$K6_PID" 2>/dev/null || true; fi
    if [ -n "$SERVER_PID" ]; then kill "$SERVER_PID" 2>/dev/null || true; wait "$SERVER_PID" 2>/dev/null || true; fi
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

printf '{"Replace":{"%s":"%s"}}\n' "$OVERLAY_SOURCE" "$SCRIPT_DIR/pprof_init.go" >"$TMP_DIR/overlay.json"
(
    cd "$TARGET_GO_DIR"
    go build -overlay="$TMP_DIR/overlay.json" -o "$OUT_DIR/command-queue-api" .
)
PPROF_ADDR="$PPROF_ADDR" PORT=${PORT:-8080} "$OUT_DIR/command-queue-api" >"$OUT_DIR/server.log" 2>&1 &
SERVER_PID=$!

ready=false
attempt=0
while [ "$attempt" -lt 30 ]; do
    if ! kill -0 "$SERVER_PID" 2>/dev/null; then
        cat "$OUT_DIR/server.log" >&2
        echo "Profiled API exited before becoming ready." >&2
        exit 1
    fi
    if curl --silent --fail "$API_URL/commands" >/dev/null \
        && curl --silent --fail "$PPROF_URL/debug/pprof/" >/dev/null; then
        ready=true
        break
    fi
    attempt=$((attempt + 1))
    sleep 1
done
if [ "$ready" != true ]; then
    cat "$OUT_DIR/server.log" >&2
    echo "API or temporary pprof listener not ready; check API_ADDR=$API_ADDR and PPROF_ADDR=$PPROF_ADDR." >&2
    exit 1
fi

echo "Profiling $(git -C "$TARGET_GO_DIR" branch --show-current) at queue size $QUEUE_SIZE"
echo "Artifacts: $OUT_DIR"
curl --silent --show-error --fail "$PPROF_URL/debug/pprof/profile?seconds=$CPU_SECONDS" -o "$OUT_DIR/cpu.pprof" &
CPU_PID=$!
k6 run \
    -e "BASE_URL=$API_URL" \
    -e "QUEUE_SIZE=$QUEUE_SIZE" \
    -e "K6_PROFILE=${K6_PROFILE:-profile}" \
    -e "THROUGHPUT_VUS=${THROUGHPUT_VUS:-20}" \
    -e "THROUGHPUT_DURATION=${THROUGHPUT_DURATION:-60s}" \
    --out "json=$OUT_DIR/k6-results.ndjson" \
    --summary-export "$OUT_DIR/k6-summary.json" \
    "$SCRIPT_DIR/commands.js" >"$OUT_DIR/k6-summary.txt" 2>&1 &
K6_PID=$!
set +e
wait "$K6_PID"
K6_STATUS=$?
set -e
K6_PID=
cat "$OUT_DIR/k6-summary.txt"
wait "$CPU_PID"
CPU_PID=

curl --silent --show-error --fail "$PPROF_URL/debug/pprof/heap?gc=1" -o "$OUT_DIR/heap.pprof"
curl --silent --show-error --fail "$PPROF_URL/debug/pprof/allocs" -o "$OUT_DIR/allocs.pprof"
go tool pprof -top "$OUT_DIR/command-queue-api" "$OUT_DIR/cpu.pprof" >"$OUT_DIR/cpu-top.txt"
go tool pprof -top -sample_index=inuse_space "$OUT_DIR/command-queue-api" "$OUT_DIR/heap.pprof" >"$OUT_DIR/inuse-top.txt"
go tool pprof -top -sample_index=alloc_space "$OUT_DIR/command-queue-api" "$OUT_DIR/allocs.pprof" >"$OUT_DIR/alloc-space-top.txt"
go tool pprof -top -nodecount=1 -unit=bytes -sample_index=inuse_space "$OUT_DIR/command-queue-api" "$OUT_DIR/heap.pprof" >"$OUT_DIR/inuse-total.txt"
go tool pprof -top -nodecount=1 -unit=bytes -sample_index=alloc_space "$OUT_DIR/command-queue-api" "$OUT_DIR/allocs.pprof" >"$OUT_DIR/alloc-total.txt"
INUSE_BYTES=$(awk '/ total$/ { for (i = 1; i <= NF; i++) if ($i == "total") { value = $(i - 1); sub(/B$/, "", value); print value; exit } }' "$OUT_DIR/inuse-total.txt")
ALLOC_BYTES=$(awk '/ total$/ { for (i = 1; i <= NF; i++) if ($i == "total") { value = $(i - 1); sub(/B$/, "", value); print value; exit } }' "$OUT_DIR/alloc-total.txt")
if [ -z "$INUSE_BYTES" ] || [ -z "$ALLOC_BYTES" ]; then
    echo "Could not extract memory totals from pprof output." >&2
    exit 1
fi
printf 'label\tqueue_size\tinuse_bytes\talloc_bytes\n%s\t%s\t%s\t%s\n' \
    "$LABEL" "$QUEUE_SIZE" "$INUSE_BYTES" "$ALLOC_BYTES" >"$OUT_DIR/memory-summary.tsv"

cat >"$OUT_DIR/metadata.txt" <<EOF
label=$LABEL
target_go_dir=$TARGET_GO_DIR
branch=$(git -C "$TARGET_GO_DIR" branch --show-current)
commit=$(git -C "$TARGET_GO_DIR" rev-parse HEAD)
queue_size=$QUEUE_SIZE
k6_exit_code=$K6_STATUS
EOF

echo "CPU profile: $OUT_DIR/cpu.pprof"
echo "Heap profile after forced GC: $OUT_DIR/heap.pprof"
echo "Allocation profile: $OUT_DIR/allocs.pprof"
echo "Memory summary: $OUT_DIR/memory-summary.tsv"
exit "$K6_STATUS"
