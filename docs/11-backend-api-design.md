# 11 — Backend API design (Go)

The Go API service (`services/api/`) is the single source of truth for every client (web-client, web-admin, mobile, public consumers). This document captures its structural decisions.

## Philosophy

1. **Boring and explicit.** Plain SQL over ORMs. Handlers call services call repositories. No framework magic.
2. **OpenAPI first.** The contract is defined in `services/api/openapi.yaml`; handlers are validated against it; TypeScript and Go clients are generated from it.
3. **Domain-oriented packages.** One package per bounded context. Shared infra lives under `internal/platform`.
4. **Fail loud in tests, gracefully in prod.** Panics in production are logged with stacks + trace IDs and returned as 500; in tests they fail the test.
5. **Context everywhere.** Every function that does I/O takes `context.Context` as its first argument. Timeouts propagate.

## Chosen libraries

| Purpose | Choice | Reason |
|---|---|---|
| HTTP router | **`github.com/go-chi/chi/v5`** | Idiomatic stdlib style, composable middleware, tiny surface. |
| Postgres driver | **`github.com/jackc/pgx/v5`** | Native protocol; best throughput on Go. |
| SQL code-gen | **`github.com/sqlc-dev/sqlc`** | Compile-time verification of queries + generated type-safe Go. |
| Migrations | **`github.com/pressly/goose/v3`** | Plain `.sql` files already drafted in `packages/schema/migrations/`. |
| Validation | **`github.com/go-playground/validator/v10`** | Struct-tag validation for request DTOs. |
| Config | **`github.com/caarlos0/env/v11`** + `.env` | Env-first, 12-factor. |
| Logging | `log/slog` (stdlib) | Structured logging, JSON in prod, pretty in dev. |
| Observability | **`go.opentelemetry.io/otel`** + OTLP exporter | Traces + metrics; Grafana Cloud receiver. |
| Auth | Self-hosted JWT + refresh, phone OTP flow via Dialog/Mobitel | See `internal/auth`. |
| Testing | stdlib + **`testcontainers-go`** for Postgres | Integration tests against real Postgres via Docker. |
| HTTP client | stdlib `net/http` + `httptrace` | Upstream API calls (weather, SMS). |
| Swagger UI | **`swaggest/swgui`** served at `/docs` in non-prod | Devs browse the OpenAPI live. |

## Repository layout (inside `services/api/`)

```
services/api/
├── cmd/
│   └── api/
│       └── main.go            # Wires config, DB, HTTP server, graceful shutdown
├── internal/
│   ├── platform/
│   │   ├── config/            # env loading
│   │   ├── db/                # pgx pool + sqlc Queries + migration runner
│   │   ├── httpx/              # shared middleware (auth, logging, recover, CORS, rate limit)
│   │   ├── otel/              # tracing + metrics bootstrap
│   │   └── errors/            # typed API errors + RFC 7807 problem+json rendering
│   ├── auth/                  # OTP flow, JWT issue/verify, refresh tokens, RBAC
│   ├── users/                 # user + farm + plot management
│   ├── crops/
│   ├── varieties/
│   ├── diseases/
│   ├── pests/
│   ├── remedies/
│   ├── recommender/           # location → crop recommendations
│   ├── scans/                 # disease-scan records + ML service client
│   ├── calendar/              # planting calendar + reminders
│   ├── feed/                  # community posts + comments
│   ├── marketplace/           # listings + transactions
│   ├── weather/               # Met Dept / Open-Meteo client
│   └── admin/                 # endpoints only admin portal can call
├── openapi.yaml               # Source of truth for the HTTP contract
├── sqlc.yaml                  # sqlc config
├── queries/                   # *.sql files grouped by domain
│   ├── crops.sql
│   ├── diseases.sql
│   └── ...
├── migrations/                # symlink to ../../packages/schema/migrations
├── Dockerfile
└── Makefile
```

Each domain package follows:

```
internal/crops/
├── handlers.go     # HTTP handlers (depend on service)
├── service.go      # business logic (depends on repo + other domain services)
├── repo.go         # thin layer over sqlc-generated queries (depends on db)
├── dto.go          # request/response DTOs (validated with go-playground/validator)
└── service_test.go
```

## Request lifecycle

