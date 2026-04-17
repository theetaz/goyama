# 08 — Roadmap & milestones

A phased plan that prioritizes **data quality** and **one delightful core loop** before breadth.

## Guiding sequencing principle

The order is: **data → knowledge base → recommendations → scanner → marketplace → community**. Each phase produces something usable on its own, and each later phase depends on the data and trust built in the earlier ones. Do not build the marketplace before there is a knowledge base people come back for.

## Phase 0 — Foundations (weeks 1–4)

Goal: a team, a schema, and a working ingestion loop.

- Hire or commit: 1 backend lead, 1 mobile lead, 1 data/ML engineer, 1 lead agronomist (part-time acceptable), 1 product/ops.
- Finalize canonical schemas (doc 03). Stand up Postgres (with PostGIS + AGE + pgvector), object storage, search index.
- Build the crawling + extraction skeleton (fetch → raw → extract → LLM-assisted parse → staging → canonical) with one end-to-end source (DOA crop list). Polite crawler framework: `robots.txt`, rate-limit, UA identity, caching, snapshot archive.
- Build the admin CMS shell — login, crud for crops/diseases/remedies, version history.
- Ingest geo base layers (admin, AEZ, soil, elevation, rainfall). Reverse-geocode + AEZ lookup API working.

**Exit criteria:** a lat/lng returns `{district, ds_division, aez, soil, elevation, rainfall_normal}` via the API.

## Phase 1 — Knowledge base MVP (weeks 5–12)

Goal: a browsable, searchable, trustworthy catalog for 30 priority crops and their top diseases.

- Complete data for 30 priority crops (see doc 02 for the list rationale).
- 60+ diseases, each with ≥ 3 reviewed images and reviewed remedies.
- Cultivation steps for each crop × 2 seasons.
- Curated videos tagged.
- Web app (Next.js) published: crop explorer, disease catalog, map view, search.
- Mobile app alpha (TestFlight / internal track): same read-only surfaces + offline sync for the 30 crops.
- Market price daily job (at least Dambulla + one other DEC).
- **Corpus v0.1** published to the public GitHub repo (`cropdoc/corpus`) under CC-BY-SA / ODbL with full provenance.

**Exit criteria:** an agronomist reviews a random sample of 50 content pages and finds ≥ 90% publish-ready accuracy. Public web launch behind a "beta" banner. First open-corpus release tagged.

## Phase 2 — Personalized recommendations (weeks 13–18)

Goal: "what should I plant on my plot this season?" works well.

- Plot creation flow (draw polygon or drop pin) in mobile + web.
- Recommender service v1 (rule-based + weighted scoring, explainable output).
- Weather integration per plot (7-day forecast, agro-alerts).
- Planting plan + reminders (push + SMS).
- Agronomist review sample-set for recommendations: 90% agronomic validity on sampled coordinates.

**Exit criteria:** target users in pilot districts (start with Kandy + Kurunegala + Anuradhapura — covering Intermediate, Dry, Wet accent) use the app for their planting decisions and are tracked via opt-in surveys.

## Phase 3 — Disease scanner (weeks 15–28, parallel to Phase 2)

Goal: a pocket agronomist that works offline.

- Data collection ops: 30 crops × 6 diseases × 3 stages × 100 images minimum.
- Phase-1 model (classification, on-device + server). Top-3 ≥ 80% on held-out set.
- Scanner flow in mobile (capture → predict → remedy → optional expert review).
- Expert review queue in CMS.
- Advisory heatmap (fuzzed) on map.

**Exit criteria:** 1,000 user scans with expert-review turnaround median ≤ 24 h. Model top-3 ≥ 85%.

## Phase 4 — Marketplace MVP (weeks 25–36)

Goal: listings + chat. No payments yet.

- Seller onboarding with optional verification (NIC + farm geo).
- Listings CRUD, discovery (list + map), filters, saved searches with alerts.
- In-app chat (WebSocket).
- Moderation tooling.
- Reputation v1.

**Exit criteria:** in pilot districts, median listing gets ≥ 1 genuine contact within 14 days; report-to-takedown time ≤ 4 h.

## Phase 5 — Community + payments (weeks 33–48)

Goal: farmers talk to each other, and buyers can pay in-app.

- Feed, posts, comments, reactions, follows, questions with verified-agronomist answers.
- PayHere payment integration with escrow.
- Reviews + reputation v2.
- Groups (district + crop).

**Exit criteria:** DAU/MAU ≥ 0.3 in pilot districts. Marketplace GMV tracked; no more than 0.5% of transactions open a dispute.

## Phase 6 — Scale + advanced (weeks 45+)

- Nationwide rollout, additional districts onboarded sequentially.
- Scanner v2 (detection, open-set, multimodal).
- Satellite-derived NDVI per plot (Sentinel-2) for crop-health heatmaps.
- IoT sensor integrations (optional partners).
- Livestock / poultry / apiculture modules (separate product squads).
- Monetization: featured listings, B2B buyer tier for restaurants/retailers.
- Partnerships with banks for crop-insurance and input-financing within the app.

## Cross-cutting workstreams (always-on)

- **Content coverage** — weekly ops target: +3 crops or +10 diseases fully reviewed.
- **Data pipeline health** — ingestion SLO: 99% of scheduled source fetches succeed; alerts on regressions.
- **Model refresh** — monthly retrain once scan volume ≥ 500/week.
- **i18n completeness** — Sinhala/Tamil parity is a release gate for any user-facing string.
- **User research** — monthly visits to farms in pilot districts; rotate engineers so the team stays grounded.
- **Security & privacy review** — quarterly external audit; annual penetration test.

## Team shape (target by end of year 1)

- 2 backend, 2 mobile, 1 web, 2 data/ML, 1 DevOps, 2 agronomists, 1 community/ops, 1 PM, 1 designer, 1 content/translation lead. ~13 people.

## Risks & mitigations

| Risk | Mitigation |
|---|---|
| Source sites block or rate-limit crawlers | Polite crawl policy, identifiable UA, on-host caching; diversify sources so no single site is load-bearing; archive snapshots for link rot |
| Upstream content licences unclear | Per-source licence register maintained before first crawl; when in doubt, link out instead of redistributing |
| Sri Lanka-specific labeled images are scarce | Fund curated field shoots from Phase 0; incentivize agronomist uploads; semi-supervised learning on user scans |
| Users give up on photo capture | Strong in-app capture hints; quality gate before submit; on-device fast path |
| Chemical recommendations cause harm | Agronomist-reviewed remedies only; never auto-suggest dosages beyond the DB; PHI prominent in UI |
| Marketplace abuse (fraud, off-platform diversion) | Verification tiers, reputation, in-app chat only for contact by default |
| Hosting cost at scale | Cloudflare-heavy stack, aggressive image processing, scale services independently |
| Single-founder / tech-lead key-person risk | Pair-on-call from phase 1; runbook discipline from day one |
