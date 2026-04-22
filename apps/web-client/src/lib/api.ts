/**
 * API client for the Goyama Go service.
 *
 * Thin wrapper around fetch — returns typed objects and throws on non-2xx.
 * TanStack Query handles caching, retries, and background refetching.
 */

export interface CropSummary {
  slug: string;
  scientific_name?: string;
  category?: string;
  names?: Record<string, string>;
  aliases?: string[];
}

export interface CropDetail extends CropSummary {
  family?: string;
  life_cycle?: string;
  growth_habit?: string;
  default_season?: string;
  duration_days?: Range;
  elevation_m?: Range;
  rainfall_mm?: Range;
  temperature_c?: Range;
  soil_ph?: Range;
  expected_yield_kg_per_acre?: Range;
  description?: Record<string, string>;
  status?: string;
  field_provenance?: Record<string, unknown>;
}

export interface Range {
  min?: number;
  max?: number;
  unit?: string;
}

export interface CultivationStepInput {
  type:
    | 'seed'
    | 'fertilizer'
    | 'pesticide'
    | 'herbicide'
    | 'fungicide'
    | 'water'
    | 'labor'
    | 'mulch'
    | 'amendment'
    | 'tool';
  name: Record<string, string>;
  amount?: number;
  unit?: string;
  per_unit_area?: 'acre' | 'hectare' | 'perch' | 'square_meter';
  notes?: Record<string, string>;
}

export interface CultivationStep {
  slug: string;
  crop_slug: string;
  variety_slug?: string;
  aez_code?: string;
  season?: string;
  stage: string;
  order_idx: number;
  day_after_planting?: Range;
  title?: Record<string, string>;
  body?: Record<string, string>;
  inputs?: CultivationStepInput[];
  media_slugs?: string[];
  status?: string;
  field_provenance?: Record<string, unknown>;
}

export interface CultivationStepsResponse {
  crop_slug: string;
  items: CultivationStep[];
  count: number;
}

// Published-only farmer-facing surfaces. Records come back stripped of
// review fields (reviewed_by etc. are admin-only on the backend).

export interface DiseaseSummary {
  slug: string;
  scientific_name?: string;
  causal_organism: string;
  severity?: string;
  affected_crop_slugs?: string[];
  names?: Record<string, string>;
}

export interface DiseaseDetail extends DiseaseSummary {
  causal_species?: string;
  affected_parts?: string[];
  transmission?: string[];
  confused_with?: string[];
  favored_conditions?: Record<string, unknown>;
  aliases?: string[];
  description?: Record<string, string>;
  field_provenance?: Record<string, unknown>;
}

export interface DiseaseListResponse {
  items: DiseaseSummary[];
  count: number;
}

export interface PestSummary {
  slug: string;
  scientific_name?: string;
  kingdom: string;
  affected_crop_slugs?: string[];
  feeding_type?: string[];
  names?: Record<string, string>;
}

export interface PestDetail extends PestSummary {
  life_stages?: string[];
  favored_conditions?: Record<string, unknown>;
  aliases?: string[];
  description?: Record<string, string>;
  economic_threshold?: Record<string, string>;
  field_provenance?: Record<string, unknown>;
}

export interface PestListResponse {
  items: PestSummary[];
  count: number;
}

export type RemedyType =
  | 'cultural'
  | 'biological'
  | 'chemical'
  | 'resistant_variety'
  | 'mechanical'
  | 'integrated';

export interface RemedySummary {
  slug: string;
  type: RemedyType;
  active_ingredient?: string;
  pre_harvest_interval_days?: number;
  organic_compatible?: boolean;
  target_disease_slugs?: string[];
  target_pest_slugs?: string[];
  applicable_crop_slugs?: string[];
  name?: Record<string, string>;
}

export interface RemedyDetail extends RemedySummary {
  concentration?: string;
  formulation?: string;
  dosage?: string;
  application_method?: string;
  frequency?: string;
  re_entry_interval_hours?: number;
  who_hazard_class?: string;
  effectiveness?: string;
  cost_tier?: string;
  description?: Record<string, string>;
  instructions?: Record<string, string>;
  safety_notes?: Record<string, string>;
  field_provenance?: Record<string, unknown>;
}

export interface RemedyListResponse {
  items: RemedySummary[];
  count: number;
}

export interface CropListResponse {
  items: CropSummary[];
  count: number;
}

export interface GeoLookupRequest {
  lat: number;
  lng: number;
}

export interface GeoAdminDistrict {
  code: string;
  name_en: string;
  name_si?: string;
  name_ta?: string;
  province_code?: string;
  province_name?: string;
}

