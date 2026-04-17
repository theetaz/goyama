# Sri Lanka Great Soil Groups — narrative notes

The corpus currently uses soil-group names as string tokens in crop records (`preferred_soil_groups`, AEZ `dominant_soil_groups`). This note documents the groups referenced across the corpus so the app can link crop records to soil-map lookups once NRMC / Land Use Policy Planning Department polygons are vectorised.

## Primary Great Soil Groups of Sri Lanka

### Reddish Brown Earth (RBE) soils
- **Where**: Upper and mid slopes of the landscape in the Dry Zone; occupies the largest area of the Dry and drier Intermediate zones.
- **Texture**: Reddish to reddish-brown; moderately deep, well-drained.
- **Crops**: Sugarcane (Uda Walawe / Walawa series), rice under tank irrigation, chilli, big onion, cowpea, sesame, groundnut, maize, cotton.
- **Notes**: Upland sugarcane productivity in the RBE depends heavily on physical-property management (aggregate stability, compaction).

### Low Humic Gley (LHG) soils
- **Where**: Valley bottoms of undulating topography across the Dry and Intermediate zones — the **classic paddy soil of the Dry Zone**.
- **Texture**: Greyish, deep, moderately fine; hydromorphic with seasonal waterlogging — which suits rice.
- **Crops**: Rice (Maha-dominant), with rice-legume rotation (mung bean, cowpea) in the Yala dry spell.

### Non-Calcic Brown (NCB) soils
- **Where**: Upper and mid slopes and well- to imperfectly-drained areas. Primarily Dry and drier Intermediate zones.
- **Crops**: Field crops similar to RBE; well suited to upland vegetables and pulses.

### Red-Yellow Podzolic (RYP) soils
- **Where**: Dominant Great Soil Group of the Wet Zone.
- **Texture**: Acidic, well-leached, loam to sandy-loam.
- **Crops**: Tea (up-country and mid-country), rubber (mid- and low-country Wet Zone), cinnamon, rambutan, mangosteen, pepper, cocoa.

### Red-Yellow Latosols
- **Where**: Inter-zone transitions and older erosion surfaces.
- **Crops**: Cinnamon in Southern coastal belt and interior; spice mix.

### Alluvial soils
- **Where**: Flood plains of the Mahaweli, Kalu, Kelani, and other major rivers.
- **Crops**: Rice, banana, ash plantain, vegetables on well-drained levees.

### Old Alluvium, Solodized Solonetz, and others
- **Where**: Dry Zone, northern coastal belts, and locally in tank cascades.
- **Crops**: Rice, specialised crops; salt-affected soils restrict cropping options.

## Sources
- [Agri Farming — soils of Sri Lanka overview](https://www.agrifarming.in/agriculture-in-sri-lanka-major-crops-soil-types)
- [JIRCAS — Soil Constraints on Sustainable Plant Production in Sri Lanka](https://www.jircas.go.jp/sites/default/files/publication/tars/tars24-_12-29.pdf)
- [WUR — Soils of Ceylon](https://edepot.wur.nl/482354)
- [ResearchGate — RBE management for Uda Walawe sugarcane](https://www.researchgate.net/publication/327305808)

## Next step
Vectorise the NRMC soil map so `aez` records carry `dominant_soil_groups` as canonical slugs that resolve to polygon layers the app can query.
