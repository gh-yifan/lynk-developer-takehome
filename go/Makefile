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

.PHONY: test go-test deploy demo teardown run

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