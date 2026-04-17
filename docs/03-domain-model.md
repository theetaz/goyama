# 03 — Domain model & knowledge graph

## Modeling approach

We use a **hybrid** store:

- **PostgreSQL + PostGIS** as the system of record — relational integrity, geospatial queries, proven operational tooling.
- **Knowledge graph projection** — built on top of Postgres using an edge table, OR materialized into a graph engine (Neo4j / Dgraph / Apache AGE) for traversal-heavy queries (e.g. "find crops suitable for this AEZ that are resistant to a disease currently spreading in the neighboring district").

Start with Postgres + AGE (Apache AGE gives you Cypher queries inside Postgres — one database to operate). Move to a dedicated graph DB only if traversal latency becomes a bottleneck.

## Core entities

All entities are multilingual: every display string is a key into a `translations` table keyed by `(entity_type, entity_id, field, locale)`.

### Crop
```
crop(
  id, slug, scientific_name, family,
  life_cycle, growth_habit,
  default_season,             # maha | yala | perennial | year_round
  duration_days_min, duration_days_max,
  elevation_m_min, elevation_m_max,
  rainfall_mm_min, rainfall_mm_max,
  temp_c_min, temp_c_max,
  soil_ph_min, soil_ph_max,
  water_requirement,          # low | medium | high
  category,                   # field | vegetable | fruit | spice | plantation
  effort_level,               # low | medium | high  (hobbyist matching)
  status, created_at, updated_at, owner_id
)
```

### CropVariety
```
crop_variety(
  id, crop_id, name, released_by, release_year,
  duration_days, yield_per_acre_kg,
  traits jsonb,               # {disease_resistance: [...], drought_tolerance: 'high', ...}
  recommended_aezs int[]      # foreign refs to AEZs
)
```

### AgroEcologicalZone
```
aez(
  id, code,                   # e.g. WL1a, IL1, DL1b
  zone_group,                 # wet | intermediate | dry
  geom geography(MultiPolygon),
  elevation_class,            # low | mid | up country
  avg_rainfall_mm, avg_temp_c,
  dominant_soil_groups text[]
)
```

### SoilType, RainfallCell, AdminArea
Standard geo tables, each with `geom` and attributes. `admin_area` is hierarchical (province → district → DS division → GN division).

### Disease, Pest
```
disease(
  id, slug, scientific_name, common_name,
  causal_organism,            # fungal | bacterial | viral | nematode | deficiency | physiological
  transmission,
  favored_conditions jsonb,   # {humidity: 'high', temp_c: [24, 30], ...}
  severity,                   # low | medium | high
  status, owner_id
)
```

### Symptom (owned by disease or pest)
```
symptom(
  id, disease_id nullable, pest_id nullable,
  stage,                      # early | mid | severe
  affected_part,              # leaf | stem | root | fruit | flower
  description_i18n,
  confusion_with int[]        # other symptom ids commonly mistaken
)
```

### SymptomImage
```
symptom_image(
  id, symptom_id, crop_id,
  url, credit, license,
  captured_at, geom,          # if known
  is_training_data boolean,
  moderation_status
)
```

### Remedy
```
remedy(
  id, disease_id, pest_id nullable, crop_id nullable,
  type,                       # cultural | biological | chemical | resistant_variety
  active_ingredient,          # for chemical
  doa_registration_no,
  dosage, application_method, frequency,
  pre_harvest_interval_days,
  safety_notes_i18n, status, owner_id
)
```

### CultivationStep
```
cultivation_step(
  id, crop_id, variety_id nullable, aez_id nullable, season,
  stage,                      # land_prep | nursery | planting | fertilization | irrigation | pest_mgmt | harvest | post_harvest
  day_after_planting_min, day_after_planting_max,
  title_i18n, body_i18n,
  inputs jsonb,               # [{type:'fertilizer', name:'Urea', amount_kg_per_acre: 50}, ...]
  media_ref int[]
)
```

### Media
```
media(
  id, type,                   # image | video | pdf | audio
  url, source, credit, license, language,
  tags text[], owner_id
)
```

### User, Farm, Plot
```
user(id, phone, email, display_name, locale, role, ...)
farm(id, owner_user_id, name, district_id)
plot(id, farm_id, geom, area_m2, soil_type_id, elevation_m, water_source, ...)
```

### Crop suitability edge
A computed, reviewed edge:
```
crop_suitability(
  crop_id, aez_id,
  suitability,                # S1 | S2 | S3 | N   (FAO land suitability classes)
  notes_i18n,
  reviewed_by, reviewed_at,
  model_version               # when computed by the recommender
)
```

### Scan (user disease submission)
```
scan(
  id, user_id, plot_id nullable, geom,
  crop_id nullable,           # user-declared or inferred
  image_urls text[],
  model_predictions jsonb,    # [{disease_id, confidence}, ...]
  user_confirmed_disease_id,
  expert_reviewed_disease_id,
  status,                     # pending | model_only | user_confirmed | expert_confirmed
  created_at, reviewed_at
)
```

### Listing (marketplace), Post, Comment, Message
Covered in doc 07.

## Knowledge graph view

Nodes: `Crop`, `Variety`, `AEZ`, `Soil`, `Season`, `Disease`, `Pest`, `Remedy`, `Nutrient`, `Input`, `Market`, `User`, `Plot`.

Edges (examples):
- `(:Crop)-[:SUITABLE_IN {class:S1}]->(:AEZ)`
- `(:Crop)-[:AFFECTED_BY]->(:Disease)`
- `(:Disease)-[:TREATED_BY]->(:Remedy)`
- `(:Crop)-[:GROWS_IN]->(:Season)`
- `(:Crop)-[:PREFERS_SOIL]->(:Soil)`
- `(:Crop)-[:ROTATES_WITH]->(:Crop)`
- `(:Crop)-[:COMPANION_OF]->(:Crop)`
- `(:Variety)-[:RESISTANT_TO]->(:Disease)`
- `(:Plot)-[:LOCATED_IN]->(:AEZ)`
- `(:User)-[:OWNS]->(:Plot)`
- `(:Scan)-[:PREDICTED]->(:Disease {confidence})`

Example queries the graph unlocks:

- "For my plot's AEZ, find crops suitable in the upcoming season, rank by market price, exclude any crop with an active disease alert within 50km."
- "Given this disease, list resistant varieties that also suit my AEZ and are available from nearby seed suppliers."
- "What's a 3-crop rotation plan starting from my current brinjal plot that maximizes soil health and avoids shared pests?"

## Versioning & provenance

Every row in canonical tables has `version`, `replaced_by`, `source_refs jsonb[]`, `confidence`, `reviewed_by`, `reviewed_at`. API clients can request either the latest published version or a historical snapshot (`?as_of=2026-03-01`).

## Indexes (non-exhaustive)

- GIST on all `geom` columns.
- Trigram + embedding indexes on names (for search + entity resolution).
- Composite `(crop_id, aez_id, season)` on cultivation_step.
- Partial index `WHERE status='published'` on all content tables.

## Schema change workflow

1. Propose a change in a migration PR with rationale + data migration.
2. Run on staging against a full production snapshot.
3. Agronomist lead approves content-impacting changes.
4. Ship with backwards-compatible API read; deprecate old fields over two releases.
