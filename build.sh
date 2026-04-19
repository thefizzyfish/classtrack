#!/usr/bin/env bash
set -e

echo "==> Building SvelteKit..."
cd web && npm run build && cd ..

echo "==> Staging web build into server/static/..."
rm -rf server/static
mkdir -p server/static
cp -r web/build/. server/static/

echo "==> Compiling Go binary for Linux (amd64)..."
cd server && GOOS=linux GOARCH=amd64 go build -o ../classtrack-linux . && cd ..

echo "==> Cleaning up server/static/..."
rm -rf server/static
mkdir -p server/static
touch server/static/.gitkeep

echo "==> Done — classtrack-linux ready to deploy"
