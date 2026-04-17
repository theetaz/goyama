# 05 — Feature specifications

**Cross-reference**: UI / UX principles are in [docs/10-ui-ux-principles.md](./10-ui-ux-principles.md); backend API contract in [docs/11-backend-api-design.md](./11-backend-api-design.md).

## 5.0 Two web apps and one mobile app

- **`apps/web-client`** — public-facing farmer SPA. Vite + React + Shadcn. Deployed to `app.goyama.lk`.
- **`apps/web-admin`** — agronomist + staff workspace. Same stack, **separate app and theme**. Deployed to `admin.goyama.lk`.
- **`apps/mobile`** — Expo React Native. Shares design tokens and API contract with web-client.

All three consume the Go API in `services/api/`.

## 5.1 Home / onboarding (client app)

**First-run flow** (mobile + web):
1. Pick language (Sinhala / Tamil / English) — big visual selector with native-script labels.
2. Optional: Phone OTP sign-up (required for posting / scanning / marketplace / map-pin visibility).
3. Grant location → reverse-geocode → show *"You're in AEZ IL1a (Intermediate Low-country). Current season: Yala."* with a warm welcome and plant-of-the-month cards.
4. Optional but encouraged: **add a plot** — either drop a pin or draw a polygon on the map, declare soil/water/area. Farmers with plots get much richer personalisation.

**Home screen** for a logged-in user defaults to the **interactive map** (§5.2). Above the map fold:
- Weather today + 7-day forecast badge at the user's plot location.
- Plant-of-the-month cards (swipeable) — 5 crops best suited to plant *this month* at this location.
- Active advisories within 50 km (disease alerts, pest outbreaks, weather warnings).
- "Nearby growers" chip — taps to overlay peer farm pins on the map.
- Bottom nav: **Map · Explore · Scan · Community · Profile** (mobile) / **side nav + map area** (web).

## 5.2 The interactive Sri Lanka map (headline feature)

The map is not a secondary view — it is **the home screen** for users with registered farms. Built on **MapLibre GL** with a custom "Ceylon" style palette (see [docs/10](./10-ui-ux-principles.md)).

**Base style**:
- Muted greens for Wet Zone, warm ochres for Dry Zone, subtle blues for tanks / reservoirs / coastline.
- District and DS-division boundaries at mid-zoom; GN-division at high zoom.
- Self-hosted vector tiles (PMTiles or tiled OpenMapTiles) — no Mapbox licensing.

**Toggleable overlays**:
- **AEZ polygons** (Wet / Intermediate / Dry sub-zones), labelled.
- **Soil groups** (RBE / LHG / NCB / RYP / etc.) — see `corpus/seed/aez/soils.md`.
- **Rainfall isohyets** (Met Dept historical normals).
- **Plant-of-the-month pins** — crops optimal to plant this month at the pinch location. Tap → crop page.
- **Nearby grower pins** — other registered farmers who opted-in to show their location (fuzzed to GN-division grain). Tap → public profile (crops, recent posts, listings).
- **Disease-pressure heatmap** — heatmap coloured by recent confirmed scans per crop within the viewport. Raises early-warning visibility.
- **Agrarian Service Centres** — government extension offices as points of interest.
- **Marketplace listing pins** — optional layer showing fresh produce within a chosen radius.

**Interactions**:
- Tap anywhere on the map → floating card shows `{AEZ, elevation, rainfall normal, current season, top-5 recommended crops}` for that point. Tap "See full recommendations" → `/recommend` route with pre-filled location.
- Search bar on the map: *"What to plant near me?"*, *"Where can I grow rambutan?"*, *"Who grows tea near Galle?"* — results re-centre and filter overlays.
- Pinch-zoom to reveal finer grain (GN-division pins blur to heatmap at far zoom, resolve to individual pins at close zoom).
- Long-press → "Add a plot here" shortcut.

**Bottom sheet** (mobile + web):
- Pulls up from the bottom, persistent at partial height.
- Top section: plant-of-the-month carousel.
- Middle: nearby activity feed (recent posts, listings, scans in the viewport).
- Quick actions: **Scan** (camera), **Add plot**, **Post**.

