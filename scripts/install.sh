#!/bin/bash
set -e

BINARY="$(cd "$(dirname "$0")/.." && pwd)/bin/context-engine"
SYMLINK="/usr/local/bin/ctx"

if [ ! -f "$BINARY" ]; then
  echo "Binary not found at $BINARY. Building..."
  cd "$(dirname "$0")/.."
  go build -o bin/context-engine ./cmd/cli
fi

chmod +x "$BINARY"

if [ -L "$SYMLINK" ]; then
  rm "$SYMLINK"
fi

ln -s "$BINARY" "$SYMLINK"
echo "Installed: ctx -> $BINARY"
echo ""
echo "Usage:"
echo "  ctx ingest ./your-project --project your-project"
echo "  ctx query \"your question\" --project your-project"
