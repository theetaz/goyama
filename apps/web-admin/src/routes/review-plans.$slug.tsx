import { useState } from 'react';
import { Link, createFileRoute } from '@tanstack/react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft, ExternalLink, ShieldAlert } from 'lucide-react';

import {
  ApiError,
  api,
  getReviewer,
  type CultivationActivity,
  type CultivationActivityInput,
  type CultivationEconomics,
  type CultivationPestRisk,
  type CultivationPlanReview,
  type RecordStatus,
} from '@/lib/api';
import { AuthorityChip } from '@/components/authority-chip';

export const Route = createFileRoute('/review-plans/$slug')({
  component: PlanReviewDetailPage,
});

// Same transition table as review-diseases — keep the backend authoritative
// on validation, but gate UI buttons to the moves that can succeed.
const ACTIONS: Record<RecordStatus, { to: RecordStatus; label: string; tone: 'primary' | 'neutral' | 'destructive' }[]> = {
  draft: [
    { to: 'in_review', label: 'Move to in review', tone: 'neutral' },
    { to: 'published', label: 'Promote to published', tone: 'primary' },
    { to: 'rejected', label: 'Reject', tone: 'destructive' },
  ],
  in_review: [
    { to: 'published', label: 'Promote to published', tone: 'primary' },
    { to: 'draft', label: 'Send back to draft', tone: 'neutral' },
    { to: 'rejected', label: 'Reject', tone: 'destructive' },
  ],
  published: [{ to: 'deprecated', label: 'Deprecate', tone: 'destructive' }],
  deprecated: [{ to: 'published', label: 'Restore to published', tone: 'primary' }],
  rejected: [{ to: 'draft', label: 'Move back to draft', tone: 'neutral' }],
};

function PlanReviewDetailPage() {
  const { slug } = Route.useParams();
  const qc = useQueryClient();
  const [notes, setNotes] = useState('');

  const plan = useQuery({
    queryKey: ['plan-review', slug],
    queryFn: () => api.getCultivationPlanForReview(slug),
  });

  const mutation = useMutation({
    mutationFn: (to: RecordStatus) =>
      api.updateCultivationPlanStatus(slug, {
        status: to,
        review_notes: notes.trim() || undefined,
      }),
    onSuccess: (updated) => {
      qc.setQueryData(['plan-review', slug], updated);
      qc.invalidateQueries({ queryKey: ['plan-review-queue'] });
      setNotes('');
    },
  });

  const reviewer = getReviewer();

  return (
    <div className="space-y-5">
      <Link
        to="/review-plans"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ChevronLeft className="h-4 w-4" aria-hidden />
        Plan review
      </Link>

      {plan.isLoading && <p>Loading…</p>}
      {plan.isError && (
        <p role="alert" className="text-destructive">
          {plan.error instanceof Error ? plan.error.message : 'Failed to load plan'}
        </p>
      )}

      {plan.data && (
        <article className="grid grid-cols-1 gap-4 lg:grid-cols-[1fr_340px]">
          <div className="space-y-4">
            <PlanCard plan={plan.data} />
            <ActivitiesPanel activities={plan.data.activities} />
            <PestRisksPanel risks={plan.data.pest_risks} />
            {plan.data.economics.length > 0 && (
              <EconomicsPanel economics={plan.data.economics[0]} />
            )}
          </div>

          <aside className="h-fit rounded-xl border bg-card p-4">
            <h2 className="text-sm font-semibold">Review actions</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Current status: <strong className="capitalize">{plan.data.status.replace('_', ' ')}</strong>
            </p>

            {!reviewer && (
              <div className="mt-3 flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/5 p-2 text-xs text-destructive">
                <ShieldAlert className="mt-0.5 h-4 w-4" aria-hidden />
                <span>
                  Set your reviewer identity in the sidebar before promoting.
                </span>
              </div>
            )}

            <label className="mt-3 block text-xs font-medium text-muted-foreground">
              Review notes (optional)
            </label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
              placeholder="Which rows did you verify? Any dosages to double-check before publishing?"
              className="mt-1 w-full resize-none rounded-md border bg-background px-2 py-1.5 text-sm"
            />

            <div className="mt-3 flex flex-col gap-1.5">
              {ACTIONS[plan.data.status].map((action) => (
                <button
                  key={action.to}
                  type="button"
                  disabled={mutation.isPending || !reviewer}
                  onClick={() => mutation.mutate(action.to)}
                  className={actionClass(action.tone)}
                >
                  {mutation.isPending && mutation.variables === action.to
                    ? 'Saving…'
                    : action.label}
                </button>
              ))}
            </div>

            {mutation.isError && (
              <p className="mt-2 text-xs text-destructive" role="alert">
                {mutation.error instanceof ApiError
                  ? mutation.error.problem.detail
                  : 'Status update failed'}
              </p>
            )}
            {mutation.isSuccess && (
              <p className="mt-2 text-xs text-primary">
                Updated to <strong className="capitalize">{mutation.data.status.replace('_', ' ')}</strong>.
              </p>
            )}
          </aside>
        </article>
      )}
    </div>
  );
}

