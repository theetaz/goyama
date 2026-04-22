import { useState } from 'react';
import { Link, createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { CalendarRange } from 'lucide-react';

import { api, type CultivationPlanReview, type RecordStatus } from '@/lib/api';
import { AuthorityChip } from '@/components/authority-chip';

export const Route = createFileRoute('/review-plans')({
  component: PlanReviewQueuePage,
});

const STATUSES: RecordStatus[] = ['draft', 'in_review', 'published', 'deprecated', 'rejected'];

function PlanReviewQueuePage() {
  const [status, setStatus] = useState<RecordStatus>('draft');
  const queue = useQuery({
    queryKey: ['plan-review-queue', status],
    queryFn: () => api.listCultivationPlansForReview(status),
  });

  return (
    <div className="space-y-5">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-semibold">
            <CalendarRange className="h-6 w-6 text-primary" aria-hidden />
            Cultivation plan review
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Plans at <code>{status}</code>. Promoting a plan to <code>published</code>
            makes it visible on the farmer crop-detail page.
          </p>
        </div>
        <div className="flex flex-wrap rounded-md border bg-card p-0.5 text-sm">
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
        <p role="alert" className="text-destructive">
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
          {queue.data.items.map((p) => (
            <QueueRow key={p.slug} plan={p} />
          ))}
        </ul>
      )}
    </div>
  );
}

function QueueRow({ plan }: { plan: CultivationPlanReview }) {
  const title = plan.title?.en ?? plan.slug;
  const aez = plan.aez_codes?.slice(0, 4).join(', ') ?? '';
  const aezExtra = plan.aez_codes && plan.aez_codes.length > 4 ? ` +${plan.aez_codes.length - 4}` : '';
  return (
    <li>
      <Link
        to="/review-plans/$slug"
        params={{ slug: plan.slug }}
        className="flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-card p-4 hover:border-primary"
      >
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium">{title}</span>
            <AuthorityChip authority={plan.authority} />
            <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] capitalize text-muted-foreground">
              {plan.season.replace('_', ' ')}
            </span>
            {plan.duration_weeks != null && (
              <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] text-muted-foreground">
                {plan.duration_weeks} weeks
              </span>
            )}
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            <strong>{plan.crop_slug}</strong> · {plan.activities.length} activities · {plan.pest_risks.length} pest-risk rows
            {aez && ` · AEZ: ${aez}${aezExtra}`}
          </div>
        </div>
        <div className="text-right text-xs">
          <div className="font-medium capitalize">{plan.status.replace('_', ' ')}</div>
          {plan.reviewed_by && (
            <div className="text-muted-foreground">by {plan.reviewed_by}</div>
          )}
        </div>
      </Link>
    </li>
  );
}
