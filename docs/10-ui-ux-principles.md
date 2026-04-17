# 10 — UI / UX principles for the farmer-facing app

The client app is the most-used surface and the one that determines whether a farmer comes back tomorrow. This document captures the design principles, reference-app studies, and visual-language decisions that drive the design system used by `apps/web-client`, `apps/mobile`, and (in a denser adaptation) `apps/web-admin`.

## Who we're designing for

Primary personas, in order of product priority:

1. **Smallholder in the dry zone**, 35–60 years old, 1–5 acres, mixed crops. Uses a mid-tier Android (Redmi Note-class) phone with a cracked screen, in bright sunlight, with intermittent 3G. Reads Sinhala primarily; may read some English. Has never used a "modern" app UI with drawers or tabs.
2. **Hobbyist / backyard grower**, 25–45, urban/suburban, 1–10 perches. Has a flagship Android or iPhone. Reads English fluently, Sinhala at home. Looks up crops weekly.
3. **Commercial farmer / estate manager**, 30–55, 5+ acres or manages a plantation. Uses a laptop more than a phone. Reads English + Sinhala. Wants bulk operations, market prices, input costing.
4. **Agronomist / extension officer** — uses `web-admin`. Reads English primarily; writes in all three languages.

## Core principles

### 1. Trilingual from day one, not as an afterthought
Every screen must be readable and fully functional in **Sinhala**, **Tamil**, and **English**. Language switch is a first-class affordance, accessible from onboarding and from every header. **No screen ships until translated copy is in place for all three.** Translations in the app bundle are versioned separately from the app shell so copy can be fixed without a release.

### 2. Sunlight-readable by default
The dry-zone farmer uses the phone outside, at midday, with glare. Dark text on light backgrounds, minimum **WCAG AAA** contrast (7:1) for body copy. Default theme is a high-contrast light mode, with an explicit "sunlight" mode that pushes contrast further (black text on near-white, saturated accents only in buttons and icons). Night mode exists but is secondary.

### 3. Thumb-reach and big targets
Primary actions sit in the bottom third of the screen on mobile. Minimum tap target **48 × 48 dp**. Text labels on every icon — icon-only buttons are banned in the farmer app.

### 4. Image-first content
Farmers recognise crops and symptoms by sight long before they parse text. Every crop, every disease, every symptom has a hero image. Search results show thumbnails. Lists are card-based with image left, title + 1–2 lines of copy right.

### 5. Plain-language copy, not agronomist copy
Our source material is full of jargon ("50% flowering at 34 DAT", "S1 suitability class", "PHI 14 days"). The app translates jargon into plain language *and* offers a "show me the source" expander for users who want the detail. Example:

- **Expert**: *"PHI: 14 days"*
- **Farmer**: *"Wait 14 days after your last spray before harvesting."* (+ tap-to-expand source)

### 6. Offline-first and delta-sync
The app is usable with no connection after first run. Every content fetch writes to local SQLite; UI reads from SQLite and reconciles in the background. "Downloaded" status is visible per crop so users know what they have offline.

### 7. Positive, encouraging tone — no gatekeeping
Error messages are kind. Missing data is framed as an opportunity ("Help us complete this page — submit a photo") not as a failure. Onboarding celebrates the user's commitment ("Welcome, grower!").

### 8. Social, not solitary
A farmer opening the app sees **what other farmers are doing near them** — "3 growers in Kurunegala planted brinjal this week" — as a default home-screen ambient signal. Community posts, questions, and nearby activity are woven into the reader experience, not siloed in a "Community" tab.

### 9. Map as a first-class home surface
The interactive Sri Lanka map is not just a feature — it's the default home screen for signed-in users with a registered farm. Plant-of-the-month pins, nearby grower pins, and AEZ overlays sit on top of an aesthetic MapLibre base style tuned for Sri Lanka's coastlines and tank cascades.