```
HTTP request
  → chi router
    → middleware stack: recover, requestID, OTel span, structured log, auth, rate limit
      → domain handler (parses + validates DTO)
        → domain service (business logic)
          → domain repo (sqlc-generated query)
            → pgx pool → Postgres
        ← returns domain object
      ← handler marshals response
    ← middleware records metrics + closes span
  ← HTTP response
```

## Handler style

```go
// internal/crops/handlers.go
func (h *Handler) getCrop(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    slug := chi.URLParam(r, "slug")

    crop, err := h.svc.Get(ctx, slug)
    if err != nil {
        errors.Render(w, r, err) // maps ErrNotFound → 404, etc.
        return
    }

    httpx.JSON(w, http.StatusOK, crop)
}
```

- No fatals. No `panic`. No global state.
- Request/response shapes defined as DTOs in `dto.go` and referenced from `openapi.yaml`.

## Service style

```go
// internal/crops/service.go
type Service struct {
    repo   Repository
    search SearchIndex
    log    *slog.Logger
    tracer trace.Tracer
}

func (s *Service) Get(ctx context.Context, slug string) (Crop, error) {
    ctx, span := s.tracer.Start(ctx, "crops.Service.Get")
    defer span.End()

    c, err := s.repo.GetBySlug(ctx, slug)
    if errors.Is(err, sql.ErrNoRows) {
        return Crop{}, apierrors.NotFound("crop", slug)
    }
    return c, err
}
```

## Repo style (sqlc-generated)

Query file:

```sql
-- name: GetCropBySlug :one
SELECT * FROM crop
WHERE slug = $1 AND status = 'published'
LIMIT 1;
```

Generated Go:

```go
func (q *Queries) GetCropBySlug(ctx context.Context, slug string) (Crop, error)
```

Repo wraps it for dependency injection and test swapping:

```go
type Repository interface {
    GetBySlug(ctx context.Context, slug string) (Crop, error)
}
```

## Versioning

- URI versioning: `/v1/...`. Breaking changes bump to `/v2/...`; previous version supported for 6 months with a `Deprecation` header.
- OpenAPI spec versioned in git; each release tags `openapi-vX.Y.Z` alongside the Docker image.

## Error envelope

We emit **RFC 7807 problem+json** for all errors:

```json
{
  "type": "https://goyama.lk/errors/crop-not-found",
  "title": "Crop not found",
  "status": 404,
  "detail": "No crop with slug 'wheat-abc'",
  "instance": "/v1/crops/wheat-abc",
  "request_id": "01HY..."
}
```

## Pagination

- **Cursor-based** for list endpoints with mutable ordering (feed, listings, scans).
- **Offset-based** allowed only for admin tables where stability isn't critical.
- Page size default 20, max 100.

## Search

`/v1/search?q=...&locale=si` fans out to Meilisearch indexes for crops / diseases / pests / posts. Results ranked with locale-aware tokenisation; the service enforces a strict 500 ms timeout.

## Recommender

`POST /v1/recommend/crops` accepts `(lat, lng, plot_area_m2, soil?, water?, effort, season, goals[])` and returns a ranked list with per-factor explanations (see [docs/05-features.md](./05-features.md)). Implementation is rule-based + weighted scoring first; upgrades to a learned reranker once we have enough usage data.

Behind the scenes it:

1. Reverse-geocodes `(lat, lng)` → `admin_area`, `aez`, `soil`, `elevation`, `rainfall_normal` using the Geobop API + local PostGIS fallback.
2. Filters candidate crops by hard constraints (AEZ suitability class ≠ N, elevation/pH/water match).
3. Scores soft factors: S1/S2/S3 weight, market-price trend, effort match, disease-pressure penalty (recent scans within 50 km), user's prior crops (avoid back-to-back same family).
4. Returns the top N with the factor breakdown so the UI can show "why".

## Scans endpoint (ML glue)

```
POST   /v1/scans                → upload image, returns scan_id + top-3 predictions
GET    /v1/scans/{id}           → scan detail with predictions + reviewed disease
POST   /v1/scans/{id}/feedback  → user flags "this doesn't look right"
```

The handler uploads to R2, then calls the Python `ml-inference` sidecar via HTTP (`http://ml-inference:8081/predict`) with a short-lived presigned URL. Confidence thresholds route low-confidence scans to the agronomist review queue (see `internal/scans/service.go`).

