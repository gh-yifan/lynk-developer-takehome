SHELL := /bin/sh
PYTHON ?= python3

LABEL ?= baseline
QUEUE_SIZE ?= 20000
RESULTS_DIR ?= perf/results/$(LABEL)
API_ADDR ?= 127.0.0.1:8080
THROUGHPUT_VUS ?= 20
THROUGHPUT_DURATION ?= 60s
PROFILE_TARGET ?= ../lynk-main/go
CPU_SECONDS ?= 60
MEMORY_K6_PROFILE ?= profile
PPROF_ADDR ?= 127.0.0.1:6060
PROFILE_ROOT ?= perf/profiles/$(LABEL)
K6_PROFILE ?= throughput
DELETE_BUCKET_SIZE ?= 100
DELETE_RESULTS ?= perf/results/$(LABEL)/delete-staircase-$(QUEUE_SIZE).ndjson

.PHONY: all benchmark memory delete-staircase
all: benchmark memory

benchmark:
	API_ADDR="$(API_ADDR)" RESULTS_DIR="$(RESULTS_DIR)" K6_PROFILE="$(K6_PROFILE)" \
		THROUGHPUT_VUS="$(THROUGHPUT_VUS)" THROUGHPUT_DURATION="$(THROUGHPUT_DURATION)" \
		./perf/run_queue_sizes.sh "$(QUEUE_SIZE)"
	python3 perf/plot_results.py \
		--run "$(LABEL):$(QUEUE_SIZE)=$(RESULTS_DIR)/results-$(QUEUE_SIZE).ndjson" \
		--output "$(RESULTS_DIR)/latency.svg"
	python3 perf/plot_throughput.py \
		--run "$(LABEL):$(QUEUE_SIZE)=$(RESULTS_DIR)/summary-$(QUEUE_SIZE).json" \
		--output "$(RESULTS_DIR)/throughput.svg"

memory:
	TARGET_GO_DIR="$(PROFILE_TARGET)" API_ADDR="$(API_ADDR)" PORT="$${PORT:-8080}" \
		PPROF_ADDR="$(PPROF_ADDR)" CPU_SECONDS="$(CPU_SECONDS)" \
		K6_PROFILE="$(MEMORY_K6_PROFILE)" ./perf/profile_memory.sh "$(QUEUE_SIZE)" "$(LABEL)" "$(PROFILE_ROOT)"
	@set --; for profile in "$(PROFILE_ROOT)"/$(LABEL)-$(QUEUE_SIZE)-*; do \
		test -d "$$profile" && set -- "$$@" --run "$(LABEL):$(QUEUE_SIZE)=$$profile"; \
	done; \
		test "$$#" -gt 0 && python3 perf/plot_memory.py "$$@" --output "$(PROFILE_ROOT)/memory.svg"

delete-staircase:
	mkdir -p "$(RESULTS_DIR)" "$(dir $(DELETE_RESULTS))"
	k6 run \
		-e "BASE_URL=http://$(API_ADDR)" \
		-e "QUEUE_SIZE=$(QUEUE_SIZE)" \
		-e "DELETE_BUCKET_SIZE=$(DELETE_BUCKET_SIZE)" \
		--out "json=$(DELETE_RESULTS)" \
		perf/delete_staircase.js
	python3 perf/plot_delete_staircase.py \
		--run "$(LABEL)=$(DELETE_RESULTS)" \
		--output "$(RESULTS_DIR)/delete-staircase-$(LABEL)-$(QUEUE_SIZE).svg"
