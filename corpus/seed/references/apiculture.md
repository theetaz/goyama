# Apiculture (beekeeping) — reference notes

Beekeeping is a significant part of Sri Lankan agriculture, both as a honey-producing activity and as a critical pollination service for major field, vegetable, fruit, and plantation crops. These notes sit outside the main crop/disease/pest schema tables pending a future `pollinator` / `livestock` entity type.

## Primary bee species in Sri Lanka

| Species | Local name | Role |
|---|---|---|
| **Apis cerana indica** | Mee Messa | Asiatic honey bee — **primary managed species** in Sri Lanka beekeeping. Disease-resistant relative to *Apis mellifera*; suited to small-scale hives. |
| *Apis dorsata* | Bambara | Giant rock bee — wild, honey harvested from cliff-face combs ("bambara"). |
| *Apis florea* | Dandu Messa | Little bee — builds small open combs; wild. |
| *Trigona iridipennis* | Kana Messa | Stingless bee — small-scale hive production; prized for medicinal honey. |

## Where beekeeping is practised

Part-time activity across most of the island, concentrated in **Eucalyptus and rubber-plantation zones** — the major honey-producing areas. Also common in coconut smallholdings where bees boost pollination and nut yield.

## Pollination services

Most cultivated vegetables, cucurbits, and coconut depend on bees for pollination. Historically coconut estates reared bees specifically to improve yield.

## Key management problems

- **Absconding** — most serious challenge for *A. cerana indica* in Sri Lanka. Triggers: nectar shortage, genetics, frequent disturbance, pests (Wax Moths, Hornets, Ants).
- Queen quality, pollen shortage, droughts, heavy monsoon rains — all reduce yields.
- Pests: wax moths, hornets, ants, small hive beetle.

## Research context

Protein-supplement feeding trials to improve brood area, colony population, colony weight, and honey yield are ongoing.

## Sources

- [Issuu — Beekeeping in Sri Lanka (Bee Farmer Digest 1991)](https://issuu.com/beesfd/docs/19_bfd_june1991/s/14515341)
- [Caritas Sri Lanka — Beekeeping training](https://www.caritaslk.org/food-security/natures-bountiful-gift-caritas-training-on-bee-keeping/)
- Punchihewa, R.W.K. — *Beekeeping for Honey Production in Sri Lanka: Management of the Asiatic Hive Honey Bee* (Apis cerana) — Google Books [link](https://books.google.com/books/about/Beekeeping_for_Honey_Production_in_Sri_L.html?id=m_BJAAAAYAAJ)
- [Wikipedia — Apis cerana indica](https://en.wikipedia.org/wiki/Apis_cerana_indica)

## Next steps

- Add a canonical `pollinator` entity type (or extend `pest`-like schema with a `beneficial_insect` type).
- Catalogue pollinator-dependence scores for each crop in the main `crop` records (e.g. coconut, cucurbits, mango, citrus, brinjal — all high).
- Geo-link hive-management advice to apiary-zone maps.
