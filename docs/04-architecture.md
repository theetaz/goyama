# 04 — System architecture

## Guiding constraints

- **Data-driven, not hardcoded** — all content served from DB, editable via admin portal.
- **Multi-client** — two separate web SPAs (client + admin), a React Native app, and a read-only public API all share one Go backend.
- **Offline-tolerant mobile** — reference content and disease model cacheable on device.
- **Sri Lanka deployment reality** — intermittent connectivity, low-RAM Android devices, hosting cost matters.
- **Split client and admin surfaces** — different themes, different auth, different attack surfaces, deployed independently.

## Tech stack (locked)

| Layer | Choice | Notes |
|---|---|---|
| **Backend API** | **Go 1.22+** (stdlib + `chi` router) | One statically-linked binary per service. Low memory. Easy operational model. |
| **DB driver** | **pgx v5** | Native Postgres protocol, no ORM overhead. |
| **Queries** | **sqlc** | Generates type-safe Go code from plain SQL. No runtime query building. |
| **Migrations** | **goose** or **dbmate** | Plain SQL migrations (already drafted in `packages/schema/migrations/`). |
| **DB** | **Postgres 16 + PostGIS + pgvector** (+ Apache AGE optional) | Relational + geospatial + vector embeddings in one engine. |
| **Search** | **Meilisearch** | Multilingual tokenization (SI / TA / EN). |
| **Object storage** | **Cloudflare R2** (S3-compatible) | Zero egress — ideal for image-heavy serving to Sri Lanka. |
| **CDN / image transforms** | **Cloudflare Images** or **imgproxy** in front of R2 | Responsive image variants (WebP/AVIF) per device. |
| **Queues / cache** | **Redis** (managed) | Rate limiting, async jobs (mealybug-scan review queue, email/SMS notifications). |
| **Auth** | Go service owns JWT issuance; phone-OTP via **Dialog / Mobitel** SMS gateway; email+TOTP for admins. | No Clerk / Supabase Auth — we own identity. |
| **Web client** (farmer-facing) | **Vite + React 18 + TypeScript + TanStack Query + TanStack Router + Shadcn UI + Tailwind** | SPA deployed to Cloudflare Pages / Vercel. Custom agri theme. |
| **Web admin** (agronomist + staff) | Same stack, **separate app**, separate subdomain (admin.cropdoc.lk), different theme (denser, data-heavy). | Independent deployment, auth, and RBAC boundary. |
| **Mobile** | **Expo (managed workflow) + React Native + TypeScript + TanStack Query + NativeWind** | EAS builds for Android + iOS. Shared design tokens with web. |
| **Maps** | **MapLibre GL JS** (web) + **@maplibre/maplibre-react-native** + self-hosted vector tiles (PMTiles/OpenMapTiles) | Avoid Mapbox licensing. AEZ, soil, plot, farmer-locations overlays. |
| **ML serving** | Go API owns the endpoint; model inference via a sidecar **ONNX Runtime** HTTP service or **Python FastAPI** shim | On-device model (TFLite / Core ML) in the Expo app. |
| **Observability** | **OpenTelemetry → Grafana Cloud** (traces/metrics/logs) + **Sentry** (client errors) | Free tiers cover pilot. |
| **CI/CD** | **GitHub Actions** | Matrix: Go test + lint → Vite build → Expo EAS → DB migration preview. |
| **Infra** | **Fly.io** or **Hetzner** for Go services; **Neon** or **Supabase Postgres** for DB; **Cloudflare** for edge + R2 + DNS | Avoid AWS complexity. |

## Repository layout (monorepo)

```
cropdoc/
├── apps/
│   ├── web-client/           # Vite + React client SPA (farmer-facing)
│   ├── web-admin/            # Vite + React admin SPA (agronomist + staff)
│   └── mobile/               # Expo React Native
├── services/
│   ├── api/                  # Go API service (public + authed)
│   ├── worker/               # Go background workers (notifications, indexer sync)
│   └── ml-inference/         # Python FastAPI + ONNX Runtime (disease scanner)
├── packages/
│   ├── schema/               # JSON Schema (already built) — source of truth
│   ├── types-ts/             # TypeScript types generated from schema
│   ├── types-go/             # Go structs generated from schema
│   ├── ui/                   # Shared Shadcn components between web-client + web-admin
│   ├── design-tokens/        # Colors, typography, spacing — shared web + mobile
│   └── i18n/                 # Locale files (si, ta, en) + message loading utils
├── pipelines/                # Python ingestion pipelines (already built)
├── corpus/                   # Data (already built)
└── docs/                     # Planning docs
```

