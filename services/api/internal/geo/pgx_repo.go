package geo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxRepo serves geo lookups via PostGIS spatial joins.
//
// One round-trip resolves all three layers using LATERAL joins; missing
// layers come back as NULL rather than failing the whole query, so a
// partially-loaded environment (say, AEZ only) still answers usefully.
type PgxRepo struct {
	pool *pgxpool.Pool
}

// NewPgxRepo returns a repo over the given pool. The pool lifecycle is
// owned by the caller.
func NewPgxRepo(pool *pgxpool.Pool) *PgxRepo {
	return &PgxRepo{pool: pool}
}

// lookupSQL composes the point once and reuses it across the three layer
// joins. ST_Covers is used instead of ST_Intersects so points exactly on a
// shared boundary resolve to a single polygon (deterministic on borders).
const lookupSQL = `
WITH p AS (
    SELECT ST_SetSRID(ST_MakePoint($1::float8, $2::float8), 4326)::geography AS pt
)
SELECT
    d.code, d.name_en, d.name_si, d.name_ta, d.province_code, d.province_name,
    s.code, s.district_code, s.name_en, s.name_si, s.name_ta,
    a.code, a.zone_group::text, a.elevation_class::text,
    a.avg_rainfall_mm, a.avg_temperature_c, a.dominant_soil_groups
FROM p
LEFT JOIN LATERAL (
    SELECT * FROM admin_district
    WHERE ST_Covers(geom, p.pt)
    LIMIT 1
) d ON TRUE
LEFT JOIN LATERAL (
    SELECT * FROM admin_ds_division
    WHERE ST_Covers(geom, p.pt)
    LIMIT 1
) s ON TRUE
LEFT JOIN LATERAL (
    SELECT * FROM aez
    WHERE ST_Covers(geom, p.pt)
    ORDER BY (status = 'published') DESC, version DESC
    LIMIT 1
) a ON TRUE
`

// Lookup resolves the (lat, lng) into administrative + agro-ecological
// envelopes. Returns ErrLocationNotFound when no layer covers the point —
// that is the signal callers use to render "outside Sri Lanka or
// unsupported area" rather than a generic 500.
func (r *PgxRepo) Lookup(ctx context.Context, p Point) (Lookup, error) {
	out := Lookup{Location: p}

	var (
		dCode, dNameEN, dNameSI, dNameTA, dProvCode, dProvName *string
		sCode, sDistrictCode, sNameEN, sNameSI, sNameTA        *string
		aCode, aZoneGroup, aElevClass                          *string
		aRainfall, aTemp                                       *float32
		aSoils                                                 []string
	)

	// pgx parameter order: $1 = lng (X), $2 = lat (Y).
	err := r.pool.QueryRow(ctx, lookupSQL, p.Lng, p.Lat).Scan(
		&dCode, &dNameEN, &dNameSI, &dNameTA, &dProvCode, &dProvName,
		&sCode, &sDistrictCode, &sNameEN, &sNameSI, &sNameTA,
		&aCode, &aZoneGroup, &aElevClass,
		&aRainfall, &aTemp, &aSoils,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Lookup{}, ErrLocationNotFound
		}
		return Lookup{}, fmt.Errorf("geo lookup: %w", err)
	}

	if dCode != nil {
		out.District = &AdminDistrict{
			Code:         *dCode,
			NameEN:       strOr(dNameEN),
			NameSI:       strOr(dNameSI),
			NameTA:       strOr(dNameTA),
			ProvinceCode: strOr(dProvCode),
			ProvinceName: strOr(dProvName),
		}
	}
	if sCode != nil {
		out.DSDivision = &DSDivision{
			Code:         *sCode,
			DistrictCode: strOr(sDistrictCode),
			NameEN:       strOr(sNameEN),
			NameSI:       strOr(sNameSI),
			NameTA:       strOr(sNameTA),
		}
	}
	if aCode != nil {
		out.AEZ = &AEZ{
			Code:               *aCode,
			ZoneGroup:          strOr(aZoneGroup),
			ElevationClass:     strOr(aElevClass),
			AvgRainfallMM:      aRainfall,
			AvgTemperatureC:    aTemp,
			DominantSoilGroups: aSoils,
		}
	}

	if out.District == nil && out.DSDivision == nil && out.AEZ == nil {
		return Lookup{}, ErrLocationNotFound
	}
	return out, nil
}

func strOr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
