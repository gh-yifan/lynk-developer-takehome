# API Performance Tools

This worktree contains k6 workloads, result plotters, and Go profiling helpers. It does not contain or run the command API implementation. Start the API from a separate implementation worktree (`lynk-main` or `lynk-dll`) and point these tools at it.

## Requirements

- Go, for compiling the selected implementation
- k6, for load generation: https://grafana.com/docs/k6/latest/set-up/install-k6/
- Python 3, for SVG result charts

## Latency and Throughput

From this worktree's `go/` directory, start the chosen implementation in another terminal. Both current variants use port 8080:

```bash
cd ../lynk-main/go
go run main.go storage.go
```

Then, from this worktree's `go/` directory, run one queue-size measurement against that already-running server:

```bash
API_ADDR=127.0.0.1:8080 RESULTS_DIR=perf/results/main \
  K6_PROFILE=throughput THROUGHPUT_VUS=20 THROUGHPUT_DURATION=60s \
  ./perf/run_queue_sizes.sh 20000
```

Stop the API with Ctrl+C, restart the other implementation, and use a separate results directory:

```bash
cd ../lynk-dll/go
go run main.go storage.go
```

```bash
API_ADDR=127.0.0.1:8080 RESULTS_DIR=perf/results/dll \
  K6_PROFILE=throughput THROUGHPUT_VUS=20 THROUGHPUT_DURATION=60s \
  ./perf/run_queue_sizes.sh 20000
```

The runner only makes HTTP requests; it does not build, start, or stop the API. It seeds the requested baseline queue before timed measurements, so invoke it once per queue size and restart the API before the next measurement. A threshold failure may return a nonzero k6 exit code while still writing the result files.

Generate latency and successful-cycle throughput comparisons:

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

## CPU and Memory Profiles

Go pprof must profile the implementation process itself. The shared `perf/pprof_init.go` file is added to a temporary build using Go's `-overlay` option; it is not copied into or committed on the implementation branch, and it registers pprof on a separate loopback listener rather than changing the API mux.

From this worktree's `go/` directory, profile a target implementation with:

```bash
TARGET_GO_DIR=../lynk-main/go \
  K6_PROFILE=smoke CPU_SECONDS=7 \
  ./perf/profile_memory.sh 100 main /tmp/profiles
```

Use the DLL target for a comparison run:

```bash
TARGET_GO_DIR=../lynk-dll/go \
  K6_PROFILE=smoke CPU_SECONDS=7 \
  ./perf/profile_memory.sh 100 dll /tmp/profiles
```

The profiling script builds the selected target branch's sources with a temporary overlay file from this worktree, then writes the instrumented binary and profile artifacts under the requested output directory. It does not edit target source files or add pprof routes to the API mux. Keep queue size, Go version, k6 profile, and machine conditions the same between runs. The target branch's own `go.mod` and `go.sum` are used; fix dependency checksum issues in that target worktree with `go mod tidy` and commit those module files on that branch.

For example, profile the DLL branch with a short smoke workload:

```bash
make memory PROFILE_TARGET=../lynk-dll/go LABEL=dll QUEUE_SIZE=100 \
  MEMORY_K6_PROFILE=smoke CPU_SECONDS=7 PROFILE_ROOT=/tmp/perf-profiles
```

Profile outputs include `cpu.pprof`, `heap.pprof`, `allocs.pprof`, text top reports, and `memory-summary.tsv`. For pprof's web UI, install Graphviz if you want graph rendering, then run `go tool pprof -http=127.0.0.1:8081` with the matching generated binary and profile.
