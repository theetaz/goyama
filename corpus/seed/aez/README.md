# Agro-ecological zones — seed notes

Sri Lanka's Natural Resources Management Centre (NRMC) classifies the country into three macro climatic zones, subdivided into agro-ecological regions. Agronomists refer to sub-region codes in the form `WL1a`, `IL2`, `DL3b`, etc. — where the first letter encodes zone group (W/I/D = Wet/Intermediate/Dry) and the second encodes elevation class (L/M/U = Low/Mid/Up-country).

**Structured polygon records are deferred to a later pass** — vectorizing the NRMC shapefiles is a dedicated workstream. This document holds the narrative summaries, each citable, so the app can show zone context for a location even before polygons are available.

---

## Wet Zone

- **Coverage**: south-western Sri Lanka including the central hill country.
- **Annual rainfall**: over 2,500 mm.
- **Dry season**: none pronounced.
- **Typical crops**: tea (up-country), rubber (mid- and low-country), coconut (low-country), cinnamon, rambutan, mangosteen, pineapple, upland vegetables (carrot, cabbage, leeks, beetroot) at higher elevations.

> "The Wet Zone covers the south-western region including the central hill country and receives relatively high mean annual rainfall over 2,500 mm without pronounced dry periods."
> — [Sri Lanka Biodiversity (CBD CHM)](https://lk.chm-cbd.net/biodiversity), fetched 2026-04-17.

## Intermediate Zone

- **Coverage**: buffer between Wet and Dry zones; 20 sub-regions (15 in the central hills).
- **Annual rainfall**: 1,750 – 2,500 mm.
- **Dry season**: short and less pronounced.
- **Typical crops**: a mix of Wet-Zone and Dry-Zone crops; rice in both Maha and Yala; mixed home gardens; fruit trees.

> "The Intermediate zone receives a mean annual rainfall between 1,750 to 2,500 mm with a short and less prominent dry season."
> — [Sri Lanka Biodiversity](https://lk.chm-cbd.net/biodiversity), fetched 2026-04-17.

## Dry Zone

- **Coverage**: northern and eastern two-thirds of the country; 11 sub-regions.
- **Annual rainfall**: under 1,750 mm.
- **Dry season**: pronounced, May – September.
- **Typical crops**: rice (Maha-dominant with tank irrigation; Yala where supplemental water available), big onion, chilli, cowpea, green gram, maize, sesame, groundnut; dry-zone fruits (mango, papaya).

> "The Dry zone receives a mean annual rainfall of less than 1,750 mm with a distinct dry season from May to September."
> — [Sri Lanka Biodiversity](https://lk.chm-cbd.net/biodiversity), fetched 2026-04-17.

---

## Sub-regional count — to reconcile

Different sources report different sub-region counts:

- Some sources cite **24 agro-ecological regions** (the classical FAO reference and many university texts).
- Other sources cite **46 AEZs** (a later NRMC subdivision).

Both numbers are found in the literature; the 46-zone classification appears to be the more recent and finer-grained one used operationally by the NRMC. The platform will publish whichever set the NRMC polygon dataset encodes, with clear citation.

Action: fetch the NRMC AEZ polygon dataset (FAO GAEZ mirror or original NRMC shapefile) and vectorize for the canonical `aez` table. Reconciliation notes stay in this file.

**Sources:**
- [Agro Ecological Regions of Sri Lanka — FAO GAEZ v4](https://gaez.fao.org/datasets/agro-ecological-regions-of-sri-lanka)
- [Sri Lanka Biodiversity — CBD Clearing House](https://lk.chm-cbd.net/biodiversity)
- [FAO: Agro-biodiversity — Sri Lanka agro-ecological zones, cropping systems](https://www.fao.org/4/ac791e/AC791E05.htm)
