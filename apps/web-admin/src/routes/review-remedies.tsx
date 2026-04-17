import { useState } from 'react';
import { Link, createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { FlaskConical } from 'lucide-react';

import { api, type RecordStatus, type Remedy, type RemedyType } from '@/lib/api';

export const Route = createFileRoute('/review-remedies')({
  component: RemedyReviewQueuePage,
});

const STATUSES: RecordStatus[] = ['draft', 'in_review', 'published', 'rejected'];

// Colour the type chip so a reviewer can spot the chemical remedies in a
// queue at a glance — they carry the tightest review gate.
const TYPE_TONE: Record<RemedyType, string> = {
  chemical: 'bg-destructive/10 text-destructive',
  biological: 'bg-emerald-500/10 text-emerald-700',
  cultural: 'bg-sky-500/10 text-sky-700',
  resistant_variety: 'bg-amber-500/10 text-amber-700',
  mechanical: 'bg-muted text-muted-foreground',
  integrated: 'bg-violet-500/10 text-violet-700',
};

function RemedyReviewQueuePage() {
  const [status, setStatus] = useState<RecordStatus>('draft');
  const queue = useQuery({
    queryKey: ['remedy-review-queue', status],
    queryFn: () => api.listRemediesForReview(status),
  });

  return (
    <div className="space-y-5">
      <header className="flex items-end justify-between gap-4">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-semibold">
            <FlaskConical className="h-6 w-6 text-primary" aria-hidden />
            Remedy review
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Remedies currently at <code>{status}</code>. Chemical dosages, PHI, and
            WHO hazard class fall under CLAUDE.md's hardest review gate — verify
            each numeric claim against its source quote before promoting.
          </p>
        </div>
        <div className="flex rounded-md border bg-card p-0.5 text-sm">
          {STATUSES.map((s) => (
            <button
              key={s}
              type="button"
              onClick={() => setStatus(s)}
              className={
                'rounded px-3 py-1.5 capitalize transition-colors ' +
                (status === s
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:bg-muted')
              }
            >
              {s.replace('_', ' ')}
            </button>
          ))}
        </div>
      </header>

      {queue.isLoading && <p>Loading…</p>}
      {queue.isError && (
        <p className="text-destructive" role="alert">
          {queue.error instanceof Error ? queue.error.message : 'Failed to load queue'}
        </p>
      )}
      {queue.data && queue.data.items.length === 0 && (
        <p className="rounded-lg border bg-card p-6 text-center text-sm text-muted-foreground">
          Nothing to review at <code>{status}</code>.
        </p>
      )}
      {queue.data && queue.data.items.length > 0 && (
        <ul className="space-y-2">
          {queue.data.items.map((r) => (
            <QueueRow key={r.slug} remedy={r} />
          ))}
        </ul>
      )}
    </div>
  );
}

function QueueRow({ remedy }: { remedy: Remedy }) {
  const name = remedy.name?.en ?? remedy.slug;
  const targets = [
    ...(remedy.target_disease_slugs ?? []),
    ...(remedy.target_pest_slugs ?? []),
  ]
    .slice(0, 3)
    .join(', ');
  return (
    <li>
      <Link
        to="/review-remedies/$slug"
        params={{ slug: remedy.slug }}
        className="flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-card p-4 hover:border-primary"
      >
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium">{name}</span>
            <span
              className={
                'rounded-full px-2 py-0.5 text-[11px] capitalize ' +
                TYPE_TONE[remedy.type]
              }
            >
              {remedy.type.replace('_', ' ')}
            </span>
            {remedy.pre_harvest_interval_days != null && (
              <span className="rounded-full bg-primary/10 px-2 py-0.5 text-[11px] text-primary">
                PHI {remedy.pre_harvest_interval_days}d
              </span>
            )}
            {remedy.organic_compatible === true && (
              <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] text-emerald-700">
                organic
              </span>
            )}
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            {remedy.active_ingredient && <>AI: {remedy.active_ingredient} · </>}
            {targets && `targets: ${targets}`}
          </div>
        </div>
        <div className="text-right text-xs">
          <div className="font-medium capitalize">{remedy.status.replace('_', ' ')}</div>
          {remedy.reviewed_by && (
            <div className="text-muted-foreground">by {remedy.reviewed_by}</div>
          )}
        </div>
      </Link>
    </li>
  );
}