**Privacy**: farm pins are always fuzzed to GN-division (a 5–20 km² polygon) for all viewers; only the owner sees their precise plot. Users opt in to show their pin at all; opt-out means they're invisible on the map to strangers.

## 5.3 Content calendar — "what to plant this month, where"

A dedicated surface (separate route + prominent home-screen card) that answers the classic farmer question: **"What should I be planting right now on my plot?"**

- Inputs: user's plot location (or a map-tapped point), current date.
- Output: ranked list of crops where the current week falls inside the **ideal sow-window** for the user's AEZ + season (Maha / Yala / perennial).
- Each result shows: crop image, sow-by-date, expected harvest date, AEZ suitability class, expected yield band, effort level.
- Filter chips: "Quick harvest" (< 90 days), "High value" (market-price trend), "Low effort" (hobbyist), "Traditional" (heirloom varieties), "Organic-friendly".
- Source: `cultivation_step` records grouped by `(crop × AEZ × season)` — see corpus schema in [docs/03](./03-domain-model.md).

**Data dependency**: this feature requires the cultivation-step and seasonal-calendar records to be populated from DOA's published calendars. Corpus gap flagged in `docs/08-roadmap.md`.

## 5.4 Location-based recommendation (API detail)

Endpoint: `POST /v1/recommend/crops`

Request:
```json
{
  "lat": 7.2906, "lng": 80.6337,
  "plot_area_m2": 120,
  "soil_type_id": null,
  "water_source": "rainfed",
  "effort_level": "low",
  "season": "auto",
  "goals": ["household", "sale"]
}
```

## 5.2 Crop explorer

A browsable catalog with filters:
- AEZ / zone / district
- Season (Maha / Yala / perennial)
- Category (field / veg / fruit / spice)
- Effort level (for hobbyists)
- Duration (quick < 90 days; mid; long)
- Water requirement

Each crop page includes:
- Names (Sinhala / Tamil / English / scientific).
- Hero image + variety carousel.
- Suitability badge for the user's plot.
- Seasonal calendar visualization (sow → grow → harvest bars).
- Cultivation guide: collapsible steps with inputs, images, embedded curated videos.
- Common pests & diseases with thumbnails.
- Market price chart (last 90 days, nearby DEC).
- Resources: DOA PDFs, YouTube playlist.
- "Start a plan" CTA — creates a personalized planting calendar with reminders.

## 5.3 Location-based recommendation

Endpoint: `POST /v1/recommend/crops`

Request:
```json
{
  "lat": 7.2906, "lng": 80.6337,
  "plot_area_m2": 120,
  "soil_type_id": null,
  "water_source": "rainfed",
  "effort_level": "low",
  "season": "auto",
  "goals": ["household", "sale"]
}
```

Response:
```json
{
  "context": { "aez": "IL1a", "district": "Kandy", "season": "yala", ... },
  "recommendations": [
    {
      "crop_id": 42, "slug": "brinjal",
      "variety_suggestion": { "id": 7, "name": "Thinnaweli Purple" },
      "score": 0.87,
      "explanation": [
        { "factor": "aez_suitability", "class": "S1", "weight": 0.35 },
        { "factor": "market_trend", "direction": "up", "weight": 0.15 },
        { "factor": "effort_match", "value": "low", "weight": 0.15 },
        { "factor": "disease_pressure_nearby", "penalty": 0.05 }
      ],
      "expected_duration_days": 90,
      "expected_yield_kg": 180
    },
    ...
  ]
}
```

The explanation is surfaced in the UI — users should understand *why* a crop is recommended. "Show your working" is non-negotiable for agronomic trust.

## 5.5 Search

Global search across crops, diseases, pests, remedies, varieties, articles. Meilisearch with:
- Sinhala/Tamil/English tokenizers.
- Synonyms (e.g., "Vambatu" ↔ "Brinjal" ↔ "Eggplant").
- Typo tolerance.
- Faceted filters.
- Results ranked with AEZ/season boost when user context is available.

