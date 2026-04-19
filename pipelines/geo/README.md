# Geo layers

Loaders and source notes for the administrative + agro-ecological layers
that back `GET /v1/geo/lookup`.

## What gets loaded

| Layer | Table | Notes |
| --- | --- | --- |
| Districts | `admin_district` | 25 polygons (LK-11 … LK-93). Required for the lookup. |
| DS divisions | `admin_ds_division` | ~331 polygons. Optional — lookup degrades to district-only. |
| Agro-ecological zones | `aez` | 46 polygons (WL1 … DL5). Required for soil + rainfall context. |

`admin_district.code` follows ISO 3166-2 (`LK-11` for Colombo). `aez.code`
follows the standard NRMC convention (zone_group + elevation_class +
ordinal — e.g. `WL3` = wet zone, low country, sub-zone 3).

## Loading

```bash
# from repo root, after `make db-up && make db-migrate`
DATABASE_URL='postgres://goyama:goyama@localhost:54320/goyama?sslmode=disable' \
  go run ./services/api/cmd/geoload --layer=districts --file=pipelines/geo/fixtures/districts.geojson

DATABASE_URL='...' go run ./services/api/cmd/geoload --layer=aez --file=pipelines/geo/fixtures/aez.geojson
```

Or use the wrapper:

```bash
make db-load-geo-fixtures
```

Then probe:

```bash
curl 'http://localhost:8080/v1/geo/lookup?lat=7.29&lng=80.63'
```

The loader is idempotent — re-running on a refreshed GeoJSON upserts by
`code`. Geometry is normalised to `MultiPolygon` automatically.

## Sourcing real data

The fixtures under `fixtures/` are **simplified rectangles** for smoke
tests only — never publish anything geo-derived from them. Pull the
canonical layers from these sources before tagging a corpus release:

| Layer | Source | Format | Licence |
| --- | --- | --- | --- |
| Districts (25) | [Survey Department of Sri Lanka](https://www.survey.gov.lk/) digital boundary release; mirrored on GADM (`gadm.org/download_country.html?country=LKA`, level 2) | Shapefile / GeoJSON | GADM: free for academic + non-commercial; check Survey Dept terms for commercial use. ODbL acceptable for our redistribution if derived from OpenStreetMap. |
| DS divisions (~331) | Department of Census and Statistics — DSD shapefile bundle | Shapefile | Public sector — request the redistribution-cleared release. |
| AEZ (46 zones) | [Natural Resources Management Centre (NRMC), Department of Agriculture](https://doa.gov.lk/nrmc/) — AEZ map of Sri Lanka (Punyawardena, 2008 update) | Shapefile / PDF map | Public sector — confirm redistribution terms before publishing. |

### Conversion to GeoJSON

For shapefile sources, convert with `ogr2ogr` (GDAL) and reproject to WGS84:

```bash
ogr2ogr -f GeoJSON -t_srs EPSG:4326 \
  -lco RFC7946=YES \
  data/raw/geo/sl-districts.geojson \
  /path/to/sl_districts.shp

# rename properties to the loader's expected schema (code, name_en, ...)
# either via -sql or with a follow-up jq pass
```

Required GeoJSON `properties` per layer:

| Layer | Required | Optional |
| --- | --- | --- |
| `districts` | `code`, `name_en` | `name_si`, `name_ta`, `province_code`, `province_name`, `field_provenance` |
| `ds-divisions` | `code`, `district_code`, `name_en` | `name_si`, `name_ta`, `field_provenance` |
| `aez` | `code`, `zone_group` (`wet`\|`intermediate`\|`dry`), `elevation_class` (`low_country`\|`mid_country`\|`up_country`) | `avg_rainfall_mm`, `avg_temperature_c`, `dominant_soil_groups[]`, `field_provenance` |

Real source files (`data/raw/geo/*.geojson`) live under `data/raw/` which
is gitignored. Only the small dev fixtures here are committed.
