#!/usr/bin/env bash
# build.sh — one-click build of moyu-reader.exe with the version baked in.
#
# The version is derived automatically from git (the latest tag), so you never
# type it by hand:
#   - on a tag (e.g. v0.6.2)   -> 0.6.2          -> moyu-reader_v0.6.2.exe
#   - commits past a tag       -> 0.6.2-3-gabc1  -> moyu-reader_v0.6.2-3-gabc1.exe
#   - uncommitted changes      -> ...-dirty
#   - no tags at all           -> a short commit hash, or "dev"
#
# Usage (from anywhere):  ./scripts/build.sh
set -euo pipefail

# Build from the repo root regardless of where the script is invoked.
cd "$(dirname "$0")/.."

ver="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
ver="${ver#v}"
out="moyu-reader_v${ver}.exe"

echo "Building ${out} (version ${ver}) ..."
GOOS=windows GOARCH=amd64 go build \
  -ldflags "-s -w -X moyureader/internal/version.Version=${ver}" \
  -o "${out}" ./cmd/reader
echo "Done: ${out}"