### 10. Fun, not clinical
Taking inspiration from games and gardening apps rather than extension pamphlets. Subtle animations on card reveal, playful empty states, a friendly mascot for tips, and reward loops for completing a planting plan. Never gamified to the point of being childish — agricultural credibility is paramount.

## Visual language

### Color system (OKLCH)

Core palette — earthy + bright, drawn from crop imagery:

| Token | OKLCH | Role |
|---|---|---|
| `--primary` | `oklch(0.55 0.16 142)` (paddy green) | Primary actions |
| `--primary-fg` | `oklch(0.99 0.01 142)` | Text on primary |
| `--accent` | `oklch(0.75 0.15 80)` (turmeric amber) | Emphasis, highlights |
| `--danger` | `oklch(0.60 0.22 27)` (ripe chilli) | Destructive + disease alerts |
| `--surface` | `oklch(0.98 0.01 90)` (washed rice) | Cards + inputs |
| `--soil` | `oklch(0.35 0.06 45)` (reddish-brown earth) | Secondary surfaces |
| `--ink` | `oklch(0.22 0.02 260)` (near-black) | Body text |

Each mode (light / sunlight / dark) maps the same semantic tokens to different OKLCH lightness values — components never reference raw colors.

### Typography

- **Latin**: Inter variable (400/500/700). System fallbacks for offline.
- **Sinhala**: Noto Sans Sinhala (400/500/700).
- **Tamil**: Noto Sans Tamil (400/500/700).
- **Numerics**: tabular lining figures via `font-variant-numeric: tabular-nums`.
- **Scale**: 14 / 16 / 18 / 22 / 28 / 36 (1.25 modular). 16 is the minimum body size.
- Line-height 1.5–1.6 for Sinhala and Tamil (which need more vertical room than Latin).

### Iconography

- **Lucide** as the base set, with a small custom agri glyph extension: paddy stalk, coconut palm, cinnamon quill, brinjal, banana bunch, compost heap, tractor, irrigation channel. Glyphs designed at 24 px with 2 px stroke to match Lucide.

### Motion

- Default easing: `cubic-bezier(0.22, 1, 0.36, 1)` (smooth ease-out).
- Durations: 150 ms (micro), 250 ms (element), 400 ms (route transition).
- **Respect `prefers-reduced-motion`** everywhere.

## Reference apps — what we're learning from

| App | What we take | What we reject |
|---|---|---|
| **Plantix** | On-device disease scan, tight crop catalog, remedy tabs (cultural/biological/chemical). | Cluttered home; ads; over-prescription of chemicals. |
| **PictureThis / Seek** | Onboarding clarity, camera-first landing, friendly empty states. | Subscription-gating of core features. |
| **iNaturalist** | Community observation feed, map layer with pins, expert verification flow. | Scientific register is too dense for non-biologist users. |
| **Kisan Suvidha (India)** | Hyper-local advisories, SMS fallback, voice-notes. | Bureaucratic UI, no social layer. |
| **Govi Mithuru (SL, Dialog)** | SMS-first accessibility, locale-aware cultivation advice, proven fit for Sri Lankan farmer. | No visual layer — everything is text SMS. |
| **Govi AI (SL)** | Trilingual disease ID, SL-tuned experience. | Narrow scope; we want more than scans. |
| **Digital Green** | Video-first extension, peer-to-peer learning loops. | Depends on staff-produced video — ours should be community-contributed too. |
| **FarmLogs / Climate FieldView** | Plot-centric data model, season planner, actionable alerts. | US-centric, broadacre focus — doesn't translate to Sri Lankan smallholders. |
| **AllTrails / Komoot** | Map as the primary canvas; discovery by browsing the map. | Not an agri precedent but the *pattern* matches — a SL farmer browsing "what grows near me" on a map is the same shape. |
| **Duolingo** | Micro-progress, celebratory reward loops, streak motivation. | Not gamified beyond light touches — farming is serious. |

## The map as the home screen

Once a user has set their location, the default route is `/map` with:

