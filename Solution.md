
# storage
3-tier map of priority to FIFO list of command pointers for the complete ordered listing
3-tier map of priority to FIFO list of pending command pointers for dispatch
map of command ID (UUID) to its element in the complete priority list for O(1) lookup (PATCH and DELETE)
map of pending command ID to its element in the pending list for O(1) removal
the priority and pending lists point to the same command object, so status changes stay consistent


# operations
GET /commands - combine the complete lists in HIGH, MEDIUM, LOW order; list order preserves FIFO within each priority
POST /commands - validate, create a PENDING command, append it to both its complete priority list and pending list, and index both elements
POST /commands/dispatch - remove and return the front pending command from the highest-priority non-empty list; set it to EXECUTING and retain it in the complete priority list
PATCH command - look up through the ID map, validate the transition, update status and timestamp; remove it from pending when transitioning PENDING -> EXECUTING
DELETE command - remove it from the complete priority list and ID map; if still PENDING, also remove it from the pending list and pending ID map


# validations
command text must not be blank and must be at most 198 characters
priority and status values must be valid
PATCH only allows PENDING -> EXECUTING and EXECUTING -> COMPLETED|FAILED; completed and failed are terminal
PATCH and DELETE require an existing command ID; malformed UUIDs follow the API's 404 response contract
dispatch returns 404 when no pending command is available

# request validation
request DTOs (Data Transfer Objects) define the JSON fields and validation rules for each endpoint; they are separate from the stored Command model
register the validator library's notblank rule once and declare it on the command DTO alongside the 198-character limit
trim leading and trailing command whitespace before validation and storage; preserve internal whitespace
validate each endpoint's DTO in its handler because POST and PATCH have different fields and rules; no general validation middleware
translate validator failures into concise, stable API messages rather than returning raw library errors

# considerations
direct maps provide O(1) lookup by command ID; linked-list elements allow O(1) removal once located
the complete and pending lists hold pointers to one command object as the single source of truth
mutex protects shared maps, lists, and command state from concurrent HTTP requests
state transitions are checked in PATCH; middleware is unnecessary because validation is endpoint-specific

# decisions
use the validator package for DTO rules and custom notblank behavior; keep API error wording in application code
keep validation endpoint-specific rather than using general middleware
use mux path-value extraction for command IDs
use linked lists because commands can be deleted by ID without shifting remaining elements; pending lists also make dispatch select the next command directly

# tests
dispatch returns 404 for an empty pending queue and selects commands by priority and FIFO order
newly created higher-priority commands dispatch before lower-priority pending commands; newly created lower-priority commands do not overtake higher-priority work
delete at beginning, middle, and end - ensure FIFO order is preserved
GET after PATCH - ensure the updated status and updated_at timestamp are reflected
race-enabled tests cover competing status transitions and concurrent command creation
API regression tests cover command validation boundaries and rejected transitions preserving status

# how to run
deploy server: `make deploy`
test the server: `make test`
demo UI to exercise the server: `make run` 

demo UI is available at localhost:8081/demo
open api spec is available at localhost:8080/docs

## Comparing Implementations with Git Worktrees

You do **not** apply or merge `main` or the DLL branch into `test/perf`. The `test/perf` worktree contains the k6 client and charts; it sends HTTP requests to whichever implementation server you start. Each implementation stays in its own worktree, so benchmark files do not need to be copied into or merged with either implementation branch.

Check the worktree list first. In this checkout, the root directory is already `test/perf`, and `lynk-main` and `lynk-dll` are already separate worktrees. If setting this up on another machine, create them from the repository root (each branch can only be checked out in one worktree at a time):

```bash
git worktree add ../lynk-main main
git worktree add ../lynk-dll feat/doubly-linked-list
```

Assume the `test/perf` branch is checked out at `/Users/yifanguo/Downloads/lynk-developer-takehome`. Its Go directory is `/Users/yifanguo/Downloads/lynk-developer-takehome/go` (there is no literal `test/perf/go` directory). Start the API from the implementation worktree, not this `test/perf` checkout. In terminal 1, start `main` like this:

```bash
cd ../lynk-main/go
go run main.go storage.go
```

Open terminal 2 and change to the Go directory of the `test/perf` checkout:

```bash
cd /Users/yifanguo/Downloads/lynk-developer-takehome/go
```

Run the k6 client from there. `USE_EXISTING_API=1` tells the script to target the running API and not to start or stop it:

```bash
USE_EXISTING_API=1 API_ADDR=127.0.0.1:8080 \
	RESULTS_DIR=perf/results/main K6_PROFILE=throughput \
	THROUGHPUT_VUS=20 THROUGHPUT_DURATION=60s \
	./perf/run_queue_sizes.sh 20000
```

Use exactly one queue size per external-server invocation. The API stores commands in memory, and k6 seeds the requested baseline during setup. Stop the API with Ctrl+C after the run. Restart it before measuring another size or implementation so the next run starts with an empty queue. The runner will not build, start, or kill the implementation server in this mode.

