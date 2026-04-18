# How to Give Context of Any Project to Context Engine

## Prerequisites

Make sure the Context Engine server is running:

```bash
# 1. Spin up infrastructure (PostgreSQL, Neo4j, Redis)
./scripts/setup.sh

# 2. Start the server
./bin/context-engine serve
```

Server runs on `http://localhost:8080` by default.

---

## One-Time Install (use `ctx` from anywhere)

Run this once so you never need to type the full binary path again:

```bash
cd /Users/administrator/Desktop/basic/context-os/context-engine
sudo ./scripts/install.sh
```

After install, use the short `ctx` command from **any project folder**:

```bash
# Ingest current folder
ctx ingest . --project your-project-name

# Query
ctx query "your question" --project your-project-name
```

---

## Step 1 — Ingest Your Project

Pick either the CLI or HTTP API:

### Option A: CLI

```bash
./bin/context-engine ingest /path/to/your-project --project your-project-name
```

**Examples:**

```bash
# Ingest a project in the current directory
./bin/context-engine ingest ./my-app --project my-app

# Ingest a project from an absolute path
./bin/context-engine ingest /Users/john/projects/billing-service --project billing-service

# Ingest any folder on your machine
./bin/context-engine ingest /any/folder/on/your/machine --project folder-name
```

### Option B: HTTP API

```bash
curl -X POST http://localhost:8080/v1/ingest-project \
  -H "Content-Type: application/json" \
  -d '{
    "project_name": "your-project-name",
    "path": "/path/to/your-project"
  }'
```

**Examples:**

```bash
# Ingest a Go service
curl -X POST http://localhost:8080/v1/ingest-project \
  -H "Content-Type: application/json" \
  -d '{"project_name": "order-service", "path": "/projects/order-service"}'

# Ingest any local folder
curl -X POST http://localhost:8080/v1/ingest-project \
  -H "Content-Type: application/json" \
  -d '{"project_name": "my-repo", "path": "./my-repo"}'
```

---

## Step 2 — Query Your Project Context

After ingestion, ask questions about your project:

### Option A: CLI

```bash
ctx query "your question here" --project your-project-name
```

**Examples:**

```bash
ctx query "how does authentication work" --project my-app
ctx query "where is the TAT calculation logic" --project billing-service
ctx query "how are database connections managed" --project order-service
```

---

## Using Context in Different IDEs

The context engine is IDE-agnostic — query it from your terminal, then paste or pipe the result into whichever IDE/AI tool you use.

### VS Code (GitHub Copilot Chat / Continue.dev)

```bash
# Save context to a temp file
ctx query "how does booking work" --project ride_hailing > /tmp/ctx.md

# Open it in VS Code
code /tmp/ctx.md
```

Then in Copilot Chat or Continue, reference it:
```
@workspace (paste content from ctx.md)

Question: How does booking work?
```

---

### Cursor IDE

```bash
ctx query "how does fare calculation work" --project ride_hailing > /tmp/ctx.md
```

In Cursor, open the Chat panel (`Cmd+L`) and paste:
```
Use this project context:
[paste content of /tmp/ctx.md]

Question: How does fare calculation work?
```

Or use Cursor's `@file` to reference it directly:
```
@/tmp/ctx.md How does fare calculation work?
```

---

### JetBrains (GoLand / IntelliJ) + AI Assistant

```bash
ctx query "how is the repository layer structured" --project ride_hailing > /tmp/ctx.md
```

Open the AI Assistant panel, paste the file content, then ask your question.

---

### Any IDE — Universal Shell Helper

Add this function to your `~/.zshrc` so you can query + copy to clipboard in one command:

```bash
# Add to ~/.zshrc
function ask() {
  local project=$1
  local question="${@:2}"
  ctx query "$question" --project "$project" | pbcopy
  echo "Context copied to clipboard for: $question"
  echo "Now paste it into your IDE chat with your question."
}
```

Usage from any terminal:
```bash
ask ride_hailing "how does driver matching work"
# Context is now in your clipboard — paste into any IDE chat
```

---

### Claude Code (this CLI)

```bash
# Query and pipe directly into Claude Code from your project folder
ctx query "how does trip status update" --project ride_hailing | claude "Explain how trip status updates work based on this context"
```

### Option B: HTTP API