- **Base**: MapLibre vector tiles styled with a "Ceylon" palette — muted greens for Wet Zone, warm ochres for Dry Zone, subtle blue for tanks and reservoirs.
- **AEZ overlay**: toggleable, labeled, with a legend chip per zone.
- **Plant-of-the-month pins**: crops that are optimal to plant *this month* at the user's location.
- **Nearby grower pins**: other registered farmers who have opted in to show location (fuzzed to GN-division). Tap to view their public profile (crops, posts, listings).
- **Disease-pressure heatmap**: optional layer, coloured by recent confirmed scans in the area — helps farmers anticipate and watch for outbreaks.
- **Weather badge**: current and next-24-h forecast overlay at the pinch location, pulled from Met Dept / Open-Meteo.
- **Search bar**: "What to plant at my plot?" or "Where can I grow rambutan?" — re-centres map on results.
- **Bottom sheet**: persistent, pulls up to reveal plant-of-the-month cards, nearby activity feed, and quick actions (Scan, Add plot, Post).

## Accessibility and inclusivity

- Minimum **WCAG AA** across the app; **AAA** for body copy.
- **Screen-reader labels** in all three languages on every interactive element.
- **Keyboard navigation** on web-client; **TalkBack** and **VoiceOver** support on mobile.
- **Voice input** for search and for posting (Sinhala + Tamil ASR where available).
- **Audio narration** of key cultivation steps for low-literacy users (phase 2).
- **Large-text preference** honoured via Dynamic Type / Android text scaling.
- **Font-loading strategy**: system fonts render instantly with Sinhala/Tamil fallback; Noto swaps in when loaded — no blank text.

## Tone of voice

- Second person singular, warm but not patronising. ("Your rice plants might be hungry — their leaves are pale.")
- No "sir/madam". No "please" before every command — it's cluttered. Direct but kind.
- Celebrate small wins. When a user completes a cultivation step, the app says "Nicely done — your next step is X in 5 days."
- Localised idioms where natural. Sinhala and Tamil copy written by Sri Lankan copywriters, not translated from English.

## UI research before first pixel

Before a line of component code is written, we run:

1. **Competitive audit** — install every app in the reference table and capture screens, note annoyances.
2. **Farmer interviews** (6–10 across personas) — show paper prototypes, watch them swipe, listen to what they call things.
3. **Landscape study** — how Sri Lankan farmers already solve each task (WhatsApp groups, Facebook groups, extension officer visits, SMS services) — so the app *complements* rather than duplicates.
4. **Sinhala / Tamil copy review** — a native speaker of each reads *every* label on the wireframes and flags anything that sounds like a machine translation.
5. **Colour + icon test** — show the palette and custom glyphs to 10 target users; iterate on any that are ambiguous or carry unintended meanings.

None of this is skippable. The cost of shipping a confusing UI to a farmer who's already wary of apps is **they never come back** — and because the product is viral within villages, one bad experience kills village-level adoption.

## Design deliverables

- **Figma file** (`design/cropdoc.fig`) with:
  - Design tokens (colour, typography, spacing, motion)
  - Component library mirroring Shadcn components
  - Screens: onboarding, map home, crop detail, disease scan, plant planner, community feed, marketplace, profile
- **Exportable tokens** synced into `packages/design-tokens` via Style Dictionary or Tokens Studio.
- **User-flow maps** for every primary task (plant a crop, diagnose a disease, list produce for sale, post to community).

## Performance targets

- **First interactive** on a mid-tier Android over 3G:  ≤ **3 s** for the web-client SPA (code-split + preconnect to API + CDN-served assets).
- **Mobile cold start**: ≤ **2 s** on mid-tier Android.
- **Disease-scan on-device**: ≤ **500 ms** inference on mid-tier Android.
- **Map first paint** (tiles + overlays): ≤ **1.5 s** on 3G after service-worker-cached base style.

These numbers drive real engineering decisions — image sizes, bundle budgets, tile pre-cache policy.