The DLL branch currently hardcodes port 8080, so do not pass it `PORT=18081` and expect that to change its listen address. Stop the `main` server with Ctrl+C. In terminal 1, which is currently in `lynk-main/go`, switch to the DLL worktree and start its API:

```bash
cd ../../lynk-dll/go
go run main.go storage.go
```

Terminal 2 remains in `/Users/yifanguo/Downloads/lynk-developer-takehome/go`; run the same workload there, changing only the output directory/label:

```bash
USE_EXISTING_API=1 API_ADDR=127.0.0.1:8080 \
	RESULTS_DIR=perf/results/dll K6_PROFILE=throughput \
	THROUGHPUT_VUS=20 THROUGHPUT_DURATION=60s \
	./perf/run_queue_sizes.sh 20000
```

Create comparison charts from the separate result files:

```bash
python3 perf/plot_results.py \
	--run main:20000=perf/results/main/results-20000.ndjson \
	--run dll:20000=perf/results/dll/results-20000.ndjson \
	--output perf/results/comparison-latency.svg

python3 perf/plot_throughput.py \
	--run main:20000=perf/results/main/summary-20000.json \
	--run dll:20000=perf/results/dll/summary-20000.json \
	--output perf/results/comparison-throughput.svg
```

Repeat with the same machine, Go version, queue size, k6 settings, and similar system load. Verify each result's `summary-<size>.json` and k6 output; a threshold failure can return a nonzero exit even though result files were written. These latency/throughput runs need no code changes to either implementation, so they cannot create merge conflicts.

### CPU and Heap Profiling with Go Overlays

The implementation branches do not need pprof routes in their application mux. The `test/perf` branch supplies `go/perf/pprof_init.go` as a virtual source file using `go build -overlay`; Go compiles it into a temporary binary without writing it into the target worktree. The pprof handlers use their own loopback listener at `PPROF_ADDR`, separate from the API port. Commit `go/perf/pprof_init.go` with the perf tooling so other users can run the overlay; `.gitignore` explicitly allows this file.

From the `test/perf/go` directory, this is the exact overlay build for `main`:

```bash
PERF_GO_DIR="$PWD"
TARGET_GO_DIR="$(cd ../lynk-main/go && pwd)"
OVERLAY_FILE="$(mktemp)"
printf '{"Replace":{"%s/zz_perf_pprof.go":"%s/perf/pprof_init.go"}}\n' \
	"$TARGET_GO_DIR" "$PERF_GO_DIR" > "$OVERLAY_FILE"
(cd "$TARGET_GO_DIR" && go build -overlay="$OVERLAY_FILE" -o /tmp/command-queue-main-profiled .)
rm "$OVERLAY_FILE"
```

For DLL, use its worktree and a separate output binary:

```bash
PERF_GO_DIR="$PWD"
TARGET_GO_DIR="$(cd ../lynk-dll/go && pwd)"
OVERLAY_FILE="$(mktemp)"
printf '{"Replace":{"%s/zz_perf_pprof.go":"%s/perf/pprof_init.go"}}\n' \
	"$TARGET_GO_DIR" "$PERF_GO_DIR" > "$OVERLAY_FILE"
(cd "$TARGET_GO_DIR" && go build -overlay="$OVERLAY_FILE" -o /tmp/command-queue-dll-profiled .)
rm "$OVERLAY_FILE"
```

Each build uses that target worktree's own `go.mod` and `go.sum`. If dependencies or checksums are missing, run `go mod tidy` in that implementation worktree and keep the resulting module-file changes on that branch; the overlay helper itself imports only Go standard-library packages.

To build, run, seed, load-test, and capture CPU/heap profiles in one command, use the wrapper instead of the manual `go build` commands:

```bash
TARGET_GO_DIR=../lynk-main/go K6_PROFILE=profile \
	./perf/profile_memory.sh 20000 main perf/profiles

TARGET_GO_DIR=../lynk-dll/go K6_PROFILE=profile \
	./perf/profile_memory.sh 20000 dll perf/profiles
```

The wrapper compiles the temporary overlay binary, runs it with the API and pprof listeners on separate ports, and saves artifacts plus the target branch/commit metadata. It does not modify either implementation worktree. For a short validation run, set `K6_PROFILE=smoke` and `CPU_SECONDS=7`. Compare the generated `memory-summary.tsv` or use `perf/plot_memory.py` to chart retained heap and cumulative allocations. k6 latency measurements should use the normal non-race build; pprof adds profiling overhead, so treat these profiles as CPU/memory diagnostics rather than the latency benchmark.

You do not need to remove the `test/perf` worktree to benchmark. When finished, optionally remove only the extra implementation worktrees from the repository root:

```bash
git worktree remove ../lynk-main
git worktree remove ../lynk-dll
```

Do not force-remove a worktree with uncommitted changes; inspect and preserve those changes first. The benchmark branch remains separate and can be updated or merged independently, avoiding perf-tooling conflicts with implementation work.

[Demo](https://github.com/user-attachments/assets/74ef10ad-6d64-483d-8e04-de08908f9d13)

