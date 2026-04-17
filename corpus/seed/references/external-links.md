# External reference links — Sri Lankan agri ecosystem

Live Sri Lankan agricultural digital resources worth linking to and watching for content. Per our rules, we **link to** these; we do not redistribute their content.

## Government portals

- [Department of Agriculture](https://doa.gov.lk/) — crop profiles, varieties, advisories.
  - RRDI (rice), HORDI (vegetables/fruits), FCRDI (field crops), FRDI (fruits), SRI (sugarcane), and research stations.
- [Ministry of Agriculture](https://www.agrimin.gov.lk/web/) — policy, release announcements.
- [Department of Export Agriculture](https://dea.gov.lk/) — spices and plantation crops.
- [Department of Agrarian Development](https://www.agrariandept.gov.lk/web/) — farmer services, ASCs.
- [Department of Census and Statistics](https://www.statistics.gov.lk/) — agri census, production stats.
- [Department of Meteorology](https://meteo.gov.lk/) — weather, climate normals.
- [HARTI (Hector Kobbekaduwa Agrarian Research & Training Institute)](https://harti.gov.lk/) — market price bulletins.
- [Natural Resources Management Centre](https://doa.gov.lk/nrmc/) — AEZ, soil, land use.

## Social media — DOA and institutional

- [Department of Agriculture Sri Lanka | Kandy — Facebook](https://www.facebook.com/SLKDOA/) — official DOA regional presence. Monitor for advisories and scan as a media source (do not redistribute images).

## Farmer digital services already in market

These are the competitive / complementary landscape to be aware of:

- [Govi Mithuru](https://www.dialog.lk/govi-mithuru) — Dialog Axiata SMS-based agri advisory (launched 2015). Tailored to farmer's crop and season.
- [Govi AI](https://goviai.com/) — "Sri Lanka's First Agriculture AI Assistant" for plant-disease detection in Sinhala and Tamil. Direct comparable to Goyama's scanner module; study their UX.
- Govi Gnana Seva — agri commodity price info service (Dialog Telekom partnership).
- [GeoGoviya Smart Farming Platform](https://www.agrarian.lk/) — Department of Agrarian Development digital initiative (2022).

## Global baselines

- [FAO GAEZ v4 Data Portal](https://gaez.fao.org/) — global suitability envelopes, AEZ backgrounds.
- [Wikidata](https://www.wikidata.org/) — taxonomic backbone for crops and pests.
- [iNaturalist](https://www.inaturalist.org/) — open biodiversity observations, images under compatible licences.
- [PlantVillage (Penn State)](https://plantvillage.psu.edu/) — baseline disease image dataset for scanner pretraining.

## YouTube content discovery (to instrument)

The YouTube Data API pipeline (per doc 06) will discover and curate channels by searching in Sinhala and Tamil for terms such as:

- "ගොවිතැන" (farming)
- "කෘෂිකර්ම" (agriculture)
- "වී" (paddy / rice)
- "කරවිල" (bitter gourd), "වම්බටු" (brinjal), "තක්කාලි" (tomato), etc.
- "வேளாண்மை" (Tamil: agriculture)
- "Department of Agriculture Sri Lanka"

Each discovered channel's videos will be stored as `media` records with `hosting: "external_link"`, metadata captured (title, channel, publish date, duration), transcripts generated via Whisper where licence permits, and all content linked back to the channel — never rehosted.
