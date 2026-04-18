# Context Engine

Reduces LLM token usage by storing and retrieving only relevant project context.

## Architecture

```
Query ──► Embedding ──► pgvector search ──► Graph expand (Neo4j) ──► Structured context
```

**Stack:** Go + Gin · PostgreSQL + pgvector · Neo4j · Redis

## Quick Start

```bash
# 1. Spin up infrastructure
./scripts/setup.sh

# 2. Start the server
./bin/context-engine serve

# 3. Ingest a Go project
./bin/context-engine ingest ./my-project --project my-project

# 4. Query for context
./bin/context-engine query "how does TAT calculation work" --project my-project
```

## API

### POST /v1/ingest-project
```json
{ "project_name": "AI-TMS", "path": "./repo" }
```

### POST /v1/query-context
```json
{ "project_name": "AI-TMS", "query": "how tat calculation works" }
```

### Response (query)
```json
{
  "query": "how tat calculation works",
  "files": [...],
  "functions": [...],
  "related_files": [...]
}
```

## Environment Variables

| Variable        | Default                                              |
|-----------------|------------------------------------------------------|
| `SERVER_PORT`   | `8080`                                               |
| `POSTGRES_DSN`  | `postgres://postgres:postgres@localhost:5432/contextdb?sslmode=disable` |
| `NEO4J_URI`     | `neo4j://localhost:7687`                             |
| `NEO4J_USERNAME`| `neo4j`                                              |
| `NEO4J_PASSWORD`| `password`                                           |
| `REDIS_ADDR`    | `localhost:6379`                                     |

## Data Flow

**Ingest:**
```
ParseDirectory → AST extract (files, functions, imports)
              → MockEmbedder.Generate(summary)
              → Neo4j: Project→File→Function nodes + IMPORTS edges
              → pgvector: files + functions tables with vector(1536)
```

**Query:**
```
query string → MockEmbedder.Generate
             → pgvector cosine search (top-5 files, top-10 functions)
             → Neo4j expand each file 2 hops
             → Redis cache result (10 min TTL)
             → Return structured context
```

## Extending

- **Real embeddings:** Replace `MockEmbedder` in `internal/embedding/` with an OpenAI/Cohere client implementing the `Embedder` interface.
- **Other languages:** Add a new parser in `internal/parser/` implementing the same `ParseDirectory` / `ParseFile` contract.
- **Auth middleware:** Add Gin middleware in `internal/api/router.go`.
