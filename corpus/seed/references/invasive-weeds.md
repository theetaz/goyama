# Invasive aquatic and agricultural weeds — reference notes

Noted here as a reference document pending a formal `weed` entity type in the schema.

## Water hyacinth — *Pontederia crassipes* (syn. *Eichhornia crassipes*)

- **Introduction**: Brought to Sri Lanka as an ornamental plant in **1904** from the Amazon Basin. Declared a prohibited weed under the **Water Hyacinth Act (1909)** — one of Sri Lanka's oldest weed-control statutes — and later under the Plant Protection Act (1924).
- **Local name**: Historically called "Japanese trouble" during WWII on rumours that the British spread it to block Japanese seaplane landings.
- **Current impact**: Widely distributed across tanks, canals, and lakes. The **Irrigation Department spends >55% of its annual budget on removing invasive aquatic species from irrigation systems** — a staggering ecological-economic cost.
- **Management**:
  - **Manual removal** — primary method for small infestations; labour-intensive.
  - **Biological control** — the weevil *Neochetina eichhorniae* introduced in 1988; breeding well, with efficacy becoming evident post-1994. Isolated fungal species also under study as biocontrol agents.
  - **Composting** — mixed with organic municipal waste, ash, and soil; composted and sold to farmers/gardeners. Valuable nutrient recycling.
  - **Chemical** — glyphosate and 2,4-D approved in controlled use (avoid fish-stocked tanks).
- **Co-invader**: *Salvinia molesta* — floating fern dominating eutrophic waterways along with water hyacinth.

## Other significant agricultural weeds (future catalogue)

- **Parthenium hysterophorus** — recent invasive in Dry Zone; allergenic.
- **Mikania micrantha** — "mile-a-minute weed"; smothers young tea, rubber, plantation crops.
- **Mimosa pigra** / **Mimosa invisa** — thorny leguminous weeds in coconut, rubber lands.
- **Lantana camara** — ornamental turned invasive; up-country pastures and road verges.
- **Chromolaena odorata** — "Siam weed" in fallow fields.
- **Nadu grass** (*Paspalum conjugatum*) — aggressive turf weed in rubber and cocoa.
- **Purple nutsedge** (*Cyperus rotundus*) — most widespread perennial weed in SL cropland.

## Sources
- [Wikipedia — Pontederia crassipes](https://en.wikipedia.org/wiki/Pontederia_crassipes)
- [ResearchGate — Water hyacinth invasion of SL irrigation tanks](https://www.researchgate.net/publication/343426601_Invasion_of_water_hyacinth_Eichhornia_crassipes_in_selected_irrigation_tanks_in_Sri_Lanka)
- [ScienceDirect — Salvinia + Eichhornia biocontrol in Sri Lanka](https://www.sciencedirect.com/science/article/abs/pii/030437709290001Y)
- [SLJAE — composting water hyacinth with leaf litter](https://sljae.sljol.info/articles/10.4038/sljae.v3i1.57)
- [SCAR Sri Lanka — Urban wetland invasive species removal](https://scar.lk/urban-wetland-invasive-species-removal-program/)

## Next step

Add a canonical `weed` entity type to the schema with fields: `scientific_name`, `affected_systems` (e.g. irrigation, paddy, plantation), `biocontrol_agents`, `approved_herbicides`, `regulatory_status` (Plant Protection Act listing).
