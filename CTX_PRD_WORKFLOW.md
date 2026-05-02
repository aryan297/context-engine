# CTX + PRD Workflow — Minimize LLM Token Usage

How to use `ctx` (Context Engine) to turn a PRD feature description into a focused, low-token LLM prompt — instead of pasting your entire codebase.

---

## Why This Matters

| Approach | Tokens used |
|----------|-------------|
| Paste all source files raw | ~85,000–100,000 |
| `ctx query` result for one PRD feature | ~1,000–3,000 |
| **Savings** | **~96% reduction** |

The entire codebase is indexed once server-side. Every query returns only the files and functions that are relevant to your feature — nothing else enters the LLM context window.

---

## The 4-Step Workflow

### Step 1 — Ingest the codebase once

```bash
ctx ingest ./ride_hailing --project ride_hailing
```

This walks every `.go`, `.ts`, `.tsx`, and `.java` file and builds an index:
- File summaries
- Function names and signatures
- Import relationships

The index lives in PostgreSQL (vectors) and Neo4j (graph). It is **not** in your LLM context.

> Re-ingest any time you add new files or make significant structural changes.

---

### Step 2 — Write the PRD feature as a query string

Take your PRD feature description and distill it into a short, keyword-rich query:

```bash
# PRD Feature: "Add scheduled rides — users can book a ride for a future time"
ctx query "scheduled ride booking future time" --project ride_hailing
```

`ctx` returns only the relevant blocks — the files and functions that match:

```json
{
  "files": [
    { "path": "ride/model.go",     "score": 0.94 },
    { "path": "ride/handler.go",   "score": 0.91 },
    { "path": "ride/service.go",   "score": 0.89 },
    { "path": "pricing/service.go","score": 0.76 }
  ],
  "functions": [
    { "name": "CreateRide",    "file": "ride/handler.go",   "score": 0.92 },
    { "name": "BookRide",      "file": "ride/service.go",   "score": 0.90 },
    { "name": "CalculateFare", "file": "pricing/service.go","score": 0.78 }
  ]
}
```

---

### Step 3 — Feed only those blocks to the LLM

Paste the relevant file contents + the feature spec — nothing else:

```
PRD Feature: Scheduled rides
Users should be able to book a ride for a future time (up to 7 days ahead).

Context from ctx:
  - ride/model.go: Ride struct with status, pickup_time fields
  - ride/handler.go: CreateRide(c *gin.Context) — current booking handler
  - ride/service.go: BookRide(req BookRideRequest) (*Ride, error) — business logic
  - pricing/service.go: CalculateFare(ride *Ride) — may need a scheduled surcharge

Task:
Add a `scheduled_at` field to the Ride model.
Validate that scheduled_at is between 15 minutes and 7 days from now.
Store it in the DB and return it in the API response.
```

---

### Step 4 — Implement, then re-ingest

After implementing the feature:

```bash
# Re-index so future queries reflect the new code
ctx ingest ./ride_hailing --project ride_hailing
```

---

## Example Queries for Common PRD Features

### Rides

```bash
ctx query "create ride booking flow"              --project ride_hailing
ctx query "scheduled ride future time booking"    --project ride_hailing
ctx query "ride status update trip lifecycle"     --project ride_hailing
ctx query "cancel ride refund policy"             --project ride_hailing
ctx query "hourly rental ride package booking"    --project ride_hailing
```

### Pricing & Fare

```bash
ctx query "fare calculation surge pricing"        --project ride_hailing
ctx query "coupon discount apply booking"         --project ride_hailing
ctx query "wallet balance deduction payment"      --project ride_hailing
ctx query "toll charges extra fare breakdown"     --project ride_hailing
```

### Driver

```bash
ctx query "driver matching assignment nearest"    --project ride_hailing
ctx query "driver earnings payout settlement"     --project ride_hailing
ctx query "driver status online offline toggle"   --project ride_hailing
ctx query "driver rating review score"            --project ride_hailing
```

### Authentication & OTP