## Feed + community endpoints

```
GET    /v1/feed                 → aggregated feed (chronological + "for you")
POST   /v1/posts                → create post (text / photo / question)
POST   /v1/posts/{id}/comments  → comment
POST   /v1/posts/{id}/react     → reactions
GET    /v1/posts/nearby         → location-scoped posts for map overlay
```

Social feed events are served via **SSE** for in-app refresh; WebSocket reserved for marketplace chat.

## Marketplace endpoints

```
GET    /v1/listings             → filter by crop / district / radius
POST   /v1/listings             → create listing (verified users only)
POST   /v1/listings/{id}/contact → express interest / open chat thread
GET    /v1/prices/daily         → HARTI / DEC market prices (read-through cache)
```

## Map + geospatial endpoints

```
GET    /v1/geo/reverse?lat&lng             → admin + AEZ + soil + rainfall normal
GET    /v1/aez.geojson                     → AEZ polygons for overlay
GET    /v1/farms/nearby?lat&lng&radius_km  → fuzzed farmer pins
GET    /v1/disease-pressure.geojson?crop   → heatmap layer
```

Responses set long `Cache-Control` on geodata; CF edges cache and serve.

## Rate limiting

- Anonymous: 60 req/min per IP.
- Authenticated: 300 req/min per user.
- Scan uploads: 20/hour per user.
- Marketplace creates: 10/hour per user.

All enforced at Cloudflare edge first, then reaffirmed in the Go API with a Redis-backed token bucket (`internal/platform/httpx/ratelimit.go`).

## Migrations and schema drift

- `packages/schema/migrations/` is the canonical migration set (already has `0001_init.sql` and `0002_graph.sql`).
- `services/api` runs migrations in CI on every preview + staging DB.
- Each migration is paired with a rollback SQL where feasible.
- `sqlc` is re-generated on every schema change; CI fails if the generated code differs from the committed version.

## Testing strategy

| Level | Tool | What |
|---|---|---|
| Unit | stdlib + `testify/assert` | Pure service logic, utils. |
| DB integration | `testcontainers-go` | Each test package spins up a Postgres container + runs migrations + runs repo tests against it. |
| API integration | `httptest.Server` + the real router | End-to-end handler tests with mocked external services. |
| Contract | `ogen-go/ogen` or schemathesis | OpenAPI spec vs. actual handler responses. |
| Load | `k6` | Smoke at 50 rps, soak at 100 rps per instance. |

CI runs all of them on every PR. Coverage target: **≥ 80% lines** on domain packages.

## Deployment

- `services/api` containerised as a minimal **distroless** image (~20 MB).
- Deployed to Fly.io (primary) with 2 instances per region for HA.
- Postgres: Neon project with branching, read replicas added under load.
- Meilisearch + Redis: managed via Fly.io volumes or a managed provider.
- R2 bucket + Cloudflare Images in front for all media.
- DNS + WAF + TLS termination at Cloudflare.

## Observability

- **Traces** — every request generates an OTel span with `request_id`, `user_id`, `route`. Exported to Grafana Tempo via OTLP.
- **Metrics** — RED (rate / errors / duration) per route; p50/p95/p99 histograms.
- **Logs** — structured via `slog`, shipped to Grafana Loki.
- **Error tracking** — Sentry on client apps; Go errors surfaced via OTel events.

## Security posture

- Every endpoint declares required role in the OpenAPI spec; a middleware enforces it at runtime.
- Secrets in Fly secrets / Vault; never in containers.
- **`govulncheck`** in CI.
- Dependency updates via **Renovate** weekly.
- Admin endpoints require TOTP in addition to the regular JWT; IP allowlist optional.

## Open questions

- Do we want a thin GraphQL layer for the mobile home screen to reduce round trips, or stick to a custom aggregate REST endpoint? Current recommendation: custom `/v1/home` aggregate, avoid GraphQL operational cost.
- Do we self-host Meilisearch or use Meilisearch Cloud? Start self-hosted on a single VM; migrate to cloud when write volume warrants.
- SSE vs. short-polling for the feed? SSE default; polling fallback for environments that strip SSE.
