import { useState } from 'react';
import { Link, createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { Bug } from 'lucide-react';

import { api, type Disease, type RecordStatus } from '@/lib/api';

export const Route = createFileRoute('/review-diseases')({
  component: DiseaseReviewQueuePage,
});

const STATUSES: RecordStatus[] = ['draft', 'in_review', 'published', 'rejected'];

function DiseaseReviewQueuePage() {
  const [status, setStatus] = useState<RecordStatus>('draft');
  const queue = useQuery({
    queryKey: ['disease-review-queue', status],
    queryFn: () => api.listDiseasesForReview(status),
  });

  return (
    <div className="space-y-5">
      <header className="flex items-end justify-between gap-4">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-semibold">
            <Bug className="h-6 w-6 text-primary" aria-hidden />
            Disease review
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Disease records currently at <code>{status}</code>. Promoting to{' '}
            <code>published</code> surfaces them in the scanner / advisory surfaces.
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
          {queue.data.items.map((d) => (
            <QueueRow key={d.slug} disease={d} />
          ))}
        </ul>
      )}
    </div>
  );
}

function QueueRow({ disease }: { disease: Disease }) {
  const name = disease.names?.en ?? disease.slug;
  const affects = (disease.affected_crop_slugs ?? []).slice(0, 3).join(', ');
  return (
    <li>
      <Link
        to="/review-diseases/$slug"
        params={{ slug: disease.slug }}
        className="flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-card p-4 hover:border-primary"
      >
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium">{name}</span>
            <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] capitalize text-muted-foreground">
              {disease.causal_organism}
            </span>
            {disease.severity && (
              <span
                className={
                  'rounded-full px-2 py-0.5 text-[11px] capitalize ' +
                  (disease.severity === 'high'
                    ? 'bg-destructive/10 text-destructive'
                    : 'bg-muted text-muted-foreground')
                }
              >
                {disease.severity}
              </span>
            )}
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            {disease.scientific_name && <em>{disease.scientific_name}</em>}
            {affects && ` · affects: ${affects}${(disease.affected_crop_slugs?.length ?? 0) > 3 ? '…' : ''}`}
          </div>
        </div>
        <div className="text-right text-xs">
          <div className="font-medium capitalize">{disease.status.replace('_', ' ')}</div>
          {disease.reviewed_by && (
            <div className="text-muted-foreground">by {disease.reviewed_by}</div>
          )}
        </div>
      </Link>
    </li>
  );
}