## 5.6 Disease scanner

See doc 06 for ML. The user flow:
1. Tap Scan → camera opens with overlay hints ("Fill the frame with one leaf").
2. User captures 1–3 photos (leaf front, leaf back, whole plant optional).
3. Optional: declare the crop (improves accuracy; pre-filled if user has an active plot).
4. On-device inference runs; shows top-3 with confidence.
5. User taps a result → disease page with remedies (cultural/biological/chemical tabs), resistant varieties, and "similar-looking diseases to rule out."
6. User can flag "this doesn't look right" → routes scan to agronomist review queue; user gets a notification when an expert confirms.
7. Offline: model runs without connection; scan queued for upload when online.

Privacy: scan location is fuzzed to GN-division before being added to the disease-pressure heatmap.

## 5.7 Planting plan & reminders

After a user commits to a crop:
- Generate a step-by-step plan: nursery date → transplant date → fertilizer applications → likely pest windows → harvest window.
- Push + SMS reminders (SMS fallback via Dialog/Mobitel SMS gateway for users with weak data connectivity).
- Each step logs as done/skipped → feeds personal history.

## 5.8 Weather

- Daily forecast per plot (7-day).
- Agro-weather alerts: heavy rain warning, heat stress, frost (up-country), drought stress days, ideal spraying windows (wind + humidity).
- Source: Department of Meteorology API if access granted; else Open-Meteo as baseline.

## 5.9 Market prices

Daily price chart per commodity, per market. Pulls from HARTI / DCS / Economic Centres. Trend indicator (7-day, 30-day). "Sell signal" heuristic when price trend up + user has harvest-ready plot.

## 5.10 Notifications & advisories

Channels: push, in-app, SMS.
Types:
- Disease outbreak in user's district.
- Cultivation step due today.
- Marketplace message / listing interest.
- Market price alert for watched commodities.
- DOA broadcast (sent by agronomist role).

## 5.11 Admin portal (`apps/web-admin`)

**A separate SPA at `admin.goyama.lk` with its own theme, auth, and RBAC.** Not a drawer inside the client app.

### Views

- **Dashboard** — ingestion health (crawl success rates, last-fetch timestamps), review queue depth (scans, listings, posts, translations), corpus coverage heatmap (which crops are missing cultivation-steps / images / Sinhala translations), daily-active users by district.
- **Record editors** — crop / variety / disease / pest / remedy / cultivation-step editors with per-field provenance visible, version history, diff view against the last published version, and `status` flip (draft → in_review → published).
- **Review queue** for disease scans — image, top-3 predictions, user-confirmed label, Grad-CAM overlay; agronomist confirms or overrides; status propagates back to the user.
- **Media library** — images + videos + PDFs with licence tags, attribution, and usage references. CC-BY-SA / CC-BY / public-domain / linked-external-only.
- **Translations console** — side-by-side editor for `(entity, field, locale)` trios; MT drafts editable by reviewers.
- **Advisory broadcaster** — push a timely advisory targeted by AEZ / district / crop / severity. Broadcast over push, in-app, and SMS fallback.
- **User + role management** — role assignment, verified-farmer approval, suspension.
- **Moderation queue** — reports on posts, listings, comments; one-click hide/suspend/ban with reversible audit log.
- **Content-calendar editor** — reviewers edit the monthly planting windows by AEZ × season. Drives §5.3.
- **Bulk import + export** — upload CSV for ranged price data, export records for external review, import corrections from corpus PR reviews.

### RBAC

Roles: `viewer`, `reviewer`, `moderator`, `agronomist`, `admin`.

- `viewer` — read-only; internal/partner access.
- `reviewer` — edit drafts; can flip to `in_review` but not `published`.
- `moderator` — takes action on community reports.
- `agronomist` — can flip records to `published`; required role for chemical-remedy PHI sign-off.
- `admin` — user management, role assignment, system settings.

### Theme

Different Shadcn preset than the client: denser, more neutral (cooler greys), built for tables and long review sessions. Sidebar-first navigation. Keyboard shortcuts for power users. Both light and dark modes.

