#!/bin/sh

set -eu

go_bin=${GO:-go}
target_os=${TARGETOS:-$("$go_bin" env GOOS)}
target_arch=${TARGETARCH:-$("$go_bin" env GOARCH)}
version=${VERSION:-v0}
build_date=${DATE:-unknown}
commit=${COMMIT:-unknown}
branch=${BRANCH:-unknown}

if [ -z "${OUTPUT:-}" ]; then
	echo "OUTPUT is required" >&2
	exit 1
fi

mkdir -p "$(dirname "$OUTPUT")"
export CGO_ENABLED="${CGO_ENABLED:-0}"
export GOOS="$target_os"
export GOARCH="$target_arch"

exec "$go_bin" build \
	-mod=readonly \
	-trimpath \
	-buildvcs=false \
	-tags release \
	-ldflags "-s -w -X main.Version=$version -X main.BuildDate=$build_date -X main.GitCommit=$commit -X main.GitBranch=$branch -X main.Platform=$target_os/$target_arch" \
	-o "$OUTPUT" \
	main.go