export interface GeoDSDivision {
  code: string;
  district_code?: string;
  name_en: string;
  name_si?: string;
  name_ta?: string;
}

export interface GeoAez {
  code: string;
  zone_group: 'wet' | 'intermediate' | 'dry';
  elevation_class: 'low_country' | 'mid_country' | 'up_country';
  avg_rainfall_mm?: number;
  avg_temperature_c?: number;
  dominant_soil_groups?: string[];
}

export interface GeoLookupResponse {
  location: { lat: number; lng: number };
  district?: GeoAdminDistrict;
  ds_division?: GeoDSDivision;
  aez?: GeoAez;
}

export interface ApiProblem {
  type: string;
  title: string;
  status: number;
  detail: string;
  instance: string;
  request_id?: string;
}

export class ApiError extends Error {
  readonly status: number;
  readonly problem: ApiProblem;
  constructor(problem: ApiProblem) {
    super(problem.detail || problem.title);
    this.name = 'ApiError';
    this.status = problem.status;
    this.problem = problem;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: 'include',
    headers: {
      Accept: 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  });
  if (!res.ok) {
    const problem = (await res.json().catch(() => ({
      type: 'unknown',
      title: res.statusText,
      status: res.status,
      detail: 'request failed',
      instance: path,
    }))) as ApiProblem;
    throw new ApiError(problem);
  }
  return (await res.json()) as T;
}

export const api = {
  health: () =>
    request<{ status: string; version: string; uptime_sec: number; time: string }>(
      '/v1/health',
    ),
  listCrops: (params: { category?: string; q?: string; limit?: number; offset?: number } = {}) => {
    const qs = new URLSearchParams();
    if (params.category) qs.set('category', params.category);
    if (params.q) qs.set('q', params.q);
    if (params.limit != null) qs.set('limit', String(params.limit));
    if (params.offset != null) qs.set('offset', String(params.offset));
    const suffix = qs.toString() ? `?${qs}` : '';
    return request<CropListResponse>(`/v1/crops${suffix}`);
  },
  getCrop: (slug: string) => request<CropDetail>(`/v1/crops/${encodeURIComponent(slug)}`),
  listCultivationSteps: (slug: string) =>
    request<CultivationStepsResponse>(
      `/v1/crops/${encodeURIComponent(slug)}/cultivation-steps`,
    ),
  listDiseases: () => request<DiseaseListResponse>('/v1/diseases'),
  getDisease: (slug: string) =>
    request<DiseaseDetail>(`/v1/diseases/${encodeURIComponent(slug)}`),
  listPests: () => request<PestListResponse>('/v1/pests'),
  getPest: (slug: string) => request<PestDetail>(`/v1/pests/${encodeURIComponent(slug)}`),
  listRemedies: () => request<RemedyListResponse>('/v1/remedies'),
  getRemedy: (slug: string) =>
    request<RemedyDetail>(`/v1/remedies/${encodeURIComponent(slug)}`),
  geoLookup: ({ lat, lng }: GeoLookupRequest) => {
    const qs = new URLSearchParams({ lat: String(lat), lng: String(lng) });
    return request<GeoLookupResponse>(`/v1/geo/lookup?${qs}`);
  },
  listMarketPrices: (params: {
    market?: string;
    crop?: string;
    since?: string;
    until?: string;
    limit?: number;
    offset?: number;
  } = {}) => {
    const qs = new URLSearchParams();
    if (params.market) qs.set('market', params.market);
    if (params.crop) qs.set('crop', params.crop);
    if (params.since) qs.set('since', params.since);
    if (params.until) qs.set('until', params.until);
    if (params.limit != null) qs.set('limit', String(params.limit));
    if (params.offset != null) qs.set('offset', String(params.offset));
    const suffix = qs.toString() ? `?${qs}` : '';
    return request<MarketPriceListResponse>(`/v1/market-prices${suffix}`);
  },
  latestMarketPrices: (market: string) =>
    request<MarketPriceLatestResponse>(
      `/v1/market-prices/latest/${encodeURIComponent(market)}`,
    ),
  listDiseaseImages: (slug: string) =>
    request<MediaListResponse>(`/v1/diseases/${encodeURIComponent(slug)}/images`),
  listCropCultivationPlans: (cropSlug: string) =>
    request<CultivationPlanListResponse>(
      `/v1/crops/${encodeURIComponent(cropSlug)}/cultivation-plans`,
    ),
  getCultivationPlan: (slug: string) =>
    request<CultivationPlan>(`/v1/cultivation-plans/${encodeURIComponent(slug)}`),
  listCropKnowledge: (cropSlug: string) =>
    request<KnowledgeResponse>(
      `/v1/crops/${encodeURIComponent(cropSlug)}/knowledge`,
    ),
  listDiseaseKnowledge: (slug: string) =>
    request<KnowledgeResponse>(
      `/v1/diseases/${encodeURIComponent(slug)}/knowledge`,
    ),
  listPestKnowledge: (slug: string) =>
    request<KnowledgeResponse>(
      `/v1/pests/${encodeURIComponent(slug)}/knowledge`,
    ),
};

