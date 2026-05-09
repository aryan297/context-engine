# How Context Engine Works Internally

A detailed walkthrough of everything that happens in the background when you run `ingest` and `query`.

---

## System Overview

Context Engine has three storage layers working together:

```
┌─────────────────────────────────────────────────────────────┐
│                        Context Engine                        │
│                                                             │
│   ┌──────────┐    ┌──────────────┐    ┌──────────────────┐ │
│   │  Parser  │    │  Embedder    │    │   HTTP API (Gin)  │ │
│   │ Registry │    │ (Mock/Real)  │    │  /v1/ingest       │ │
│   └──────────┘    └──────────────┘    │  /v1/query        │ │
│                                       │  /v1/projects     │ │
│   ┌──────────────────────────────┐    └──────────────────┘ │
│   │         Storage Layer        │                          │
│   │  PostgreSQL+pgvector  Neo4j  │  Redis                   │
│   │  (vector similarity)  (graph)│  (cache)                 │
│   └──────────────────────────────┘                          │
└─────────────────────────────────────────────────────────────┘
```

---

## Part 1 — INGEST

What happens when you run:
```bash
ctx ingest ./my-project --project my-project
```

### Step 1 — HTTP Request

The CLI sends a POST to the server:
```
POST /v1/ingest-project
{ "project_name": "my-project", "path": "./my-project" }
```

The handler (`internal/api/handler/ingest_handler.go`) receives it and calls `IngestService.IngestProject()`.

---

### Step 2 — Project Node Created in Neo4j

```
IngestService → graphRepo.StoreProject(project)
```

