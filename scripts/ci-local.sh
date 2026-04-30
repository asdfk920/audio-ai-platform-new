#!/usr/bin/env bash
# Local CI: same stages as .github/workflows/ci.yml (lint, merged coverage, build).
# Linux / macOS / Git Bash on Windows:  bash scripts/ci-local.sh
# Native Windows PowerShell:            .\scripts\ci-local.ps1
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
export GOTOOLCHAIN=local

GOPATH_BIN="$(go env GOPATH)/bin"
GOLANGCI=""
if [[ -x "$GOPATH_BIN/golangci-lint" ]]; then
	GOLANGCI="$GOPATH_BIN/golangci-lint"
elif [[ -x "$GOPATH_BIN/golangci-lint.exe" ]]; then
	GOLANGCI="$GOPATH_BIN/golangci-lint.exe"
else
	echo "golangci-lint not found in $GOPATH_BIN" >&2
	echo "Install CI version: go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8" >&2
	exit 1
fi

if [[ ! -f "$ROOT/.golangci.yml" ]]; then
	echo "missing $ROOT/.golangci.yml" >&2
	exit 1
fi

SERVICES=(services/user services/device services/content services/media-processing)

echo ""
echo "=== Root: go mod download / verify ==="
go mod download
go mod verify

echo ""
echo "=== Check DB migrations (*_down.sql paired) ==="
bash "$ROOT/scripts/check-migrations.sh"

echo ""
echo "=== Lint common ==="
(
	cd "$ROOT"
	"$GOLANGCI" run --config "$ROOT/.golangci.yml" --out-format=line-number --timeout=5m ./common/...
)

for svc in "${SERVICES[@]}"; do
	echo ""
	echo "=== Lint $svc ==="
	(
		cd "$ROOT/$svc"
		go mod tidy
		go mod verify
		"$GOLANGCI" run --config "$ROOT/.golangci.yml" --out-format=line-number --timeout=5m ./...
	)
done
echo ""
echo "Lint OK"

COV="$ROOT/coverage.out"
echo "mode: atomic" >"$COV"
for svc in "${SERVICES[@]}"; do
	echo ""
	echo "=== Test $svc ==="
	(
		cd "$ROOT/$svc"
		go test -v -race -covermode=atomic -count=1 -coverprofile=coverage.tmp ./...
		tail -n +2 coverage.tmp >>"$COV"
		rm -f coverage.tmp
	)
done
echo ""
echo "Test OK, merged coverage: $COV"

echo ""
echo "=== Build ==="
mkdir -p "$ROOT/bin"
EX="$(go env GOEXE)"
build_svc() {
	local dir="$1"
	local outname="$2"
	(
		cd "$ROOT/$dir"
		go mod tidy
		go mod verify
		go build -o "$ROOT/bin/${outname}${EX}" .
	)
}
build_svc services/user user-service
build_svc services/device device-service
build_svc services/content content-service
build_svc services/media-processing media-processing

echo "Build OK -> $ROOT/bin"
echo "Local CI finished."