function PlanCard({ plan }: { plan: CultivationPlanReview }) {
  const title = plan.title?.en ?? plan.slug;
  const summary = plan.summary?.en;
  return (
    <section className="rounded-xl border bg-card p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-xl font-semibold">{title}</h1>
          <p className="mt-1 text-xs text-muted-foreground">
            <code>{plan.slug}</code> · crop <strong>{plan.crop_slug}</strong> · season{' '}
            <span className="capitalize">{plan.season.replace('_', ' ')}</span>
            {plan.duration_weeks != null && <> · {plan.duration_weeks} weeks</>}
          </p>
        </div>
        <AuthorityChip authority={plan.authority} />
      </div>

      {plan.aez_codes && plan.aez_codes.length > 0 && (
        <p className="mt-3 text-xs text-muted-foreground">
          <strong className="text-foreground">AEZ:</strong>{' '}
          <code className="break-all">{plan.aez_codes.join(', ')}</code>
        </p>
      )}

      {summary && <p className="mt-3 whitespace-pre-line text-sm leading-relaxed">{summary}</p>}

      {plan.source_document_url && (
        <a
          href={plan.source_document_url}
          target="_blank"
          rel="noreferrer"
          className="mt-3 inline-flex items-center gap-1 text-xs text-primary hover:underline"
        >
          {plan.source_document_title ?? 'Source document'}
          <ExternalLink className="h-3 w-3" aria-hidden />
        </a>
      )}
    </section>
  );
}

