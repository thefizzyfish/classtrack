#!/usr/bin/env zsh
# Start classtrack — Go API on :4000, SvelteKit on :5173
set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"

echo "Starting classtrack..."

# Kill anything already on these ports
lsof -ti:4000 | xargs kill -9 2>/dev/null || true
lsof -ti:5173 | xargs kill -9 2>/dev/null || true

# Go server (background)
(cd "$ROOT/server" && go run . ) &
GO_PID=$!
echo "Go API → http://localhost:4000 (pid $GO_PID)"

# SvelteKit dev (foreground so logs show)
(cd "$ROOT/web" && npm run dev)

# On exit, kill Go server
kill $GO_PID 2>/dev/null || true