// ─── cultivation plans ────────────────────────────────────────────────────

export type AuthorityLevel =
  | 'doa_official'
  | 'peer_reviewed'
  | 'regional_authority'
  | 'practitioner_report'
  | 'inferred_by_analogy'
  | 'agent_synthesis';

export interface CultivationPlanSummary {
  slug: string;
  crop_slug: string;
  season: string;
  authority: AuthorityLevel;
  aez_codes?: string[];
  title?: Record<string, string>;
  summary?: Record<string, string>;
  duration_weeks?: number;
  expected_yield_kg_per_acre?: Range;
  source_document_title?: string;
}

export interface CultivationActivityInput {
  type: string;
  name?: Record<string, string>;
  amount?: number;
  unit?: string;
  per_unit_area?: string;
  notes?: Record<string, string>;
}

export interface CultivationActivity {
  week_idx: number;
  order_in_week?: number;
  activity: string;
  dap_min?: number;
  dap_max?: number;
  title?: Record<string, string>;
  body?: Record<string, string>;
  inputs?: CultivationActivityInput[];
  weather_hint?: string;
  media_slugs?: string[];
}

export interface CultivationPestRisk {
  week_idx: number;
  disease_slug?: string;
  pest_slug?: string;
  risk: 'low' | 'moderate' | 'high';
  recommended_remedy_slugs?: string[];
  notes?: Record<string, string>;
}

export interface CultivationEconomicsCost {
  category: string;
  label?: Record<string, string>;
  amount: number;
  notes?: string;
}

export interface CultivationEconomics {
  reference_year: number;
  unit_area: string;
  currency: string;
  cost_lines?: CultivationEconomicsCost[];
  total_cost_without_family_labour?: number;
  total_cost_with_family_labour?: number;
  yield_kg?: number;
  unit_price?: number;
  gross_revenue?: number;
  net_revenue_without_family_labour?: number;
  net_revenue_with_family_labour?: number;
}

export interface CultivationPlan extends CultivationPlanSummary {
  variety_slug?: string;
  start_month?: number;
  status?: string;
  source_document_url?: string;
  activities: CultivationActivity[];
  pest_risks: CultivationPestRisk[];
  economics: CultivationEconomics[];
  field_provenance?: Record<string, unknown>;
}

export interface CultivationPlanListResponse {
  crop_slug: string;
  items: CultivationPlanSummary[];
  count: number;
}

// ─── knowledge graph ──────────────────────────────────────────────────────

export interface KnowledgeEntityRef {
  type: string;
  slug: string;
}

export interface KnowledgeChunk {
  slug: string;
  source_slug: string;
  chunk_idx?: number;
  language: string;
  title?: string;
  body: string;
  entity_refs?: KnowledgeEntityRef[];
  authority: AuthorityLevel;
  applies_to_aez_codes?: string[];
  applies_to_countries?: string[];
  topic_tags?: string[];
  confidence?: number;
  quote?: string;
  status: string;
}

export interface KnowledgeSource {
  slug: string;
  display_name: string;
  medium: string;
  publisher?: string;
  authority: AuthorityLevel;
  url?: string;
  language?: string;
  licence?: string;
  published_at?: string;
}

export interface KnowledgeResponse {
  entity_type: string;
  entity_slug: string;
  chunks: KnowledgeChunk[];
  sources: KnowledgeSource[];
  count: number;
}

export interface MediaItem {
  slug: string;
  type: string;
  hosting: 'own' | 'external_link';
  url?: string;
  external_url?: string;
  credit?: string;
  licence: string;
  caption?: Record<string, string>;
  tags?: string[];
}

export interface MediaListResponse {
  entity_type: string;
  entity_slug: string;
  items: MediaItem[];
  count: number;
}

export interface MarketPrice {
  market_code: string;
  crop_slug?: string;
  commodity_label: string;
  grade?: string;
  observed_on: string;
  price_lkr_per_kg_min?: number;
  price_lkr_per_kg_max?: number;
  price_lkr_per_kg_avg?: number;
  unit?: string;
  currency?: string;
  sample_size?: number;
  source_url?: string;
}

export interface MarketPriceListResponse {
  items: MarketPrice[];
  count: number;
}

export interface MarketPriceLatestResponse {
  market: string;
  items: MarketPrice[];
  count: number;
}
