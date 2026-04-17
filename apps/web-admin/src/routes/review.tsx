import { useState } from 'react';
import { Link, createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { ClipboardList } from 'lucide-react';

import {
  api,
  type CultivationStep,
  type RecordStatus,
} from '@/lib/api';

export const Route = createFileRoute('/review')({
  component: ReviewQueuePage,
});

const STATUSES: RecordStatus[] = ['draft', 'in_review', 'published', 'rejected'];

function ReviewQueuePage() {
  const [status, setStatus] = useState<RecordStatus>('draft');
  const queue = useQuery({
    queryKey: ['review-queue', status],
    queryFn: () => api.listCultivationStepsForReview(status),
  });

  return (
    <div className="space-y-5">
      <header className="flex items-end justify-between gap-4">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-semibold">
            <ClipboardList className="h-6 w-6 text-primary" aria-hidden />
            Review queue
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Cultivation steps currently at <code>{status}</code>. Promoting to{' '}
            <code>published</code> surfaces them in the farmer-facing app.
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
          {queue.data.items.map((step) => (
            <QueueRow key={step.slug} step={step} />
          ))}
        </ul>
      )}
    </div>
  );
}

function QueueRow({ step }: { step: CultivationStep }) {
  return (
    <li>
      <Link
        to="/review/$slug"
        params={{ slug: step.slug }}
        className="flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-card p-4 hover:border-primary"
      >
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium">{step.title?.en ?? step.slug}</span>
            <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] capitalize text-muted-foreground">
              {step.stage.replace(/_/g, ' ')}
            </span>
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            crop: <code>{step.crop_slug}</code> · step {step.order_idx} · {step.slug}
          </div>
        </div>
        <div className="text-right text-xs">
          <div className="font-medium capitalize">{step.status.replace('_', ' ')}</div>
          {step.reviewed_by && (
            <div className="text-muted-foreground">by {step.reviewed_by}</div>
          )}
        </div>
      </Link>
    </li>
  );
}
