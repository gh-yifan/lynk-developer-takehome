#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

base_url="${BASE_URL:-http://localhost:8080}"
demo_url="${base_url%/}/demo"
demo_version='<meta name="demo-version" content="1">'

demo_is_current() {
	local page
	page="$(curl -fsS "$1" 2>/dev/null)" || return 1
	[[ "$page" == *"$demo_version"* ]]
}

wait_for_demo() {
	local url="$1"
	local response_file
	response_file="$(mktemp)"
	if curl -fsS --retry 30 --retry-connrefused --retry-delay 1 --retry-max-time 30 "$url" -o "$response_file" 2>/dev/null && grep -qF "$demo_version" "$response_file"; then
		rm -f "$response_file"
		return 0
	fi
	rm -f "$response_file"
	return 1
}

if ! demo_is_current "$demo_url"; then
	if [[ "$base_url" != "http://localhost:8080" && "$base_url" != "http://127.0.0.1:8080" ]]; then
		printf 'Interactive demo is not reachable at %s; restart the API with the updated code.\n' "$demo_url" >&2
		exit 1
	fi

	port=8080
	if curl -fsS "$base_url/docs" -o /dev/null 2>/dev/null; then
		port=""
		for candidate in {8081..8090}; do
			if ! lsof -nP -iTCP:"$candidate" -sTCP:LISTEN >/dev/null 2>&1; then
				port="$candidate"
				break
			fi
		done
		if [[ -z "$port" ]]; then
			printf 'Ports 8081-8090 are occupied; stop an unused server or set BASE_URL.\n' >&2
			exit 1
		fi
		base_url="http://localhost:$port"
		demo_url="$base_url/demo"
	fi

	log_file="${TMPDIR:-/tmp}/command-queue-api-$port.log"
	PORT="$port" nohup go run main.go storage.go >"$log_file" 2>&1 </dev/null &
	printf 'Starting updated API on port %s; startup log: %s\n' "$port" "$log_file"
	if ! wait_for_demo "$demo_url"; then
		printf 'Could not start updated API. See %s\n' "$log_file" >&2
		exit 1
	fi
fi

printf 'Opening interactive queue demo: %s\n' "$demo_url"
open "$demo_url"