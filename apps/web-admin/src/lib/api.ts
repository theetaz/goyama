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

export interface CropListResponse {
  items: CropSummary[];
  count: number;
}

export type RecordStatus =
  | 'draft'
  | 'in_review'
  | 'published'
  | 'deprecated'
  | 'rejected';

export interface CultivationStepInput {
  type: string;
  name: Record<string, string>;
  amount?: number;
  unit?: string;
  per_unit_area?: string;
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
  status: RecordStatus;
  field_provenance?: Record<string, FieldProvenance>;
  reviewed_by?: string;
  reviewed_at?: string;
  review_notes?: string;
}

export interface FieldProvenance {
  source_id: string;
  source_url: string;
  fetched_at: string;
  quote?: string;
  extractor_version?: string;
  model_id?: string;
  confidence?: number;
  reviewed_by?: string;
  reviewed_at?: string;
  review_notes?: string;
}

export interface ReviewQueueResponse {
  status: RecordStatus;
  items: CultivationStep[];
  count: number;
}

// ReviewerIdentity is stored in localStorage so an agronomist only types
// their email once per browser. Swap for real SSO identity in a future PR.
const REVIEWER_KEY = 'goyama.admin.reviewer';

export function getReviewer(): string {
  if (typeof window === 'undefined') return '';
  return window.localStorage.getItem(REVIEWER_KEY) ?? '';
}

export function setReviewer(email: string): void {
  if (typeof window === 'undefined') return;
  if (email) window.localStorage.setItem(REVIEWER_KEY, email);
  else window.localStorage.removeItem(REVIEWER_KEY);
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
  const headers: Record<string, string> = {
    Accept: 'application/json',
    ...((init?.headers as Record<string, string>) ?? {}),
  };
  // Attach the reviewer header to every admin-scoped request so we never
  // forget it on a mutation. The backend ignores it for public endpoints.
  const reviewer = getReviewer();
  if (reviewer) headers['X-Goyama-Reviewer'] = reviewer;

  const res = await fetch(path, {
    credentials: 'include',
    ...init,
    headers,
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
  listCultivationStepsForReview: (status: RecordStatus = 'draft') =>
    request<ReviewQueueResponse>(
      `/v1/admin/cultivation-steps?status=${encodeURIComponent(status)}`,
    ),
  getCultivationStep: (slug: string) =>
    request<CultivationStep>(`/v1/admin/cultivation-steps/${encodeURIComponent(slug)}`),
  updateCultivationStepStatus: (
    slug: string,
    body: { status: RecordStatus; review_notes?: string },
  ) =>
    request<CultivationStep>(`/v1/admin/cultivation-steps/${encodeURIComponent(slug)}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),
};
