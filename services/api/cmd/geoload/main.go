// Command geoload ingests a GeoJSON FeatureCollection into the Goyama
// geo layer tables (admin_district, admin_ds_division, aez).
//
// Usage:
//
//	geoload --layer=districts    --file=data/raw/geo/sl-districts.geojson
//	geoload --layer=ds-divisions --file=data/raw/geo/sl-ds-divisions.geojson
//	geoload --layer=aez          --file=data/raw/geo/sl-aez.geojson
//
// Reads DATABASE_URL from the environment. Loader is idempotent — each
// feature is upserted by its `code` property, so re-running on a refreshed
// GeoJSON safely overwrites existing rows. Geometry is normalised to
// MultiPolygon via ST_Multi so the schema's geography(MultiPolygon, 4326)
// column accepts both Polygon and MultiPolygon inputs.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type feature struct {
	Type       string          `json:"type"`
	Geometry   json.RawMessage `json:"geometry"`
	Properties map[string]any  `json:"properties"`
}

type featureCollection struct {
	Type     string    `json:"type"`
	Features []feature `json:"features"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	layer := flag.String("layer", "", "one of: districts, ds-divisions, aez")
	path := flag.String("file", "", "path to a GeoJSON FeatureCollection")
	flag.Parse()

	if *layer == "" || *path == "" {
		flag.Usage()
		return fmt.Errorf("both --layer and --file are required")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	fc, err := readFeatureCollection(*path)
	if err != nil {
		return fmt.Errorf("read %s: %w", *path, err)
	}
	if len(fc.Features) == 0 {
		return fmt.Errorf("no features in %s", *path)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("pgx pool: %w", err)
	}
	defer pool.Close()

	loader, err := loaderFor(*layer)
	if err != nil {
		return err
	}

	inserted := 0
	for i, f := range fc.Features {
		if len(f.Geometry) == 0 {
			return fmt.Errorf("feature %d: missing geometry", i)
		}
		if err := loader(ctx, pool, f); err != nil {
			return fmt.Errorf("feature %d: %w", i, err)
		}
		inserted++
	}
	fmt.Printf("loaded %d features into layer %q from %s\n", inserted, *layer, *path)
	return nil
}

func readFeatureCollection(path string) (*featureCollection, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	body, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var fc featureCollection
	if err := json.Unmarshal(body, &fc); err != nil {
		return nil, fmt.Errorf("decode geojson: %w", err)
	}
	if !strings.EqualFold(fc.Type, "FeatureCollection") {
		return nil, fmt.Errorf("expected FeatureCollection, got %q", fc.Type)
	}
	return &fc, nil
}

type loaderFn func(context.Context, *pgxpool.Pool, feature) error

func loaderFor(layer string) (loaderFn, error) {
	switch strings.ToLower(layer) {
	case "districts":
		return loadDistrict, nil
	case "ds-divisions", "ds_divisions", "dsd":
		return loadDSDivision, nil
	case "aez":
		return loadAEZ, nil
	default:
		return nil, fmt.Errorf("unknown layer %q (want districts | ds-divisions | aez)", layer)
	}
}

const upsertDistrictSQL = `
INSERT INTO admin_district (code, name_en, name_si, name_ta, province_code, province_name, geom, field_provenance, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, ST_Multi(ST_GeomFromGeoJSON($7))::geography, $8, now())
ON CONFLICT (code) DO UPDATE SET
    name_en = EXCLUDED.name_en,
    name_si = EXCLUDED.name_si,
    name_ta = EXCLUDED.name_ta,
    province_code = EXCLUDED.province_code,
    province_name = EXCLUDED.province_name,
    geom = EXCLUDED.geom,
    field_provenance = EXCLUDED.field_provenance,
    updated_at = now()
`

func loadDistrict(ctx context.Context, pool *pgxpool.Pool, f feature) error {
	code, err := requireStr(f.Properties, "code")
	if err != nil {
		return err
	}
	nameEN, err := requireStr(f.Properties, "name_en")
	if err != nil {
		return err
	}
	prov, _ := f.Properties["field_provenance"].(map[string]any)
	if prov == nil {
		prov = map[string]any{}
	}
	provJSON, _ := json.Marshal(prov)

	_, err = pool.Exec(ctx, upsertDistrictSQL,
		code, nameEN,
		nullableStr(f.Properties, "name_si"),
		nullableStr(f.Properties, "name_ta"),
		nullableStr(f.Properties, "province_code"),
		nullableStr(f.Properties, "province_name"),
		string(f.Geometry), provJSON,
	)
	return err
}

const upsertDSDSQL = `
INSERT INTO admin_ds_division (code, district_code, name_en, name_si, name_ta, geom, field_provenance, updated_at)
VALUES ($1, $2, $3, $4, $5, ST_Multi(ST_GeomFromGeoJSON($6))::geography, $7, now())
ON CONFLICT (code) DO UPDATE SET
    district_code = EXCLUDED.district_code,
    name_en = EXCLUDED.name_en,
    name_si = EXCLUDED.name_si,
    name_ta = EXCLUDED.name_ta,
    geom = EXCLUDED.geom,
    field_provenance = EXCLUDED.field_provenance,
    updated_at = now()
`

func loadDSDivision(ctx context.Context, pool *pgxpool.Pool, f feature) error {
	code, err := requireStr(f.Properties, "code")
	if err != nil {
		return err
	}
	district, err := requireStr(f.Properties, "district_code")
	if err != nil {
		return err
	}
	nameEN, err := requireStr(f.Properties, "name_en")
	if err != nil {
		return err
	}
	prov, _ := f.Properties["field_provenance"].(map[string]any)
	if prov == nil {
		prov = map[string]any{}
	}
	provJSON, _ := json.Marshal(prov)

	_, err = pool.Exec(ctx, upsertDSDSQL,
		code, district, nameEN,
		nullableStr(f.Properties, "name_si"),
		nullableStr(f.Properties, "name_ta"),
		string(f.Geometry), provJSON,
	)
	return err
}

const upsertAEZSQL = `
INSERT INTO aez (code, version, status, zone_group, elevation_class,
                 avg_rainfall_mm, avg_temperature_c, dominant_soil_groups,
                 geom, field_provenance, updated_at)
VALUES ($1, 1, 'published',
        $2::zone_group, $3::elevation_class,
        $4, $5, $6,
        ST_Multi(ST_GeomFromGeoJSON($7))::geography,
        $8, now())
ON CONFLICT (code, version) DO UPDATE SET
    status = EXCLUDED.status,
    zone_group = EXCLUDED.zone_group,
    elevation_class = EXCLUDED.elevation_class,
    avg_rainfall_mm = EXCLUDED.avg_rainfall_mm,
    avg_temperature_c = EXCLUDED.avg_temperature_c,
    dominant_soil_groups = EXCLUDED.dominant_soil_groups,
    geom = EXCLUDED.geom,
    field_provenance = EXCLUDED.field_provenance,
    updated_at = now()
`

func loadAEZ(ctx context.Context, pool *pgxpool.Pool, f feature) error {
	code, err := requireStr(f.Properties, "code")
	if err != nil {
		return err
	}
	zoneGroup, err := requireStr(f.Properties, "zone_group")
	if err != nil {
		return err
	}
	elevationClass, err := requireStr(f.Properties, "elevation_class")
	if err != nil {
		return err
	}
	prov, _ := f.Properties["field_provenance"].(map[string]any)
	if prov == nil {
		prov = map[string]any{}
	}
	provJSON, _ := json.Marshal(prov)

	soils := stringSlice(f.Properties, "dominant_soil_groups")

	_, err = pool.Exec(ctx, upsertAEZSQL,
		code, zoneGroup, elevationClass,
		nullableFloat(f.Properties, "avg_rainfall_mm"),
		nullableFloat(f.Properties, "avg_temperature_c"),
		soils,
		string(f.Geometry), provJSON,
	)
	return err
}

func requireStr(props map[string]any, key string) (string, error) {
	v, ok := props[key].(string)
	if !ok || v == "" {
		return "", fmt.Errorf("missing required property %q", key)
	}
	return v, nil
}

func nullableStr(props map[string]any, key string) *string {
	v, ok := props[key].(string)
	if !ok || v == "" {
		return nil
	}
	return &v
}

func nullableFloat(props map[string]any, key string) *float64 {
	switch v := props[key].(type) {
	case float64:
		return &v
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return nil
		}
		return &f
	}
	return nil
}

func stringSlice(props map[string]any, key string) []string {
	raw, ok := props[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