```bash
curl -X POST http://localhost:8080/v1/query-context \
  -H "Content-Type: application/json" \
  -d '{
    "project_name": "your-project-name",
    "query": "your question here"
  }'
```

**Example response:**

```json
{
  "query": "how does authentication work",
  "files": [
    { "path": "internal/auth/middleware.go", "score": 0.95 }
  ],
  "functions": [
    { "name": "ValidateToken", "file": "internal/auth/jwt.go", "score": 0.91 }
  ],
  "related_files": [
    "internal/api/router.go",
    "internal/models/user.go"
  ]
}
```

---

## Step 3 — Feed Context into a Claude Prompt

Once you have the query response, pass the relevant files/functions as context to Claude.

### Pattern: Fetch context → Build prompt → Call Claude API

```python
import httpx
import anthropic

# 1. Fetch context from Context Engine
response = httpx.post("http://localhost:8080/v1/query-context", json={
    "project_name": "my-app",
    "query": "how does authentication work"
})
ctx = response.json()

# 2. Read the actual file contents for top matches
file_contents = ""
for f in ctx["files"]:
    with open(f["path"]) as fp:
        file_contents += f"\n\n### {f['path']}\n```go\n{fp.read()}\n```"

# 3. Build the prompt with context injected
system_prompt = f"""You are a senior engineer helping with a Go codebase.
Use ONLY the context below to answer questions.

<project_context>
{file_contents}
</project_context>
"""

# 4. Call Claude
client = anthropic.Anthropic()
message = client.messages.create(
    model="claude-sonnet-4-6",
    max_tokens=1024,
    system=system_prompt,
    messages=[
        {"role": "user", "content": "How does authentication work in this project?"}
    ]
)
print(message.content[0].text)
```

---

### Shell One-Liner (quick debugging)

```bash
# Fetch context and pipe into a Claude prompt via CLI
CONTEXT=$(curl -s -X POST http://localhost:8080/v1/query-context \
  -H "Content-Type: application/json" \
  -d '{"project_name":"my-app","query":"how does TAT calculation work"}')

echo "Project context: $CONTEXT

Question: How does TAT calculation work?" | claude
```

---

### Prompt Template (copy-paste)

Use this structure when manually pasting context into Claude:

```
You are a senior engineer. Answer using ONLY the project context provided.

<project_context>
### internal/orders/tat.go
[paste file content here]

### internal/models/order.go
[paste file content here]
</project_context>

Question: How does TAT calculation work?
```

---

### Best Practices for Claude Prompts with Context

| Practice | Why |
|---|---|
| Wrap context in `<project_context>` tags | Keeps it clearly separated from the question |
| Include only top 3–5 files | Avoids overwhelming the context window |
| Paste function signatures + body, not just names | Claude needs the actual logic to reason |
| State the project language in system prompt | Helps Claude apply correct idioms |
| Ask one specific question per prompt | More focused = more accurate answer |

---

## Multiple Projects

You can ingest as many projects as you want — each is isolated by `project_name`:

```bash
./bin/context-engine ingest ./service-a --project service-a
./bin/context-engine ingest ./service-b --project service-b
./bin/context-engine ingest ./frontend   --project frontend

# Query each independently
./bin/context-engine query "how does login work" --project frontend
./bin/context-engine query "how does payment work" --project service-b
```

---

## What Gets Ingested

| What           | How                                              |
|----------------|--------------------------------------------------|
| Files          | All source files parsed via AST                 |
| Functions      | Function names, signatures, and summaries        |
| Imports        | Dependency relationships between files           |
| Embeddings     | Vector representation of each file/function     |
| Graph          | Neo4j nodes: `Project → File → Function`        |

---

## Language Support

| Language | Status        |
|----------|---------------|
| Go       | Supported     |
| Others   | Add a parser in `internal/parser/` |

---

## Troubleshooting

| Problem                        | Fix                                                        |
|--------------------------------|------------------------------------------------------------|
| Server not reachable           | Run `./bin/context-engine serve` first                    |
| Empty results after ingest     | Check the path is correct and contains `.go` files        |
| Port conflict                  | Set `SERVER_PORT=9090` env var before starting the server |
| Neo4j / Postgres not running   | Run `./scripts/setup.sh` to start infrastructure          |
