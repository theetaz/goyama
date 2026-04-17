import { useState } from 'react';
import { Link, createFileRoute } from '@tanstack/react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft, ExternalLink, ShieldAlert } from 'lucide-react';

import {
  ApiError,
  api,
  getReviewer,
  type CultivationStep,
  type FieldProvenance,
  type RecordStatus,
} from '@/lib/api';

export const Route = createFileRoute('/review/$slug')({
  component: ReviewDetailPage,
});

// Transitions the UI offers per current status. The backend enforces the
// same set — this just gates which buttons we draw.
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
  published: [
    { to: 'deprecated', label: 'Deprecate', tone: 'destructive' },
  ],
  deprecated: [
    { to: 'published', label: 'Restore to published', tone: 'primary' },
  ],
  rejected: [
    { to: 'draft', label: 'Move back to draft', tone: 'neutral' },
  ],
};

function ReviewDetailPage() {
  const { slug } = Route.useParams();
  const qc = useQueryClient();
  const [notes, setNotes] = useState('');

  const step = useQuery({
    queryKey: ['review-step', slug],
    queryFn: () => api.getCultivationStep(slug),
  });

  const mutation = useMutation({
    mutationFn: (to: RecordStatus) =>
      api.updateCultivationStepStatus(slug, {
        status: to,
        review_notes: notes.trim() || undefined,
      }),
    onSuccess: (updated) => {
      qc.setQueryData(['review-step', slug], updated);
      // Invalidate every queue-bucket so counts refresh in the list view.
      qc.invalidateQueries({ queryKey: ['review-queue'] });
      setNotes('');
    },
  });

  const reviewer = getReviewer();

  return (
    <div className="space-y-5">
      <Link
        to="/review"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ChevronLeft className="h-4 w-4" aria-hidden />
        Review queue
      </Link>

      {step.isLoading && <p>Loading…</p>}
      {step.isError && (
        <p className="text-destructive" role="alert">
          {step.error instanceof Error ? step.error.message : 'Failed to load record'}
        </p>
      )}

      {step.data && (
        <article className="grid grid-cols-1 gap-4 lg:grid-cols-[1fr_340px]">
          <div className="space-y-4">
            <StepCard step={step.data} />
            {step.data.field_provenance && (
              <ProvenancePanel provenance={step.data.field_provenance} />
            )}
          </div>

          <aside className="h-fit rounded-xl border bg-card p-4">
            <h2 className="text-sm font-semibold">Review actions</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Current status: <strong className="capitalize">{step.data.status.replace('_', ' ')}</strong>
            </p>

            {!reviewer && (
              <div className="mt-3 flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/5 p-2 text-xs text-destructive">
                <ShieldAlert className="mt-0.5 h-4 w-4" aria-hidden />
                <span>
                  Set your reviewer identity in the sidebar before promoting — the
                  backend rejects unidentified writes.
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
              placeholder="Why this transition? What did you verify?"
              className="mt-1 w-full resize-none rounded-md border bg-background px-2 py-1.5 text-sm"
            />

            <div className="mt-3 flex flex-col gap-1.5">
              {ACTIONS[step.data.status].map((action) => (
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

            {step.data.reviewed_by && (
              <div className="mt-4 border-t pt-3 text-xs text-muted-foreground">
                <div>
                  Last change by <strong>{step.data.reviewed_by}</strong>
                </div>
                {step.data.reviewed_at && (
                  <div>at {new Date(step.data.reviewed_at).toLocaleString()}</div>
                )}
                {step.data.review_notes && (
                  <p className="mt-1 whitespace-pre-line text-foreground">
                    “{step.data.review_notes}”
                  </p>
                )}
              </div>
            )}
          </aside>
        </article>
      )}
    </div>
  );
}

function StepCard({ step }: { step: CultivationStep }) {
  const dap = formatDap(step.day_after_planting);
  return (
    <section className="rounded-xl border bg-card p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">{step.title?.en ?? step.slug}</h1>
          <div className="mt-1 text-xs text-muted-foreground">
            <code>{step.slug}</code> · crop <code>{step.crop_slug}</code> · step {step.order_idx}
          </div>
        </div>
        <div className="flex flex-wrap gap-1.5 text-[11px]">
          <span className="rounded-full bg-muted px-2 py-0.5 capitalize text-muted-foreground">
            {step.stage.replace(/_/g, ' ')}
          </span>
          {step.season && (
            <span className="rounded-full bg-muted px-2 py-0.5 capitalize text-muted-foreground">
              {step.season.replace('_', ' ')}
            </span>
          )}
          {dap && <span className="rounded-full bg-primary/10 px-2 py-0.5 text-primary">{dap}</span>}
        </div>
      </div>

      <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-3">
        {(['en', 'si', 'ta'] as const).map((locale) => (
          <div key={locale} className="rounded-lg border bg-background p-3">
            <div className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">
              {locale}
            </div>
            <div className="mt-1 font-medium">{step.title?.[locale] ?? '—'}</div>
            {step.body?.[locale] && (
              <p className="mt-1 whitespace-pre-line text-xs text-muted-foreground">
                {step.body[locale]}
              </p>
            )}
          </div>
        ))}
      </div>

      {step.inputs && step.inputs.length > 0 && (
        <div className="mt-4">
          <h2 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Inputs
          </h2>
          <ul className="mt-2 space-y-1 text-sm">
            {step.inputs.map((input, i) => (
              <li key={i} className="rounded-md border bg-background px-2 py-1">
                <span className="mr-2 text-[10px] uppercase tracking-wide text-muted-foreground">
                  {input.type}
                </span>
                <span>{input.name?.en}</span>
                {input.amount != null && (
                  <span className="ml-2 text-muted-foreground">
                    {input.amount} {input.unit}
                    {input.per_unit_area ? `/${input.per_unit_area.replace('_', ' ')}` : ''}
                  </span>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  );
}

function ProvenancePanel({
  provenance,
}: {
  provenance: Record<string, FieldProvenance>;
}) {
  const fields = Object.keys(provenance).sort();
  if (fields.length === 0) return null;
  return (
    <section className="rounded-xl border bg-card p-5">
      <h2 className="text-sm font-semibold">Provenance</h2>
      <p className="mt-1 text-xs text-muted-foreground">
        Every numeric or factual claim must trace back to a Sri Lanka-authoritative
        source before this record is promoted. Verify each quote below.
      </p>
      <dl className="mt-3 space-y-3">
        {fields.map((field) => {
          const p = provenance[field];
          return (
            <div key={field} className="rounded-md border bg-background p-3 text-sm">
              <dt className="flex items-center justify-between gap-2">
                <code>{field}</code>
                <a
                  href={p.source_url}
                  target="_blank"
                  rel="noreferrer"
                  className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
                >
                  {p.source_id}
                  <ExternalLink className="h-3 w-3" aria-hidden />
                </a>
              </dt>
              {p.quote && (
                <dd className="mt-2 border-l-2 border-muted pl-3 italic text-muted-foreground">
                  “{p.quote}”
                </dd>
              )}
              <dd className="mt-1 text-[11px] text-muted-foreground">
                fetched {new Date(p.fetched_at).toLocaleDateString()}
                {p.confidence != null && ` · confidence ${p.confidence}`}
              </dd>
            </div>
          );
        })}
      </dl>
    </section>
  );
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

function formatDap(range: { min?: number; max?: number } | undefined): string | null {
  if (!range) return null;
  const { min, max } = range;
  if (min == null && max == null) return null;
  const fmt = (n: number) => (n < 0 ? `${-n} d before planting` : `day ${n}`);
  if (min != null && max != null && min !== max) return `${fmt(min)} → ${fmt(max)}`;
  return fmt(min ?? max!);
}