## 5.12 Community, feed, and social layer (first-class, not Phase 5)

Social activity is not ghettoised in a "Community" tab — it's woven into the map and the crop pages. But there's also a dedicated feed for users who want to scroll and converse.

### Feed route (`/feed`)

- **Two tabs**: "For you" (ranked by relevance to your crops + district + followed users) and "Nearby" (chronological, viewport-scoped).
- **Post types**:
  - **Photo** — 1–6 images, optional crop/disease tags, caption.
  - **Question** — crop/disease-tagged, marked `type=question`, upvotable answers, asker marks a "solution."
  - **Short video** — ≤ 60 s, optional transcript in SI/TA/EN.
  - **Poll** — 2–4 options, 24 h–7 day duration.
  - **Listing re-share** — amplify a marketplace listing to followers.
- **Reactions**: 👍 Like, 🌱 Helpful (counts on questions), 🤝 Same problem here.
- **Follow**: users, crops, districts, diseases. Followed signals surface in "For you."

### Map integration

The map home screen (§5.2) pulls the feed's **nearby posts as pins**. Tap a pin → the post card shows inline, with comments collapsible. This makes the map an ambient feed: scrolling the map is scrolling the village.

### Notifications

Posts from followed users + answers on your questions + mentions + reactions generate push notifications (respecting quiet hours). SMS fallback for users on low-data plans.

### Moderation

Community-drafted rules. Three reports from distinct users ⇒ auto-hide pending moderator review. Moderator dashboard in the admin portal (§5.11).

### Social-media integrations

**Outbound**:
- One-tap re-share to **WhatsApp**, **Facebook**, **X** from any post or listing — copies a deep link back to the app.
- **Save to camera roll** on mobile.

**Inbound**:
- Registered SL agri Facebook Pages / Instagram accounts can be **ingested** as read-only feeds (where the Page grants permission) — shown in the feed with a "from Facebook" badge. Users opt in to following each source.
- Government DOA / Ministry / research-institute accounts (DOA Kandy FB, Govi Mithuru, etc.) surface as pinned official channels.
- YouTube **channel follow** — follow curated DOA / HORDI / farmer-creator YouTube channels; new videos surface in the feed as rich cards.

**Critical**: we don't republish third-party copyrighted content verbatim — we link out and show a thumbnail + caption. Our own content and user-contributed content sits under the corpus's CC-BY-SA licensing.

## 5.13 Public read-only API

API-key gated, rate-limited, for DOA, researchers, third-party apps. Same core content endpoints as the client app, JSON + CSV exports. Attribution required per CC-BY-SA.

## 5.14 Feature-priority matrix

| Feature | Client app | Admin portal | Mobile | Day-1 (pilot) | Day-30 | Day-90 |
|---|---|---|---|:---:|:---:|:---:|
| Onboarding + OTP | ✓ | — | ✓ | ✓ | | |
| Crop explorer (read) | ✓ | ✓ | ✓ | ✓ | | |
| Map home (base + AEZ overlays) | ✓ | — | ✓ | ✓ | | |
| Plant-of-the-month pins | ✓ | ✓ | ✓ | | ✓ | |
| Content calendar | ✓ | ✓ | ✓ | | ✓ | |
| Recommender | ✓ | — | ✓ | | ✓ | |
| Weather + advisories | ✓ | ✓ | ✓ | | ✓ | |
| Disease scanner | — | ✓ review queue | ✓ | | | ✓ |
| Community feed | ✓ | ✓ moderation | ✓ | | ✓ | |
| Nearby-growers map pins | ✓ | — | ✓ | | ✓ | |
| Social-media feed ingest | ✓ | ✓ | ✓ | | | ✓ |
| Marketplace (listings) | ✓ | ✓ moderation | ✓ | | | ✓ |
| Marketplace (payments) | ✓ | ✓ | ✓ | | | ✓+ |
| Record editors | — | ✓ | — | ✓ | | |
| Translations console | — | ✓ | — | ✓ | | |
| Public read-only API | ✓ | — | — | | ✓ | |
