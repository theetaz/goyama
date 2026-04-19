// Package geo serves the reverse-geocode / agro-context lookup that the rest
// of the platform builds on. Given a (lat, lng) anywhere in Sri Lanka the
// repo returns the surrounding administrative + agro-ecological envelope:
// district, DS division, AEZ code, dominant soil groups, elevation class,
// and rainfall normal. This is the Phase 0 exit-criterion endpoint.
package geo

import (
	"context"
	"errors"
)

// ErrLocationNotFound is returned when the point is outside Sri Lanka or
// the layer is not loaded for that area.
var ErrLocationNotFound = errors.New("location not covered by any administrative layer")

// ErrRequiresDatabase is returned by the stub repo when the API is running
// against the JSONL fallback (no DATABASE_URL). Geo lookup is Postgres-only
// because it depends on PostGIS spatial joins.
var ErrRequiresDatabase = errors.New("geo lookup requires Postgres (set DATABASE_URL and load geo layers)")

// Point is a WGS84 lat/lng pair.
type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// AdminDistrict is the district envelope returned by the lookup.
type AdminDistrict struct {
	Code         string `json:"code"`
	NameEN       string `json:"name_en"`
	NameSI       string `json:"name_si,omitempty"`
	NameTA       string `json:"name_ta,omitempty"`
	ProvinceCode string `json:"province_code,omitempty"`
	ProvinceName string `json:"province_name,omitempty"`
}

// DSDivision is the divisional-secretariat envelope returned by the lookup.
// May be empty when the DSD layer hasn't been loaded for that area.
type DSDivision struct {
	Code         string `json:"code"`
	DistrictCode string `json:"district_code,omitempty"`
	NameEN       string `json:"name_en"`
	NameSI       string `json:"name_si,omitempty"`
	NameTA       string `json:"name_ta,omitempty"`
}

// AEZ is the agro-ecological zone envelope, sourced from the existing aez
// table. Soil + rainfall normal travel with the AEZ record because Sri
// Lanka's AEZ dataset is the canonical resolution at which those normals
// are published.
type AEZ struct {
	Code               string   `json:"code"`
	ZoneGroup          string   `json:"zone_group"`      // 'wet' | 'intermediate' | 'dry'
	ElevationClass     string   `json:"elevation_class"` // 'low_country' | 'mid_country' | 'up_country'
	AvgRainfallMM      *float32 `json:"avg_rainfall_mm,omitempty"`
	AvgTemperatureC    *float32 `json:"avg_temperature_c,omitempty"`
	DominantSoilGroups []string `json:"dominant_soil_groups,omitempty"`
}

// Lookup is the response payload for GET /v1/geo/lookup. Each envelope is
// a pointer so the client can tell "not loaded for this area" (null) apart
// from "loaded but empty fields" (object with empty strings).
type Lookup struct {
	Location   Point          `json:"location"`
	District   *AdminDistrict `json:"district,omitempty"`
	DSDivision *DSDivision    `json:"ds_division,omitempty"`
	AEZ        *AEZ           `json:"aez,omitempty"`
}

// Repository is the geo lookup surface. Kept narrow on purpose — geo data
// is read-only at runtime; ingestion lives in cmd/geoload.
type Repository interface {
	Lookup(ctx context.Context, p Point) (Lookup, error)
}

// StubRepo satisfies Repository when no database is configured. Every call
// returns ErrRequiresDatabase so the JSONL-fallback API never silently
// returns "no data" for a geo lookup — that would be misleading.
type StubRepo struct{}

// NewStubRepo returns the JSONL-mode placeholder.
func NewStubRepo() *StubRepo { return &StubRepo{} }

// Lookup always returns ErrRequiresDatabase.
func (*StubRepo) Lookup(context.Context, Point) (Lookup, error) {
	return Lookup{}, ErrRequiresDatabase
}
