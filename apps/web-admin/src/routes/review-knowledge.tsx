import { useState } from 'react';
import { Link, createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { Sparkles } from 'lucide-react';

import { api, type KnowledgeChunkReview, type RecordStatus } from '@/lib/api';
import { AuthorityChip } from '@/components/authority-chip';

export const Route = createFileRoute('/review-knowledge')({
  component: KnowledgeReviewQueuePage,
});

const STATUSES: RecordStatus[] = ['draft', 'in_review', 'published', 'deprecated', 'rejected'];

function KnowledgeReviewQueuePage() {
  const [status, setStatus] = useState<RecordStatus>('draft');
  const queue = useQuery({
    queryKey: ['knowledge-review-queue', status],
    queryFn: () => api.listKnowledgeChunksForReview(status),
  });

  return (
    <div className="space-y-5">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-semibold">
            <Sparkles className="h-6 w-6 text-primary" aria-hidden />
            Knowledge chunk review
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Unstructured agronomic knowledge at <code>{status}</code>. Chunks from
            cross-regional sources (TNAU, ICAR, FAO) require careful verification
            before publishing.
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
          {queue.data.items.map((c) => (
            <QueueRow key={c.slug} chunk={c} />
          ))}
        </ul>
      )}
    </div>
  );
}

function QueueRow({ chunk }: { chunk: KnowledgeChunkReview }) {
  const preview = chunk.body.slice(0, 200);
  const entities =
    chunk.entity_refs
      ?.slice(0, 5)
      .map((r) => `${r.type}:${r.slug}`)
      .join(' · ') ?? '';
  return (
    <li>
      <Link
        to="/review-knowledge/$slug"
        params={{ slug: chunk.slug }}
        className="block rounded-lg border bg-card p-4 hover:border-primary"
      >
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-2">
            <span className="font-medium">{chunk.title ?? chunk.slug}</span>
            <AuthorityChip authority={chunk.authority} />
            {chunk.confidence != null && (
              <span className="rounded-full border bg-background px-2 py-0.5 text-[10px] uppercase tracking-wide text-muted-foreground">
                {Math.round(chunk.confidence * 100)}%
              </span>
            )}
          </div>
          <div className="text-right text-xs">
            <div className="font-medium capitalize">{chunk.status.replace('_', ' ')}</div>
            {chunk.reviewed_by && (
              <div className="text-muted-foreground">by {chunk.reviewed_by}</div>
            )}
          </div>
        </div>
        <p className="mt-2 line-clamp-2 text-xs text-muted-foreground">
          {preview}
          {chunk.body.length > preview.length && '…'}
        </p>
        {entities && (
          <p className="mt-1 text-[11px] text-muted-foreground">→ {entities}</p>
        )}
      </Link>
    </li>
  );
}
