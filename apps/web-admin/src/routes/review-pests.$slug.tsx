import { useState } from 'react';
import { Link, createFileRoute } from '@tanstack/react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft, ExternalLink, ShieldAlert } from 'lucide-react';

import {
  ApiError,
  api,
  getReviewer,
  type FieldProvenance,
  type Pest,
  type RecordStatus,
} from '@/lib/api';

export const Route = createFileRoute('/review-pests/$slug')({
  component: PestReviewDetailPage,
});

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

function PestReviewDetailPage() {
  const { slug } = Route.useParams();
  const qc = useQueryClient();
  const [notes, setNotes] = useState('');

  const pest = useQuery({
    queryKey: ['pest-review', slug],
    queryFn: () => api.getPest(slug),
  });

  const mutation = useMutation({
    mutationFn: (to: RecordStatus) =>
      api.updatePestStatus(slug, {
        status: to,
        review_notes: notes.trim() || undefined,
      }),
    onSuccess: (updated) => {
      qc.setQueryData(['pest-review', slug], updated);
      qc.invalidateQueries({ queryKey: ['pest-review-queue'] });
      setNotes('');
    },
  });

  const reviewer = getReviewer();

  return (
    <div className="space-y-5">
      <Link
        to="/review-pests"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ChevronLeft className="h-4 w-4" aria-hidden />
        Pest review
      </Link>

      {pest.isLoading && <p>Loading…</p>}
      {pest.isError && (
        <p className="text-destructive" role="alert">
          {pest.error instanceof Error ? pest.error.message : 'Failed to load record'}
        </p>
      )}

      {pest.data && (
        <article className="grid grid-cols-1 gap-4 lg:grid-cols-[1fr_340px]">
          <div className="space-y-4">
            <PestCard pest={pest.data} />
            {pest.data.field_provenance && Object.keys(pest.data.field_provenance).length > 0 && (
              <ProvenancePanel provenance={pest.data.field_provenance} />
            )}
          </div>

          <aside className="h-fit rounded-xl border bg-card p-4">
            <h2 className="text-sm font-semibold">Review actions</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Current status: <strong className="capitalize">{pest.data.status.replace('_', ' ')}</strong>
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
              {ACTIONS[pest.data.status].map((action) => (
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

            {pest.data.reviewed_by && (
              <div className="mt-4 border-t pt-3 text-xs text-muted-foreground">
                <div>
                  Last change by <strong>{pest.data.reviewed_by}</strong>
                </div>
                {pest.data.reviewed_at && (
                  <div>at {new Date(pest.data.reviewed_at).toLocaleString()}</div>
                )}
                {pest.data.review_notes && (
                  <p className="mt-1 whitespace-pre-line text-foreground">
                    “{pest.data.review_notes}”
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

function PestCard({ pest }: { pest: Pest }) {
  const name = pest.names?.en ?? pest.slug;
  return (
    <section className="rounded-xl border bg-card p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">{name}</h1>
          {pest.scientific_name && (
            <p className="mt-1 text-sm italic text-muted-foreground">{pest.scientific_name}</p>
          )}
          <div className="mt-1 text-xs text-muted-foreground">
            <code>{pest.slug}</code>
          </div>
        </div>
        <div className="flex flex-wrap gap-1.5 text-[11px]">
          <span className="rounded-full bg-muted px-2 py-0.5 capitalize text-muted-foreground">
            {pest.kingdom}
          </span>
        </div>
      </div>

      {pest.description?.en && (
        <div className="mt-3 rounded-lg border bg-background p-3 text-sm leading-relaxed">
          {pest.description.en}
        </div>
      )}

      {pest.economic_threshold?.en && (
        <div className="mt-3 rounded-lg border border-primary/30 bg-primary/5 p-3 text-sm">
          <div className="text-xs font-semibold uppercase tracking-wide text-primary">
            Economic threshold / monitoring
          </div>
          <p className="mt-1 text-foreground">{pest.economic_threshold.en}</p>
        </div>
      )}

      <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-3">
        <Facts label="Affected crops" items={pest.affected_crop_slugs} />
        <Facts label="Life stages" items={pest.life_stages} />
        <Facts label="Feeding type" items={pest.feeding_type} />
      </div>

      {pest.aliases && pest.aliases.length > 0 && (
        <div className="mt-3">
          <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Aliases
          </span>
          <ul className="mt-1 flex flex-wrap gap-1.5">
            {pest.aliases.map((a) => (
              <li
                key={a}
                className="rounded-md border bg-background px-2 py-0.5 text-[11px] text-muted-foreground"
              >
                {a}
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  );
}

function Facts({ label, items }: { label: string; items?: string[] }) {
  if (!items || items.length === 0) return null;
  return (
    <div className="rounded-lg border bg-background p-3 text-xs">
      <div className="font-semibold uppercase tracking-wide text-muted-foreground">{label}</div>
      <ul className="mt-1.5 space-y-0.5">
        {items.map((v) => (
          <li key={v} className="capitalize text-foreground">
            {v.replace(/_/g, ' ')}
          </li>
        ))}
      </ul>
    </div>
  );
}

function ProvenancePanel({
  provenance,
}: {
  provenance: Record<string, FieldProvenance>;
}) {
  const fields = Object.keys(provenance).sort();
  return (
    <section className="rounded-xl border bg-card p-5">
      <h2 className="text-sm font-semibold">Provenance</h2>
      <p className="mt-1 text-xs text-muted-foreground">
        Every claim — especially pesticide recommendations — must trace back to an
        authoritative source before this pest record is promoted.
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
