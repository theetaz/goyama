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
};