A `Project` node is written to Neo4j using MERGE (so re-ingesting doesn't duplicate it):

```cypher
MERGE (p:Project {id: $id})
SET p.name = $name, p.path = $path
```

The graph now has:
```
(Project {id, name, path})
```

---

### Step 3 — File Walk + Language Dispatch

```
IngestService → parser.Registry.ParseDirectory(projectID, projectName, path)
```

The `Registry` walks the directory tree with `filepath.Walk`. For every file it hits:

1. Checks if the **directory** should be skipped:
   - Hard-coded list: `node_modules`, `.git`, `vendor`, `.next`, `dist`, `build`, `__pycache__`, `target`, etc.
   - Also reads the project's `.gitignore` and skips any directories listed there
2. Checks the **file extension** against the registered parsers:

```
.go   → GoParser   (uses Go's built-in go/ast — full AST)
.ts   → TypeScriptParser (regex)
.tsx  → TypeScriptParser (regex)
.java → JavaParser  (regex)
.py   → PythonParser (regex)
.rs   → RustParser  (regex)
other → skipped
```

---

### Step 4 — Parsing Each File

Each parser extracts three things from a source file:

| Extracted | Description |
|-----------|-------------|
| **Imports** | What other modules/packages this file depends on |
| **Functions** | Function/method/class names and signatures |
| **Summary** | A plain-text description: `"Go file foo.go with 12 functions and 5 imports"` |

The summary is the text that gets turned into a vector embedding in the next step.

**Go parser** uses the real AST — it can extract exact parameter types and return types.
**All other parsers** use regex patterns — they extract names and signatures accurately but without full type resolution.

---

### Step 5 — Embedding Generation

```
IngestService → embedder.Generate(file.Summary) → []float32 (1536 dimensions)
```

For every file and every function, the embedder converts the summary text into a **1536-dimensional float32 vector**.

This vector is a numerical representation of the meaning of the text. Similar summaries produce vectors that are close together in 1536-dimensional space — this is what makes semantic search work.

**Current implementation:** `MockEmbedder` generates a random normalized unit vector. The math ensures the vector has length 1 (unit vector), which is required for cosine similarity to work correctly.

```go
// Normalize to unit vector so cosine similarity = dot product
norm := sqrt(sum of squares)
vec[i] /= norm
```

**Production replacement:** swap `MockEmbedder` with an OpenAI/Cohere API call — the interface is identical.

---

### Step 6 — Write to Neo4j (Graph)

For each file, two things are written:

**File node + CONTAINS edge:**
```cypher
MERGE (f:File {id: $id})
SET f.name = $name, f.path = $path, f.summary = $summary
MERGE (p:Project {id: $project_id})-[:CONTAINS]->(f)
```

**Import edges (dependency map):**
```cypher
MERGE (dep:Import {path: $import_path})
MERGE (f:File {id: $file_id})-[:IMPORTS]->(dep)
```

**Function node + DEFINES edge:**
```cypher
MERGE (fn:Function {id: $id})
SET fn.name = $name, fn.signature = $signature
MERGE (f:File {id: $file_id})-[:DEFINES]->(fn)
```

After ingestion the Neo4j graph looks like this:

```
(Project)
    └─[:CONTAINS]→ (File: ride/service.go)
                        ├─[:DEFINES]→ (Function: BookRide)
                        ├─[:DEFINES]→ (Function: CancelRide)
                        └─[:IMPORTS]→ (Import: github.com/lib/pq)

    └─[:CONTAINS]→ (File: ride/handler.go)
                        ├─[:DEFINES]→ (Function: CreateRide)
                        └─[:IMPORTS]→ (Import: github.com/gin-gonic/gin)
```

The graph captures **which files depend on which**, and **which functions live in which file** — this is used later during query to expand context.

---

### Step 7 — Write to PostgreSQL + pgvector

```
IngestService → vectorRepo.StoreFileEmbedding(file)
IngestService → vectorRepo.StoreFunctionEmbedding(fn)
```

Two tables store the vectors:

```sql
-- files table
INSERT INTO files (id, project_name, name, path, summary, embedding)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET summary = ..., embedding = ...

-- functions table
INSERT INTO functions (id, file_id, project_name, name, signature, summary, embedding)
VALUES (...)
ON CONFLICT (id) DO UPDATE SET summary = ..., embedding = ...
```

The `embedding` column is type `vector(1536)` — a pgvector type. pgvector stores the 1536 floats and creates an **IVFFlat index** for fast approximate nearest-neighbour search:

```sql
CREATE INDEX files_embedding_idx
    ON files USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
```

`ON CONFLICT DO UPDATE` means re-ingesting the same project updates the vectors in-place — no duplicates.

---

### Ingest Summary

```
CLI → POST /v1/ingest-project
      └─ IngestService
           ├─ Neo4j: MERGE Project node
           ├─ Registry.ParseDirectory
           │    ├─ Skip: node_modules, .git, vendor, .gitignore entries, etc.
           │    └─ Dispatch by extension → GoParser / TSParser / JavaParser / PythonParser / RustParser
           └─ For each file:
                ├─ Embedder.Generate(summary) → vector[1536]
                ├─ Neo4j: MERGE File + Function nodes, CONTAINS / DEFINES / IMPORTS edges
                └─ PostgreSQL: INSERT files + functions with embedding vector
```

---

## Part 2 — QUERY

What happens when you run:
```bash
ctx query "how does driver matching work" --project ride-hailing
```

### Step 1 — HTTP Request

```
POST /v1/query-context
{ "project_name": "ride-hailing", "query": "how does driver matching work" }
```

`QueryService.QueryContext()` is called.

---

### Step 2 — Redis Cache Check

```go
cacheKey := "ctx:ride-hailing:how does driver matching work"
cached, err := cacheRepo.Get(cacheKey)
```

If this exact query was run in the last **10 minutes**, the full result is returned from Redis immediately — no DB calls at all. Cache hit is logged and the function returns.

---

### Step 3 — Embed the Query

```
embedder.Generate("how does driver matching work") → vector[1536]
```

The same embedder that was used during ingest converts the query string into a vector. This is the key insight: **files and queries live in the same vector space** — so "how does driver matching work" produces a vector close to the summary vectors of files that contain matching logic.

---

### Step 4 — Vector Search (PostgreSQL + pgvector)

Two cosine similarity searches run against the pgvector tables:

**Top-5 files:**
```sql
SELECT id, project_name, name, path, summary
FROM files
WHERE project_name = 'ride-hailing'
ORDER BY embedding <=> '[0.023, -0.441, ...]'::vector
LIMIT 5
```

**Top-10 functions:**
```sql
SELECT id, file_id, project_name, name, signature, summary
FROM functions
WHERE project_name = 'ride-hailing'
ORDER BY embedding <=> '[0.023, -0.441, ...]'::vector
LIMIT 10
```

The `<=>` operator is pgvector's **cosine distance** operator. Lower = more similar. The IVFFlat index makes this fast even with millions of rows.

The result: the 5 files and 10 functions whose summaries are semantically closest to the query.

---

### Step 5 — Graph Expansion (Neo4j)

For each of the top-5 matched files, the graph is traversed **2 hops** outward:

```cypher
MATCH (start:File {id: $file_id})-[*1..2]-(related:File)
RETURN DISTINCT related.id, related.name, related.path, related.summary
```

`[*1..2]` means: follow any relationship type, up to 2 edges deep.

**Why this matters:** The vector search finds files that semantically match the query. The graph expansion finds files that are **structurally related** — files that import the matched files, or are imported by them. These are often critical context that the vector search alone would miss.

Example: the query matches `ride/service.go`. The graph expansion also returns `ride/handler.go` (which imports the service) and `ride/repository.go` (which the service imports).

---

### Step 6 — Cache and Return

```go
result := &QueryResult{
    Query:     query,
    Files:     files,      // top-5 by vector similarity
    Functions: functions,  // top-10 by vector similarity
    Related:   related,    // graph-expanded neighbours
}

// Cache for 10 minutes
cacheRepo.Set(cacheKey, json.Marshal(result), 10*time.Minute)
```

The result is serialized to JSON, stored in Redis with a 10-minute TTL, and returned to the caller.

---

### Query Summary

```
CLI → POST /v1/query-context
      └─ QueryService
           ├─ Redis: check cache → return if hit
           ├─ Embedder.Generate(query) → vector[1536]
           ├─ pgvector: cosine search → top-5 files + top-10 functions
           ├─ Neo4j: for each matched file, expand 2 hops → related files
           ├─ Redis: cache result for 10 min
           └─ Return { files, functions, related_files }
```

---

## Data Flow Diagram

```
INGEST                                    QUERY
──────                                    ─────
Source files                              "how does X work"
     │                                          │
     ▼                                          ▼
Parser Registry                          Embedder.Generate
(dispatch by .ext)                       → vector[1536]
     │                                          │
     ▼                                    ┌─────┴──────┐
File{                                     │            │
  imports: [...]                          ▼            ▼
  functions: [...]               pgvector cosine   Redis cache
  summary: "..."                  search            check
}                                  │   │
     │                         top-5   top-10
     ▼                         files   funcs
Embedder.Generate                 │
→ vector[1536]                    ▼
     │                       Neo4j expand
  ┌──┴──────────┐             2 hops each
  ▼             ▼                  │
Neo4j        pgvector              ▼
MERGE        INSERT           related files
nodes+edges  with vector           │
                               ┌───┴────────────┐
                               │  Final result  │
                               │  files         │
                               │  functions     │
                               │  related_files │
                               └────────────────┘
                                       │
                                   Redis SET
                                   (10 min TTL)
```

---

## Storage Responsibilities

| Store | What it holds | Used for |
|-------|---------------|----------|
| **PostgreSQL + pgvector** | File and function vectors (1536-dim float32) | Cosine similarity search — "what is semantically similar to this query?" |
| **Neo4j** | Project/File/Function nodes + CONTAINS/DEFINES/IMPORTS edges | Graph traversal — "what files are structurally related to this file?" |
| **Redis** | Serialized `QueryResult` JSON keyed by `ctx:<project>:<query>` | Cache — avoid re-running DB queries for repeated questions |

---

## Key Numbers

| Parameter | Value | Defined in |
|-----------|-------|------------|
| Embedding dimensions | 1536 | `internal/embedding/mock_embedding.go` |
| Top-K files returned | 5 | `internal/service/query_service.go` |
| Top-K functions returned | 10 | `internal/service/query_service.go` |
| Graph expansion depth | 2 hops | `internal/service/query_service.go` |
| Cache TTL | 10 minutes | `internal/service/query_service.go` |
| pgvector IVFFlat lists | 100 | `internal/storage/postgres/vector_store.go` |

---

## Source Code Map

```
cmd/server/main.go              — wires all components together, starts HTTP server
internal/config/config.go       — reads env vars (DSN, ports, passwords)
internal/api/router.go          — Gin router, CORS, /health + /v1/* routes
internal/api/handler/
  ingest_handler.go             — POST /v1/ingest-project
  query_handler.go              — POST /v1/query-context
  project_handler.go            — GET /v1/projects
internal/service/
  ingest_service.go             — orchestrates parse → embed → store
  query_service.go              — orchestrates cache → embed → search → expand → cache
internal/parser/
  parser.go                     — Parser interface
  registry.go                   — walks dirs, dispatches by extension, skips noise dirs
  go_parser.go                  — Go AST parser
  ts_parser.go                  — TypeScript/TSX regex parser
  java_parser.go                — Java regex parser
  python_parser.go              — Python regex parser
  rust_parser.go                — Rust regex parser
internal/embedding/
  mock_embedding.go             — random normalized 1536-dim vector (replace for production)
internal/storage/
  postgres/vector_store.go      — pgvector INSERT + cosine search
  neo4j/graph_store.go          — MERGE nodes, IMPORTS/DEFINES/CONTAINS edges, 2-hop expand
  redis/cache.go                — JSON GET/SET with TTL
internal/domain/
  model/                        — Project, File, Function structs
  repository/                   — GraphRepository, VectorRepository, CacheRepository interfaces
pkg/cli/
  ingest.go                     — CLI ingest subcommand (POSTs to server)
  query.go                      — CLI query subcommand (POSTs to server, pretty-prints)
```
