#!/usr/bin/env bash
# Render every VHS tape in docs/tapes into a GIF under docs/img.
#
# Requirements: vhs (github.com/charmbracelet/vhs), plus its own dependencies
# ttyd and ffmpeg on PATH.
#
# Usage: scripts/render-demos.sh
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

if ! command -v vhs >/dev/null 2>&1; then
	echo "vhs not found. Install it from https://github.com/charmbracelet/vhs" >&2
	exit 1
fi

# Build all tools into ./bin and put them on PATH so the tapes can call them
# by bare name (e.g. "giraffe 18m").
echo "Building tools into ./bin ..."
mkdir -p bin docs/img
go build -o bin ./cmd/...
export PATH="$repo_root/bin:$PATH"

for tape in docs/tapes/*.tape; do
	echo "Rendering $tape ..."
	vhs "$tape"
done

echo "Done. GIFs are in docs/img/."
