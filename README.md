# Context Engine

Reduces LLM token usage by storing and retrieving only relevant project context.

## Architecture

```
Query ──► Embedding ──► pgvector search ──► Graph expand (Neo4j) ──► Structured context
```

**Stack:** Go + Gin · PostgreSQL + pgvector · Neo4j · Redis

---

## Quick Start (macOS / Linux)

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

---

## Running on Windows

### Prerequisites

Install the following before starting:

| Tool | Download |
|------|----------|
| Go 1.21+ | https://go.dev/dl/ |
| Docker Desktop | https://www.docker.com/products/docker-desktop/ |
| Git | https://git-scm.com/download/win |

> All commands below are for **PowerShell**. Run as Administrator where noted.

---

### Step 1 — Start Infrastructure (Docker)

```powershell
# From the context-engine directory
docker compose up -d
```

Wait for all containers to be healthy:
```powershell
docker ps
# postgres, neo4j, redis should all show "healthy" or "Up"
```

---

### Step 2 — Build the Binary

```powershell
go mod tidy
go build -o bin\context-engine.exe .\cmd\server
```

---

### Step 3 — Set Environment Variables (optional)

```powershell
$env:SERVER_PORT   = "8080"
$env:POSTGRES_DSN  = "postgres://postgres:postgres@localhost:5432/contextdb?sslmode=disable"
$env:NEO4J_URI     = "neo4j://localhost:7687"
$env:NEO4J_USERNAME= "neo4j"
$env:NEO4J_PASSWORD= "password"
$env:REDIS_ADDR    = "localhost:6379"
```

Or create a `.env` file and load it:
```powershell
Get-Content .env | ForEach-Object {
  if ($_ -match "^([^#][^=]+)=(.+)$") {
    [System.Environment]::SetEnvironmentVariable($Matches[1].Trim(), $Matches[2].Trim())
  }
}
```

---

### Step 4 — Start the Server

```powershell
.\bin\context-engine.exe serve
```

You should see:
```
context-engine listening on :8080
```

---

### Step 5 — Ingest a Project

Open a **second PowerShell window**:

```powershell
# Ingest any Go project by path
.\bin\context-engine.exe ingest C:\path\to\your-project --project your-project-name

# Example — ingest this repo itself
.\bin\context-engine.exe ingest . --project context-engine

# Example — ingest a ride-hailing project
.\bin\context-engine.exe ingest C:\repos\ride-hailing --project ride-hailing
```

---

### Step 6 — Query Context

```powershell
.\bin\context-engine.exe query "how does driver matching work" --project ride-hailing
```

---

### Step 7 — Re-ingest After Code Changes

Made changes to your project? Just run ingest again with the same project name:

```powershell
.\bin\context-engine.exe ingest C:\repos\ride-hailing --project ride-hailing
```

The engine re-parses all `.go` files, regenerates embeddings, and updates the graph and vector store.

---

### Windows Troubleshooting

| Problem | Fix |
|---------|-----|
| `docker: command not found` | Start Docker Desktop and ensure it's added to PATH |
| `go: command not found` | Re-open PowerShell after installing Go, or add `C:\Program Files\Go\bin` to PATH manually |
| Port 5432 already in use | Stop any local PostgreSQL service: `Stop-Service postgresql*` |
| `dial tcp: connection refused` on Neo4j | Wait 30s after `docker compose up` — Neo4j takes longer to start |
| `context-engine.exe` blocked by antivirus | Add an exclusion for the `bin\` folder in Windows Defender |
| PowerShell `execution policy` error | Run `Set-ExecutionPolicy -Scope CurrentUser RemoteSigned` |

---

## API

### POST /v1/ingest-project
```json
{ "project_name": "AI-TMS", "path": "./repo" }
```

### POST /v1/query-context
```json
{ "project_name": "AI-TMS", "query": "how tat calculation works" }
```

### GET /v1/projects
```json
{
  "projects": [
    { "name": "ride-hailing", "file_count": 42, "func_count": 318 }
  ]
}
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

---

## Environment Variables

| Variable         | Default                                                                 |
|------------------|-------------------------------------------------------------------------|
| `SERVER_PORT`    | `8080`                                                                  |
| `POSTGRES_DSN`   | `postgres://postgres:postgres@localhost:5432/contextdb?sslmode=disable` |
| `NEO4J_URI`      | `neo4j://localhost:7687`                                                |
| `NEO4J_USERNAME` | `neo4j`                                                                 |
| `NEO4J_PASSWORD` | `password`                                                              |
| `REDIS_ADDR`     | `localhost:6379`                                                        |

---

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

---

## Extending

- **Real embeddings:** Replace `MockEmbedder` in `internal/embedding/` with an OpenAI/Cohere client implementing the `Embedder` interface.
- **Other languages:** Add a new parser in `internal/parser/` implementing the same `ParseDirectory` / `ParseFile` contract.
- **Auth middleware:** Add Gin middleware in `internal/api/router.go`.