function ActivitiesPanel({ activities }: { activities: CultivationActivity[] }) {
  if (activities.length === 0) return null;
  return (
    <section className="rounded-xl border bg-card p-5">
      <h2 className="text-sm font-semibold">Activities ({activities.length})</h2>
      <p className="mt-1 text-xs text-muted-foreground">
        Verify every dosage and DAP range against the source before promoting.
      </p>
      <div className="mt-3 overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="text-left text-[11px] uppercase tracking-wide text-muted-foreground">
            <tr>
              <th className="px-2 py-1.5">W</th>
              <th className="px-2 py-1.5">Activity</th>
              <th className="px-2 py-1.5">DAP</th>
              <th className="px-2 py-1.5">Title (en)</th>
              <th className="px-2 py-1.5">Inputs</th>
            </tr>
          </thead>
          <tbody>
            {activities.map((a, i) => (
              <tr key={i} className="border-t align-top">
                <td className="px-2 py-1.5 font-mono text-xs">{a.week_idx}</td>
                <td className="px-2 py-1.5 capitalize">{a.activity.replace(/_/g, ' ')}</td>
                <td className="px-2 py-1.5 font-mono text-xs">{formatDap(a.dap_min, a.dap_max)}</td>
                <td className="px-2 py-1.5 text-xs">{a.title?.en ?? '—'}</td>
                <td className="px-2 py-1.5">
                  <InputsList inputs={a.inputs} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function InputsList({ inputs }: { inputs?: CultivationActivityInput[] }) {
  if (!inputs || inputs.length === 0) return <span className="text-muted-foreground">—</span>;
  return (
    <ul className="flex flex-wrap gap-1">
      {inputs.map((i, idx) => (
        <li
          key={idx}
          className="inline-flex items-center gap-1 rounded border bg-background px-1.5 py-0.5 text-[10px]"
        >
          <span className="uppercase tracking-wide text-muted-foreground">{i.type}</span>
          {i.name?.en && <span>{i.name.en}</span>}
          {i.amount != null && (
            <span className="text-muted-foreground">
              · {i.amount}
              {i.unit ? ` ${i.unit}` : ''}
              {i.per_unit_area ? `/${i.per_unit_area.replace('_', ' ')}` : ''}
            </span>
          )}
        </li>
      ))}
    </ul>
  );
}

function PestRisksPanel({ risks }: { risks: CultivationPestRisk[] }) {
  if (risks.length === 0) return null;
  return (
    <section className="rounded-xl border bg-card p-5">
      <h2 className="text-sm font-semibold">Pest / disease risks ({risks.length})</h2>
      <div className="mt-3 overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="text-left text-[11px] uppercase tracking-wide text-muted-foreground">
            <tr>
              <th className="px-2 py-1.5">W</th>
              <th className="px-2 py-1.5">Organism</th>
              <th className="px-2 py-1.5">Risk</th>
              <th className="px-2 py-1.5">Notes (en)</th>
            </tr>
          </thead>
          <tbody>
            {risks.map((r, i) => (
              <tr key={i} className="border-t align-top">
                <td className="px-2 py-1.5 font-mono text-xs">{r.week_idx}</td>
                <td className="px-2 py-1.5 font-mono text-xs">
                  {r.disease_slug ?? r.pest_slug}
                </td>
                <td className="px-2 py-1.5">
                  <span
                    className={
                      'rounded-full border px-2 py-0.5 text-[10px] capitalize ' +
                      (r.risk === 'high'
                        ? 'bg-destructive/10 text-destructive border-destructive/40'
                        : r.risk === 'moderate'
                          ? 'bg-amber-500/10 text-amber-700 border-amber-500/30'
                          : 'bg-emerald-500/10 text-emerald-700 border-emerald-500/30')
                    }
                  >
                    {r.risk}
                  </span>
                </td>
                <td className="px-2 py-1.5 text-xs text-muted-foreground">{r.notes?.en ?? '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function EconomicsPanel({ economics }: { economics: CultivationEconomics }) {
  const money = (v?: number) =>
    v == null ? '—' : new Intl.NumberFormat('en-LK', { maximumFractionDigits: 0 }).format(v);
  return (
    <section className="rounded-xl border bg-card p-5">
      <h2 className="text-sm font-semibold">
        Economics · {economics.reference_year} · per {economics.unit_area}
      </h2>
      <dl className="mt-3 grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
        <Metric label="Gross revenue" value={money(economics.gross_revenue)} suffix={economics.currency} />
        <Metric label="Net (w/ family)" value={money(economics.net_revenue_with_family_labour)} suffix={economics.currency} />
        <Metric label="Net (hired only)" value={money(economics.net_revenue_without_family_labour)} suffix={economics.currency} />
      </dl>
      {economics.cost_lines && economics.cost_lines.length > 0 && (
        <ul className="mt-3 space-y-1 text-xs">
          {economics.cost_lines.map((line, i) => (
            <li key={i} className="flex justify-between border-t pt-1">
              <span className="capitalize text-muted-foreground">
                {line.category.replace('_', ' ')} — {line.label?.en ?? ''}
              </span>
              <span className="font-mono tabular-nums">{money(line.amount)}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function Metric({ label, value, suffix }: { label: string; value: string; suffix?: string }) {
  return (
    <div>
      <dt className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 text-base font-semibold tabular-nums">
        {value} {suffix && <span className="text-[11px] text-muted-foreground">{suffix}</span>}
      </dd>
    </div>
  );
}

function formatDap(min: number | undefined, max: number | undefined): string {
  if (min == null && max == null) return '—';
  const fmt = (n: number) => (n < 0 ? `${-n}d before` : `${n}d after`);
  if (min != null && max != null && min !== max) return `${fmt(min)} → ${fmt(max)}`;
  return fmt(min ?? max!);
}

function actionClass(tone: 'primary' | 'neutral' | 'destructive'): string {
  const base =
    'w-full rounded-md px-3 py-2 text-sm font-medium transition-colors disabled:opacity-50';
  switch (tone) {
    case 'primary':
      return `${base} bg-primary text-primary-foreground hover:bg-primary/90`;
    case 'destructive':
      return `${base} border border-destructive/40 text-destructive hover:bg-destructive/10`;
    default:
      return `${base} border bg-background text-foreground hover:bg-muted`;
  }
}
