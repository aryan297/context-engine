#!/usr/bin/env bash
set -euo pipefail

echo "==> Starting infrastructure..."
docker compose up -d

echo "==> Waiting for Postgres to be ready..."
until docker exec context_postgres pg_isready -U postgres > /dev/null 2>&1; do
  sleep 1
done

echo "==> Waiting for Neo4j to be ready..."
until curl -s http://localhost:7474 > /dev/null 2>&1; do
  sleep 2
done

echo "==> Waiting for Redis to be ready..."
until docker exec context_redis redis-cli ping > /dev/null 2>&1; do
  sleep 1
done

echo "==> Downloading Go dependencies..."
go mod tidy

echo "==> Building binary..."
go build -o bin/context-engine ./cmd/server

echo ""
echo "Setup complete!"
echo ""
echo "Run the server:              ./bin/context-engine serve"
echo "Ingest a project:            ./bin/context-engine ingest /path/to/project --project my-project"
echo "Query context:               ./bin/context-engine query 'how does X work' --project my-project"
