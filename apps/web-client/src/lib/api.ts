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
};