Package management: **pnpm workspaces** for JS/TS; **Go workspaces** (`go.work`) for services; **uv** for Python.

## High-level topology

```
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│  apps/mobile     │  │ apps/web-client  │  │ apps/web-admin   │
│  (Expo RN)       │  │ (Vite + React)   │  │ (Vite + React)   │
│  cropdoc.lk app  │  │ app.cropdoc.lk   │  │ admin.cropdoc.lk │
└────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘
         │                     │                     │
         │  HTTPS + JWT / OTP  │                     │ HTTPS + TOTP
         └──────────┬──────────┴─────────────────────┘
                    │
            ┌───────▼────────┐
            │   API gateway  │   Cloudflare (TLS, WAF, rate limit)
            └───────┬────────┘
                    │
   ┌────────────────┼──────────────────┬──────────────────┐
   │                │                  │                  │
┌──▼────────┐ ┌─────▼───────┐  ┌───────▼─────────┐  ┌─────▼───────────┐
│ services/ │ │ services/   │  │ services/       │  │ pipelines/      │
│ api (Go)  │ │ worker (Go) │  │ ml-inference    │  │ (Python crawl + │
│  chi,     │ │  notifs,    │  │  (Py FastAPI +  │  │  extract + LLM  │
│  pgx,sqlc │ │  indexing   │  │   ONNX Runtime) │  │  agronomist CMS)│
└──┬────────┘ └─────┬───────┘  └───────┬─────────┘  └──────┬──────────┘
   │                │                  │                   │
   └────────────────┼──────────────────┴───────────────────┘
                    │
            ┌───────┴────────────────────────────────────┐
            │                                            │
      ┌─────▼──────┐   ┌────────────┐   ┌────────────┐  ┌▼─────────────┐
      │ Postgres + │   │ Meilisearch│   │ R2 object  │  │ Redis        │
      │ PostGIS +  │   │            │   │ storage    │  │ (cache,      │
      │ pgvector   │   │            │   │ + CF Images│  │  queues)     │
      └────────────┘   └────────────┘   └────────────┘  └──────────────┘
```

## Go API service design — the short version

See [docs/11-backend-api-design.md](./11-backend-api-design.md) for detail. Key choices:

- **`chi` router** over Gin/Echo — minimal, stdlib-idiomatic, composable middleware.
- **sqlc + pgx** instead of an ORM — hand-write SQL, let sqlc generate type-safe Go.
- **Domain-oriented packages**: `internal/crops`, `internal/diseases`, `internal/recommender`, `internal/scans`, etc. Each owns handler, service, and repo layers; shared infra in `internal/platform`.
- **OpenAPI 3.1 spec** authored in `services/api/openapi.yaml` — drives contract tests and TS client generation for the web+mobile apps.
- **Context-aware everything** — every handler takes `context.Context`; DB + HTTP clients propagate cancellation.
- **Graceful shutdown + structured logging** (`slog`) with OTel traces.

## Client web app architecture (`apps/web-client`)

- **Vite 5** dev + build; **React 18** + TypeScript strict.
- **TanStack Router** (type-safe file-based routing) and **TanStack Query** (data fetching, caching, background refetch).
- **Shadcn UI** + **Tailwind** + **Radix primitives** for accessible components.
- **Custom agri theme** — earthy greens/ambers, high contrast for sunlight reading, big tap targets, Sinhala/Tamil font stacks. See [docs/10-ui-ux-principles.md](./10-ui-ux-principles.md).
- **TanStack Query devtools** in dev; **TanStack Table** for admin-style tables.
- **i18next** (or **Lingui**) driving `packages/i18n` — SI/TA/EN runtime switching.
- **MapLibre GL JS** via `react-map-gl/maplibre` — AEZ overlays, user-farm pins.
- **Vitest + React Testing Library** for component tests; **Playwright** for critical-path e2e.
- PWA manifest for home-screen install; offline read of cached content via a service worker.

