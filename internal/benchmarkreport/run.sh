#!/bin/bash

set -eo pipefail

here="$(dirname "$(readlink -f "$0")")"

tmp="$(mktemp)"

for ((i = 0; i < 50; i++)); do
	go test \
		github.com/GruffGemini/simplefeatures/internal/perf \
		-run=^\$ -bench=SetOperation \
		-benchtime 0.1s
done | tee "$tmp"

go run "$here/main.go" "$tmp"
