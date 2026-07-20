#!/bin/sh

set -eu

artifact_dir=${1:-.bin}
cd "$artifact_dir"

set -- \
	secrets-init-darwin-amd64 \
	secrets-init-darwin-arm64 \
	secrets-init-linux-amd64 \
	secrets-init-linux-arm64

for artifact do
	if [ ! -s "$artifact" ]; then
		echo "missing release artifact: $artifact" >&2
		exit 1
	fi
done

if command -v sha256sum >/dev/null 2>&1; then
	sha256sum "$@" > SHA256SUMS
else
	shasum -a 256 "$@" > SHA256SUMS
fi
