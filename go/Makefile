SHELL := /bin/sh
PYTHON ?= python3
PORT ?= 8080

LABEL ?= baseline
SIZES ?= 0 1000 5000
RESULTS_DIR ?= perf/results/$(LABEL)
API_ADDR ?= 127.0.0.1:18080
THROUGHPUT_VUS ?= 20
THROUGHPUT_DURATION ?= 60s
MEMORY_QUEUE_SIZE ?= 10000
CPU_SECONDS ?= 60
MEMORY_K6_PROFILE ?= profile
PPROF_ADDR ?= 127.0.0.1:6060
PROFILE_ROOT ?= perf/profiles/make-$(LABEL)-$(shell date +%Y%m%d-%H%M%S)

.PHONY: all benchmark memory test go-test deploy demo teardown run

all: benchmark memory

benchmark:
	API_ADDR="$(API_ADDR)" K6_PROFILE=throughput THROUGHPUT_VUS="$(THROUGHPUT_VUS)" THROUGHPUT_DURATION="$(THROUGHPUT_DURATION)" RESULTS_DIR="$(RESULTS_DIR)" ./perf/run_queue_sizes.sh $(SIZES)
	@set --; for size in $(SIZES); do set -- "$$@" --run "$(LABEL):$$size=$(RESULTS_DIR)/results-$$size.ndjson"; done; \
		python3 perf/plot_results.py "$$@" --output "$(RESULTS_DIR)/latency.svg"
	@set --; for size in $(SIZES); do set -- "$$@" --run "$(LABEL):$$size=$(RESULTS_DIR)/summary-$$size.json"; done; \
		python3 perf/plot_throughput.py "$$@" --output "$(RESULTS_DIR)/throughput.svg"

memory:
	API_ADDR="$(API_ADDR)" PPROF_ADDR="$(PPROF_ADDR)" CPU_SECONDS="$(CPU_SECONDS)" \
		K6_PROFILE="$(MEMORY_K6_PROFILE)" ./perf/profile_memory.sh "$(MEMORY_QUEUE_SIZE)" "$(LABEL)" "$(PROFILE_ROOT)"
	@set --; for profile in "$(PROFILE_ROOT)"/$(LABEL)-$(MEMORY_QUEUE_SIZE)-*; do \
		test -d "$$profile" && set -- "$$@" --run "$(LABEL):$(MEMORY_QUEUE_SIZE)=$$profile"; \
	done; \
		test "$$#" -gt 0 && python3 perf/plot_memory.py "$$@" --output "$(PROFILE_ROOT)/memory.svg"

deploy:
	go run main.go storage.go

teardown:
	@pids=$$(lsof -tiTCP:$(PORT) -sTCP:LISTEN 2>/dev/null); \
	if [ -n "$$pids" ]; then \
		echo "Stopping listener(s) on port $(PORT): $$pids"; \
		kill $$pids; \
	else \
		echo "No listener on port $(PORT)"; \
	fi

demo:
	bash ./demo.sh

run: demo

go-test:
	go test ./...

test: go-test
	cd .. && $(PYTHON) validate.py go