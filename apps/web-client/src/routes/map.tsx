import { useMemo, useState } from 'react';
import { Link, createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import Map, {
  Layer,
  type MapLayerMouseEvent,
  NavigationControl,
  Popup,
  Source,
} from 'react-map-gl/maplibre';
import type { FillLayerSpecification, LineLayerSpecification } from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';

import {
  sriLankaAez,
  sriLankaBounds,
  sriLankaCenter,
  type AezProperties,
} from '@/data/sri-lanka-aez';
import {
  ApiError,
  api,
  type CropSummary,
  type GeoLookupResponse,
} from '@/lib/api';
import { pickLocalised, type Locale } from '@/i18n';
import { cn } from '@/lib/utils';

export const Route = createFileRoute('/map')({
  component: MapPage,
});

/**
 * Interactive Sri Lanka map — the headline home surface per docs/10.
 * First pass: OpenFreeMap base tiles + hand-authored AEZ overlay + tap-to-zone
 * cards that pull live recommendations from the Goyama API. Will be replaced by
 * the NRMC polygon dataset and the production recommender as they land.
 */
function MapPage() {
  const { t, i18n } = useTranslation();
  const locale = i18n.language as Locale;
  const [selected, setSelected] = useState<{
    point: { longitude: number; latitude: number };
    aez: AezProperties | null;
  } | null>(null);

  const aezCrops = useQuery({
    queryKey: ['crops-by-aez', selected?.aez?.group],
    queryFn: () => api.listCrops({ category: 'vegetable', limit: 6 }),
    enabled: selected != null,
  });

  // Live administrative + agro-ecological lookup against the Go API. Decoupled
  // from the hand-authored AEZ overlay above — this returns the canonical
  // district / DSD / AEZ envelope from PostGIS once geo layers are loaded
  // server-side. Falls back to a friendly status when the API is in JSONL
  // mode (503) or the point falls outside loaded layers (404).
  const geoLookup = useQuery({
    queryKey: ['geo-lookup', selected?.point.latitude, selected?.point.longitude],
    queryFn: () =>
      api.geoLookup({
        lat: selected!.point.latitude,
        lng: selected!.point.longitude,
      }),
    enabled: selected != null,
    retry: false,
  });

  // Match zone group to a CSS variable from packages/design-tokens.
  const aezFillPaint = useMemo<FillLayerSpecification['paint']>(
    () => ({
      'fill-color': [
        'match',
        ['get', 'group'],
        'wet',
        'hsl(150 55% 52%)',
        'intermediate',
        'hsl(95 58% 64%)',
        'dry',
        'hsl(60 60% 62%)',
        '#888',
      ],
      'fill-opacity': 0.28,
    }),
    [],
  );

  const aezOutlinePaint = useMemo<LineLayerSpecification['paint']>(
    () => ({
      'line-color': [
        'match',
        ['get', 'group'],
        'wet',
        'hsl(150 55% 32%)',
        'intermediate',
        'hsl(95 45% 42%)',
        'dry',
        'hsl(60 60% 36%)',
        '#666',
      ],
      'line-width': 1.5,
    }),
    [],
  );

  function handleClick(evt: MapLayerMouseEvent) {
    const feature = evt.features?.[0];
    setSelected({
      point: { longitude: evt.lngLat.lng, latitude: evt.lngLat.lat },
      aez: (feature?.properties as AezProperties) ?? null,
    });
  }

  return (
    <div className="space-y-4">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">{t('map.title')}</h1>
          <p className="max-w-2xl text-sm text-muted-foreground">{t('map.subtitle')}</p>
        </div>
        <Link
          to="/"
          className="rounded-md border px-3 py-1.5 text-sm text-muted-foreground hover:bg-muted"
        >
          {t('nav.explore')}
        </Link>
      </header>

      <div className="flex flex-wrap gap-3 text-xs">
        <AezBadge colour="hsl(150 55% 52%)" label={t('map.legend_wet')} />
        <AezBadge colour="hsl(95 58% 64%)" label={t('map.legend_intermediate')} />
        <AezBadge colour="hsl(60 60% 62%)" label={t('map.legend_dry')} />
        <span className="text-muted-foreground">{t('map.legend_tap_hint')}</span>
      </div>

      <div className="relative h-[70vh] overflow-hidden rounded-xl border bg-card">
        <Map
          initialViewState={{
            longitude: sriLankaCenter.longitude,
            latitude: sriLankaCenter.latitude,
            zoom: 7.3,
          }}
          maxBounds={sriLankaBounds}
          mapStyle="https://tiles.openfreemap.org/styles/liberty"
          interactiveLayerIds={['aez-fill']}
          onClick={handleClick}
          style={{ width: '100%', height: '100%' }}
        >
          <NavigationControl position="top-right" showCompass={false} />
          <Source id="aez" type="geojson" data={sriLankaAez}>
            <Layer id="aez-fill" type="fill" paint={aezFillPaint} />
            <Layer id="aez-outline" type="line" paint={aezOutlinePaint} />
          </Source>

          {selected && (
            <Popup
              longitude={selected.point.longitude}
              latitude={selected.point.latitude}
              anchor="bottom"
              closeOnClick={false}
              onClose={() => setSelected(null)}
              maxWidth="340px"
            >
              <PopupContent
                aez={selected.aez}
                locale={locale}
                crops={aezCrops.data?.items}
                loading={aezCrops.isLoading}
                geo={geoLookup.data}
                geoLoading={geoLookup.isLoading}
                geoError={geoLookup.error}
              />
            </Popup>
          )}
        </Map>
      </div>

      <p className="text-xs text-muted-foreground">{t('map.disclaimer')}</p>
    </div>
  );
}

function AezBadge({ colour, label }: { colour: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full border bg-background px-2 py-1">
      <span
        aria-hidden
        className="inline-block h-3 w-3 rounded-full"
        style={{ backgroundColor: colour }}
      />
      {label}
    </span>
  );
}

function PopupContent({
  aez,
  locale,
  crops,
  loading,
  geo,
  geoLoading,
  geoError,
}: {
  aez: AezProperties | null;
  locale: Locale;
  crops: CropSummary[] | undefined;
  loading: boolean;
  geo: GeoLookupResponse | undefined;
  geoLoading: boolean;
  geoError: unknown;
}) {
  const { t } = useTranslation();

  if (!aez) {
    return (
      <div className="p-2 text-sm">
        <p>{t('map.popup_off_zone')}</p>
        <GeoEnvelope geo={geo} loading={geoLoading} error={geoError} locale={locale} />
      </div>
    );
  }
  const name =
    locale === 'si' ? aez.name_si : locale === 'ta' ? aez.name_ta : aez.name;

  return (
    <div className="p-1 text-sm text-foreground">
      <div className="mb-1 flex items-center gap-2">
        <span
          aria-hidden
          className={cn(
            'h-2.5 w-2.5 rounded-full',
            aez.group === 'wet' && 'bg-[hsl(150_55%_52%)]',
            aez.group === 'intermediate' && 'bg-[hsl(95_58%_64%)]',
            aez.group === 'dry' && 'bg-[hsl(60_60%_62%)]',
          )}
        />
        <strong className="text-base">{name}</strong>
      </div>
      <p className="text-xs text-muted-foreground">{aez.summary}</p>

      <GeoEnvelope geo={geo} loading={geoLoading} error={geoError} locale={locale} />

      <hr className="my-2" />
      <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {t('map.popup_sample_crops')}
      </div>
      {loading && <p className="mt-1 text-sm">{t('crops.loading')}</p>}
      {!loading && crops && (
        <ul className="mt-1 grid grid-cols-2 gap-x-3 gap-y-0.5 text-sm">
          {crops.slice(0, 6).map((c) => {
            const n = pickLocalised(c.names, locale) ?? c.slug;
            return (
              <li key={c.slug}>
                <Link
                  to="/crops/$slug"
                  params={{ slug: c.slug }}
                  className="text-primary hover:underline"
                >
                  {n}
                </Link>
              </li>
            );
          })}
        </ul>
      )}
      <p className="mt-2 text-[11px] italic text-muted-foreground">
        {t('map.popup_placeholder_note')}
      </p>
    </div>
  );
}

/**
 * Renders the live `/v1/geo/lookup` envelope: district, DS division, and the
 * canonical AEZ from PostGIS. Stays out of the way until the lookup
 * resolves so the popup feels instant from the cached overlay; then folds
 * the authoritative envelope in. Soft-fails on 404 + 503 — the popup is
 * still useful with just the hand-authored zone.
 */
function GeoEnvelope({
  geo,
  loading,
  error,
  locale,
}: {
  geo: GeoLookupResponse | undefined;
  loading: boolean;
  error: unknown;
  locale: Locale;
}) {
  const { t } = useTranslation();

  if (loading) return null;

  if (error instanceof ApiError) {
    if (error.status === 404) {
      return (
        <p className="mt-2 text-[11px] italic text-muted-foreground">
          {t('map.geo_no_coverage')}
        </p>
      );
    }
    if (error.status === 503) {
      return (
        <p className="mt-2 text-[11px] italic text-muted-foreground">
          {t('map.geo_disabled')}
        </p>
      );
    }
  }

  if (!geo || (!geo.district && !geo.ds_division && !geo.aez)) return null;

  const districtName = geo.district
    ? (locale === 'si' && geo.district.name_si) ||
      (locale === 'ta' && geo.district.name_ta) ||
      geo.district.name_en
    : null;

  const dsdName = geo.ds_division
    ? (locale === 'si' && geo.ds_division.name_si) ||
      (locale === 'ta' && geo.ds_division.name_ta) ||
      geo.ds_division.name_en
    : null;

  return (
    <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 text-xs">
      {districtName && (
        <>
          <dt className="font-medium text-muted-foreground">{t('map.geo_district')}</dt>
          <dd>
            {districtName}
            {geo.district?.province_name ? ` · ${geo.district.province_name}` : ''}
          </dd>
        </>
      )}
      {dsdName && (
        <>
          <dt className="font-medium text-muted-foreground">{t('map.geo_dsd')}</dt>
          <dd>{dsdName}</dd>
        </>
      )}
      {geo.aez && (
        <>
          <dt className="font-medium text-muted-foreground">{t('map.geo_aez')}</dt>
          <dd>
            {geo.aez.code} · {geo.aez.zone_group} · {geo.aez.elevation_class.replace('_', ' ')}
          </dd>
        </>
      )}
      {geo.aez?.avg_rainfall_mm != null && (
        <>
          <dt className="font-medium text-muted-foreground">{t('map.geo_rainfall')}</dt>
          <dd>{Math.round(geo.aez.avg_rainfall_mm)} mm/yr</dd>
        </>
      )}
      {geo.aez?.dominant_soil_groups?.length ? (
        <>
          <dt className="font-medium text-muted-foreground">{t('map.geo_soil')}</dt>
          <dd>{geo.aez.dominant_soil_groups.join(', ').replaceAll('_', ' ')}</dd>
        </>
      ) : null}
    </dl>
  );
}
