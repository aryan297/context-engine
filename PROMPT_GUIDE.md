# LLM Prompt Engineering Guide

How to write better prompts and reduce token usage when working with any LLM (Claude, GPT, Gemini, Cursor, Copilot).

---

## 1. Convert PRD / Confluence Doc → Structured Blocks

Instead of pasting a raw PRD, extract it into focused sections:

```
[FEATURE]
What are we building?
e.g. "Driver matching system that assigns the nearest available driver within 5 seconds"

[GOAL]
Why does it exist?
e.g. "Reduce rider wait time and improve driver utilization rate"

[INPUTS / OUTPUTS]
What goes in / comes out?
Input:  rider location (lat/lng), ride type (economy/premium)
Output: assigned driver ID, ETA in seconds

[CONSTRAINTS]
Scale, latency, cost, edge cases
- 10,000 concurrent requests
- Match must complete in < 2s
- Fallback if no driver within 5 km
- Driver must not be on an active trip

[EXISTING SYSTEM]
Tech stack, APIs, DB
- Language: Go, Framework: Gin
- DB: PostgreSQL (drivers table), Redis (active driver locations)
- Existing: /v1/create-ride, /v1/driver-status

[TASK]
What do you want from the LLM?
"Design the API and DB schema for the matching service"
```

> The LLM now has everything it needs — no guessing, no back-and-forth.

---

## 2. Progressive Prompting (VERY IMPORTANT)

Never dump everything in one prompt. Break it into steps:

| Step | Prompt | Why |
|------|--------|-----|
| 1 — Architecture | "Design high-level architecture for a ride-matching service" | Get the blueprint right first |
| 2 — Schema | "Now design the DB schema for the matching tables" | Drill into one layer |
| 3 — Code | "Now write the Go service for `POST /assign-driver`" | Implement once design is locked |
| 4 — Review | "Review this for edge cases and production readiness" | Catch issues before shipping |

**Each step builds on the last — smaller prompts = better output + far fewer tokens.**

---

## 3. Reference Compression

Think like: *"What would a senior engineer need in 30 seconds?"*

**Don't paste 5000 words of Confluence.**

❌ Bad:
```
[paste entire Confluence page with background, history, diagrams, meeting notes...]
```

✅ Good:
```
Summary:
- Booking flow: rider requests → driver assigned → trip starts → payment
- Payment via Razorpay (webhook confirms)
- Retry logic is currently missing on payment failure

Key rules:
- Payment must be idempotent (same request ID = no double charge)
- Timeout = 5 seconds on Razorpay call
- On timeout: mark order as "payment_pending", retry via background job
```

---

## 4. Role + Output Format

Assigning a role and specifying output format gives the best results with the fewest tokens.

**Template:**
```
You are a [Role] at [Company type].

Task:
[1-2 sentence description]

Output format:
1. Architecture
2. APIs
3. DB Schema
4. Edge cases
5. Production concerns
```

**Example:**
```
You are a Staff Engineer at a ride-hailing company.

Task:
Design a scalable driver-matching system that assigns the nearest available
driver to a rider within 2 seconds under 10,000 concurrent requests.

Output format:
1. Architecture (components + data flow)
2. APIs (method, path, request/response)
3. DB Schema (tables, indexes)
4. Edge cases
5. Production concerns (monitoring, failure modes)
```

---

## 5. Claude Code / Cursor / Copilot Pattern

Use this exact structure for IDE AI tools — it minimizes token waste and eliminates vague answers:

```markdown
# Context
We are building <feature name>

# Codebase Info
- Language: Go
- Framework: Gin
- DB: PostgreSQL + pgvector
- Cache: Redis

# Task
Implement API:
POST /v1/assign-driver

# Requirements
- Find nearest available driver within 5 km
- Driver must not have an active trip
- Retry up to 3 times with 500ms backoff
- Timeout: 2 seconds total
- Return 503 if no driver found

# Output
Production-ready Go code only. No explanations.
```

---

## 6. Context Engine Integration Pattern

Use **Context Engine** to fetch only the relevant files before building your prompt.
This replaces manually hunting for files across the codebase.

### Step 1 — Query relevant context

```bash
ctx query "how does driver matching work" --project ride-hailing
```

### Step 2 — Build a compressed prompt

Take only the top 3–5 files from the response and inject them:

```
You are a senior Go engineer.

<project_context>
### internal/matching/service.go
[paste file content]

### internal/models/driver.go
[paste file content]
</project_context>

Task: Add retry logic with 3 attempts and 500ms backoff to the matching service.
Output: Only the modified Go code.
```

### Step 3 — Automate it (Python)

```python
import httpx, anthropic

# Fetch context
ctx = httpx.post("http://localhost:8080/v1/query-context", json={
    "project_name": "ride-hailing",
    "query": "driver matching and assignment"
}).json()

# Read top files
file_contents = ""
for f in ctx["files"][:3]:
    with open(f["path"]) as fp:
        file_contents += f"\n\n### {f['path']}\n```go\n{fp.read()}\n```"

# Prompt
client = anthropic.Anthropic()
msg = client.messages.create(
    model="claude-sonnet-4-6",
    max_tokens=2048,
    system=f"""You are a senior Go engineer. Use ONLY the context below.

<project_context>
{file_contents}
</project_context>""",
    messages=[{"role": "user", "content": "Add retry logic to the matching service."}]
)
print(msg.content[0].text)
```

---

## Quick Reference Cheatsheet

| Situation | Pattern to use |
|-----------|----------------|
| Have a PRD / Confluence doc | Structured Blocks (#1) |
| Building something new end-to-end | Progressive Prompting (#2) |
| Large doc, need to compress | Reference Compression (#3) |
| Want precise, formatted output | Role + Output Format (#4) |
| Working in Cursor / Copilot / Claude Code | IDE Pattern (#5) |
| Have an existing codebase | Context Engine + Inject (#6) |

---

## Token-Saving Rules

1. **One question per prompt** — more focused = more accurate + fewer retries
2. **Paste signatures, not entire files** — function names + key logic only
3. **State the language upfront** — avoids generic answers
4. **Ask for code only when needed** — separate design from implementation
5. **Use `<tags>`** to wrap context — helps the LLM separate context from the question
