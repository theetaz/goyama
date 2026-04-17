import { useState } from 'react';
import { Link, createFileRoute } from '@tanstack/react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, ChevronLeft, ExternalLink, ShieldAlert } from 'lucide-react';

import {
  ApiError,
  api,
  getReviewer,
  type FieldProvenance,
  type RecordStatus,
  type Remedy,
} from '@/lib/api';

export const Route = createFileRoute('/review-remedies/$slug')({
  component: RemedyReviewDetailPage,
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

function RemedyReviewDetailPage() {
  const { slug } = Route.useParams();
  const qc = useQueryClient();
  const [notes, setNotes] = useState('');

  const remedy = useQuery({
    queryKey: ['remedy-review', slug],
    queryFn: () => api.getRemedy(slug),
  });

  const mutation = useMutation({
    mutationFn: (to: RecordStatus) =>
      api.updateRemedyStatus(slug, {
        status: to,
        review_notes: notes.trim() || undefined,
      }),
    onSuccess: (updated) => {
      qc.setQueryData(['remedy-review', slug], updated);
      qc.invalidateQueries({ queryKey: ['remedy-review-queue'] });
      setNotes('');
    },
  });

  const reviewer = getReviewer();

  return (
    <div className="space-y-5">
      <Link
        to="/review-remedies"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ChevronLeft className="h-4 w-4" aria-hidden />
        Remedy review
      </Link>

      {remedy.isLoading && <p>Loading…</p>}
      {remedy.isError && (
        <p className="text-destructive" role="alert">
          {remedy.error instanceof Error ? remedy.error.message : 'Failed to load record'}
        </p>
      )}

      {remedy.data && (
        <article className="grid grid-cols-1 gap-4 lg:grid-cols-[1fr_340px]">
          <div className="space-y-4">
            <RemedyCard remedy={remedy.data} />
            {remedy.data.field_provenance && Object.keys(remedy.data.field_provenance).length > 0 && (
              <ProvenancePanel provenance={remedy.data.field_provenance} />
            )}
          </div>

          <aside className="h-fit rounded-xl border bg-card p-4">
            <h2 className="text-sm font-semibold">Review actions</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Current status: <strong className="capitalize">{remedy.data.status.replace('_', ' ')}</strong>
            </p>

            {remedy.data.type === 'chemical' && (
              <div className="mt-3 flex items-start gap-2 rounded-md border border-amber-500/40 bg-amber-500/5 p-2 text-xs text-amber-800">
                <AlertTriangle className="mt-0.5 h-4 w-4" aria-hidden />
                <span>
                  Chemical remedy — verify PHI, active-ingredient concentration, and
                  dosage against the DOA source quote before promoting.
                </span>
              </div>
            )}

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
              placeholder="Why this transition? What numeric claims did you verify?"
              className="mt-1 w-full resize-none rounded-md border bg-background px-2 py-1.5 text-sm"
            />

            <div className="mt-3 flex flex-col gap-1.5">
              {ACTIONS[remedy.data.status].map((action) => (
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

            {remedy.data.reviewed_by && (
              <div className="mt-4 border-t pt-3 text-xs text-muted-foreground">
                <div>
                  Last change by <strong>{remedy.data.reviewed_by}</strong>
                </div>
                {remedy.data.reviewed_at && (
                  <div>at {new Date(remedy.data.reviewed_at).toLocaleString()}</div>
                )}
                {remedy.data.review_notes && (
                  <p className="mt-1 whitespace-pre-line text-foreground">
                    “{remedy.data.review_notes}”
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

function RemedyCard({ remedy }: { remedy: Remedy }) {
  const name = remedy.name?.en ?? remedy.slug;
  return (
    <section className="rounded-xl border bg-card p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">{name}</h1>
          <div className="mt-1 text-xs text-muted-foreground">
            <code>{remedy.slug}</code>
          </div>
        </div>
        <div className="flex flex-wrap gap-1.5 text-[11px]">
          <span className="rounded-full bg-muted px-2 py-0.5 capitalize text-muted-foreground">
            {remedy.type.replace('_', ' ')}
          </span>
          {remedy.effectiveness && (
            <span className="rounded-full bg-muted px-2 py-0.5 capitalize text-muted-foreground">
              effectiveness · {remedy.effectiveness}
            </span>
          )}
          {remedy.cost_tier && (
            <span className="rounded-full bg-muted px-2 py-0.5 capitalize text-muted-foreground">
              cost · {remedy.cost_tier}
            </span>
          )}
          {remedy.organic_compatible === true && (
            <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-emerald-700">
              organic-compatible
            </span>
          )}
        </div>
      </div>

      {remedy.description?.en && (
        <div className="mt-3 rounded-lg border bg-background p-3 text-sm leading-relaxed">
          {remedy.description.en}
        </div>
      )}

      {remedy.type === 'chemical' && (
        <div className="mt-3 rounded-lg border border-destructive/30 bg-destructive/5 p-3">
          <div className="text-xs font-semibold uppercase tracking-wide text-destructive">
            Chemical profile
          </div>
          <dl className="mt-2 grid grid-cols-1 gap-y-1 text-sm md:grid-cols-2">
            <ProfileRow label="Active ingredient" value={remedy.active_ingredient} />
            <ProfileRow label="Concentration" value={remedy.concentration} />
            <ProfileRow label="Formulation" value={remedy.formulation} />
            <ProfileRow label="DOA registration" value={remedy.doa_registration_no} />
            <ProfileRow
              label="Pre-harvest interval"
              value={remedy.pre_harvest_interval_days != null ? `${remedy.pre_harvest_interval_days} days` : undefined}
            />
            <ProfileRow
              label="Re-entry interval"
              value={remedy.re_entry_interval_hours != null ? `${remedy.re_entry_interval_hours} hours` : undefined}
            />
            <ProfileRow label="WHO hazard class" value={remedy.who_hazard_class} />
            <ProfileRow label="Application method" value={remedy.application_method?.replace(/_/g, ' ')} />
          </dl>
          {remedy.dosage && (
            <p className="mt-2 text-sm">
              <span className="text-xs font-semibold uppercase tracking-wide text-destructive">
                Dosage
              </span>{' '}
              {remedy.dosage}
            </p>
          )}
          {remedy.safety_notes?.en && (
            <p className="mt-2 rounded-md border border-destructive/30 bg-background p-2 text-xs leading-relaxed text-destructive">
              <strong>Safety:</strong> {remedy.safety_notes.en}
            </p>
          )}
        </div>
      )}

      {remedy.instructions?.en && (
        <div className="mt-3 rounded-lg border bg-background p-3 text-sm">
          <div className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Instructions
          </div>
          <p className="mt-1 text-foreground">{remedy.instructions.en}</p>
        </div>
      )}

      <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-3">
        <Facts label="Targets diseases" items={remedy.target_disease_slugs} />
        <Facts label="Targets pests" items={remedy.target_pest_slugs} />
        <Facts label="Applicable crops" items={remedy.applicable_crop_slugs} />
      </div>
    </section>
  );
}

function ProfileRow({ label, value }: { label: string; value?: string }) {
  if (!value) return null;
  return (
    <div className="flex gap-2">
      <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </dt>
      <dd>{value}</dd>
    </div>
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
        Chemical dosages, PHI, and hazard-class claims must each trace to a Sri
        Lanka-authoritative source before promotion. Verify every quote below.
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