## Admin web app architecture (`apps/web-admin`)

- Same stack but **separate app and deployment**. Different Shadcn theme preset (denser, more neutral, optimized for tables and long review sessions).
- **Data-heavy views**: review queue, edit records, approve/reject, moderation dashboards, broadcast advisories.
- **RBAC gate** at the app shell — unauthenticated users redirected to login; viewer/reviewer/admin roles gate specific routes.
- Dedicated subdomain `admin.cropdoc.lk` with stricter CSP, shorter JWT TTL, and IP allowlist optional for high-privilege operations.

## Mobile app architecture (`apps/mobile`)

- **Expo managed workflow** — EAS Build for Android + iOS, EAS Update for OTA JS pushes.
- **React Native + TypeScript**, **Expo Router**, **TanStack Query**, **NativeWind** (Tailwind for RN).
- **Local store**: SQLite via `op-sqlite` or `expo-sqlite` mirroring a trimmed subset of the corpus (top 30 crops, all diseases, user's plots).
- **Delta sync** keyed on `updated_at` cursors; tombstones for deletes.
- **Maps**: `@maplibre/maplibre-react-native` — same MapLibre style JSON as web.
- **Camera + disease scanner**: `expo-camera` + on-device TFLite model via `react-native-fast-tflite`.
- **Image cache**: filesystem-backed LRU, user-capped size.
- **Push**: Expo Notifications + FCM.
- **SMS fallback**: Dialog/Mobitel SMS gateway for users with unreliable data connections.

## Design-token sharing

- `packages/design-tokens` emits: CSS variables for web + a JSON/TS file consumed by NativeWind for mobile.
- Colors defined in OKLCH for perceptual consistency; dark mode + high-contrast "sunlight" mode both mandatory.
- Typography: Noto Sans Sinhala + Noto Sans Tamil + Inter (Latin) loaded on web; bundled with the Expo app for mobile.
- Icon set: **Lucide** + custom agri glyphs (paddy stalk, coconut, cinnamon quill, etc.).

## API shape

- **REST (`/v1/...`)** is the primary shape. Reads cached via TanStack Query.
- **Server-sent events (SSE)** for live advisories, community feed updates.
- **WebSocket** only where bidirectional needed (marketplace chat).
- Public read-only API (API-key gated) for researchers and third-party apps.
- Every endpoint documented in the OpenAPI spec; TS client generated into `packages/types-ts`.

## Auth

- **Phone OTP** for end users (Dialog/Mobitel SMS). JWT with short access tokens + refresh tokens.
- **Email + TOTP** for agronomists + admins.
- **RBAC roles**: `anonymous`, `user`, `verified_farmer`, `agronomist`, `moderator`, `admin`.
- **Row-Level Security** in Postgres for UGC (listings, posts, scans, farms).
- JWT signing key in a secrets manager; rotation quarterly.

## Data residency + privacy

- Primary DB hosted in-region where possible (Singapore / India). User PII (phone, precise plot polygon) never leaves the primary region.
- Aggregate/opt-in signals (disease heatmap at GN-division grain) can propagate to read replicas and CDN.
- Right-to-erasure implemented: soft-delete + 30-day retention before hard delete.

## Security

- TLS everywhere (Cloudflare + Let's Encrypt). HSTS preload. CSP with strict sources.
- Rate limiting at the edge per IP + per authenticated user.
- Image uploads: ClamAV virus scan + NSFW classifier before publish.
- Secrets via cloud KMS (no `.env` on production hosts).
- Dependency audit in CI: `govulncheck`, `pnpm audit`, `trivy` for container images.

## Environments

- `dev` — local Docker Compose + Postgres + Meilisearch + Redis + R2 stub.
- `staging` — full prod mirror, synthetic + imported demo data, used by agronomists for review training.
- `prod` — pilot districts first, then nationwide.

Every PR spins up an ephemeral **preview environment** with a branch DB (Neon branching makes this cheap).

## Cost posture

- Pilot infra target: **< $200/mo** end-to-end (Cloudflare free + Neon free + Fly.io small + Meilisearch free tier).
- Scale horizontally per service after product-market fit; Postgres stays one instance until we see measurable load.
- Image storage dominates; aggressive resizing to WebP/AVIF via Cloudflare Images, lazy-loading on mobile.