```bash
ctx query "OTP verification login signup"         --project ride_hailing
ctx query "JWT token refresh authentication"      --project ride_hailing
ctx query "session management logout"             --project ride_hailing
```

### Notifications

```bash
ctx query "push notification ride update"         --project ride_hailing
ctx query "SMS alert driver assigned"             --project ride_hailing
```

---

## When to Re-Ingest

| Situation | Re-ingest? |
|-----------|-----------|
| Added new `.go` / `.ts` / `.java` files | Yes |
| Renamed a function or file | Yes |
| Only changed logic inside an existing function | Optional |
| Only changed tests | No |
| Only changed comments or docs | No |

```bash
# Always safe to run — it replaces the old index
ctx ingest ./your-project --project your-project
```

---

## PRD File Template

Save this as `PRD_FEATURE.md` alongside your codebase. Fill in one section per feature, then use the "Query string" as your `ctx query` input.

```markdown
## Feature: <Feature Name>

**Goal:**
<One sentence — what problem does this solve?>

**User story:**
As a <user type>, I want to <action> so that <outcome>.

**Inputs:**
- <field>: <type> — <description>
- <field>: <type> — <description>

**Outputs:**
- <what the API returns or what changes in the DB>

**Business rules:**
- <rule 1>
- <rule 2>
- <edge case>

**Constraints:**
- Latency: < X ms
- Scale: X requests/sec
- Dependencies: <service or table name>

**ctx query string:**
"<3–6 keywords that describe this feature>"

**Relevant files (filled after ctx query):**
- [ ] <file path> — <why it's relevant>
- [ ] <file path> — <why it's relevant>
```

---

## Full Example — Scheduled Rides Feature

### PRD

```markdown
## Feature: Scheduled Rides

**Goal:**
Allow riders to pre-book a ride for a future date and time.

**User story:**
As a rider, I want to schedule a ride up to 7 days in advance so that
I don't need to be online at the exact time of pickup.

**Inputs:**
- pickup_location: LatLng
- dropoff_location: LatLng
- scheduled_at: timestamp (UTC) — must be 15 min to 7 days from now
- ride_type: string (economy | premium)

**Outputs:**
- ride_id, status: "scheduled", scheduled_at, estimated_fare

**Business rules:**
- scheduled_at must be >= now + 15 minutes
- scheduled_at must be <= now + 7 days
- Driver is NOT assigned at booking time — assigned 10 min before pickup
- Cancellation allowed up to 5 min before scheduled_at

**Constraints:**
- Latency: < 300ms
- No double-booking for same rider within ±30 min window

**ctx query string:**
"scheduled ride booking future time"
```

### ctx query

```bash
ctx query "scheduled ride booking future time" --project ride_hailing
```

### LLM Prompt (after ctx)

```
You are a senior Go engineer.

PRD Feature: Scheduled rides (see rules above)

<project_context>
### ride/model.go
[paste content]

### ride/service.go
[paste content]

### ride/handler.go
[paste content]
</project_context>

Task:
1. Add `scheduled_at *time.Time` field to the Ride model and DB migration.
2. In BookRide service: validate the 15-min/7-day window, set status to "scheduled".
3. In CreateRide handler: accept and pass through scheduled_at from request body.

Output: Production-ready Go code only. No explanations.
```

---

## Quick Reference

```bash
# Ingest / re-ingest
ctx ingest ./project --project project-name

# Query for a PRD feature
ctx query "feature keywords" --project project-name

# List all ingested projects
curl http://localhost:8080/v1/projects

# Check server health
curl http://localhost:8080/health
```

---

## Tips

- **Keywords beat sentences** — `"scheduled ride booking future time"` outperforms `"I want to add a feature where users can schedule rides"`
- **One feature per query** — mixing two features in one query dilutes the results
- **Score < 0.7 = likely irrelevant** — ignore low-score files when building your prompt
- **Ingest before a sprint** — run `ctx ingest` at the start of each sprint so the index reflects the latest code
