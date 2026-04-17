# 01 — Vision & scope

## The problem

Sri Lankan agriculture sits on rich domain knowledge scattered across the Department of Agriculture (DOA), research institutes (HORDI, RRDI, FCRDI, SRI, CRI), university extension notes, cooperative pamphlets, YouTube channels, and WhatsApp groups. Professional farmers rely on decades of personal experience; hobbyists and new backyard growers have no coherent entry point. Disease identification depends on word-of-mouth or a trip to the nearest Agrarian Services Centre. Market access is fragmented — produce moves through middlemen with no price transparency.

## What we're building

A single platform (mobile-first, with a companion web app and admin CMS) that:

1. **Answers "what can I grow here?"** for any coordinate in Sri Lanka, given the user's plot size, soil, water access, budget, and effort level.
2. **Answers "what's wrong with my plant?"** from a photo, with remedies linked back to the knowledge base.
3. **Answers "how do I grow it?"** with step-by-step cultivation guides, seasonal calendars, input lists, and curated video resources.
4. **Connects farmers to each other and to buyers** through a community feed and a produce marketplace.

## Users

| Persona | Description | Primary needs |
|---|---|---|
| **Seasoned farmer** | 10+ years, 1+ acres, commercial focus. Often in Dry/Intermediate zones. | Market prices, new varieties, pest/disease alerts, bulk buyer contacts |
| **Smallholder** | <1 acre, subsistence + partial sale. Mixed crops. | Seasonal planning, input optimization, extension advice, marketplace |
| **Hobbyist / backyard grower** | Urban/suburban, 1–10 perches, weekends. | Beginner guides, what-to-plant-now, disease ID, community |
| **Agronomist / extension officer** | DOA field staff, NGO workers. | Contribute content, review flagged scans, broadcast advisories |
| **Buyer** | Households, restaurants, small retailers. | Discover local produce, place orders |

## Scope: in

- Field crops (rice, maize, soybean, green gram, cowpea, groundnut, finger millet)
- Vegetables — lowland (brinjal, okra, long bean, pumpkin, bitter gourd, snake gourd, luffa, ash plantain, manioc, sweet potato) and upland (carrot, cabbage, leeks, beetroot, tomato, capsicum)
- Fruits (banana, papaya, mango, pineapple, rambutan, mangosteen, durian, avocado, passion fruit, citrus)
- Spices (cinnamon, pepper, cardamom, cloves, nutmeg, ginger, turmeric)
- Plantation crops at reference level (tea, rubber, coconut) — not primary focus but linked
- Home-garden and organic variants
- Pest and disease catalog with images, symptoms, remedies (chemical, biological, cultural)
- Geospatial recommendation: AEZ, soil, elevation, rainfall, temperature match
- Seasonal calendars: Maha (Oct–Mar), Yala (Apr–Sep)
- Market price tracking (wholesale/retail from Dambulla, Meegoda, Pettah economic centres where published)

## Scope: out (for v1)

- Livestock, poultry, fisheries, apiculture (add later as separate modules)
- Payment escrow and logistics for marketplace (start with listing + contact; add payments in phase 5)
- Satellite-based crop monitoring (later phase)
- IoT sensor integration (later phase)

## Open-source knowledge corpus

The curated knowledge base — crops, diseases, remedies, cultivation steps, AEZ/soil/rainfall layers we vectorize — is released as a **public, versioned corpus** under CC-BY-SA 4.0 (content) and ODbL (geodata). The app consumes the corpus; it does not own it. The corpus is the durable public good; the app is how we make it useful to farmers. Anyone — other developers, researchers, NGOs, the DOA itself — can fork, mirror, or build derivative tools on top of it.

This is a deliberate choice, not a side-effect. It shapes what we gather (only what we can lawfully release), how we gather (crawl + own-extraction, no proprietary feeds), and how we govern quality (public issue tracker, community corrections via PR).

## Languages

Trilingual content from day one: **Sinhala, Tamil, English**. Every crop, disease, symptom, remedy, and UI string is a translatable key. Audio narration for low-literacy users is a phase-2 add-on.

## Success metrics

- **Coverage** — ≥ 150 crops, ≥ 300 diseases/pests with ≥ 3 images each, ≥ 80% of AEZs with ≥ 10 crop recommendations.
- **Recommendation accuracy** — agronomist review confirms top-5 recommendations are agronomically valid for ≥ 90% of sampled coordinates.
- **Scanner accuracy** — ≥ 85% top-3 accuracy on a held-out Sri Lanka-specific test set.
- **Adoption** — DAU/MAU ≥ 0.3 six months post-launch in pilot districts.
- **Marketplace liquidity** — median listing sells or expires with ≥ 1 contact within 14 days.
