import { useState } from 'react';
import { Link, createFileRoute } from '@tanstack/react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft, ShieldAlert } from 'lucide-react';

import {
  ApiError,
  api,
  getReviewer,
  type KnowledgeChunkReview,
  type RecordStatus,
} from '@/lib/api';
import { AuthorityChip } from '@/components/authority-chip';

export const Route = createFileRoute('/review-knowledge/$slug')({
  component: KnowledgeReviewDetailPage,
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

function KnowledgeReviewDetailPage() {
  const { slug } = Route.useParams();
  const qc = useQueryClient();
  const [notes, setNotes] = useState('');

  const chunk = useQuery({
    queryKey: ['knowledge-review', slug],
    queryFn: () => api.getKnowledgeChunkForReview(slug),
  });

  const mutation = useMutation({
    mutationFn: (to: RecordStatus) =>
      api.updateKnowledgeChunkStatus(slug, {
        status: to,
        review_notes: notes.trim() || undefined,
      }),
    onSuccess: (updated) => {
      qc.setQueryData(['knowledge-review', slug], updated);
      qc.invalidateQueries({ queryKey: ['knowledge-review-queue'] });
      setNotes('');
    },
  });

  const reviewer = getReviewer();

  return (
    <div className="space-y-5">
      <Link
        to="/review-knowledge"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ChevronLeft className="h-4 w-4" aria-hidden />
        Knowledge review
      </Link>

      {chunk.isLoading && <p>Loading…</p>}
      {chunk.isError && (
        <p role="alert" className="text-destructive">
          {chunk.error instanceof Error ? chunk.error.message : 'Failed to load chunk'}
        </p>
      )}

      {chunk.data && (
        <article className="grid grid-cols-1 gap-4 lg:grid-cols-[1fr_340px]">
          <div className="space-y-4">
            <ChunkCard chunk={chunk.data} />
          </div>

          <aside className="h-fit rounded-xl border bg-card p-4">
            <h2 className="text-sm font-semibold">Review actions</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Current status: <strong className="capitalize">{chunk.data.status.replace('_', ' ')}</strong>
            </p>

            {chunk.data.authority !== 'doa_official' && chunk.data.authority !== 'peer_reviewed' && (
              <div className="mt-3 flex items-start gap-2 rounded-md border border-amber-500/40 bg-amber-500/5 p-2 text-xs text-amber-800 dark:text-amber-300">
                <ShieldAlert className="mt-0.5 h-4 w-4" aria-hidden />
                <span>
                  Lower-authority chunk. Verify the quote against the source before
                  publishing and consider keeping at <code>in_review</code> until a
                  Sri Lanka agronomist validates the claim locally.
                </span>
              </div>
            )}

            {!reviewer && (
              <div className="mt-3 flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/5 p-2 text-xs text-destructive">
                <ShieldAlert className="mt-0.5 h-4 w-4" aria-hidden />
                <span>Set your reviewer identity in the sidebar before promoting.</span>
              </div>
            )}

            <label className="mt-3 block text-xs font-medium text-muted-foreground">
              Review notes (optional)
            </label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
              placeholder="Did the quote match the source? Any caveats before farmer-facing display?"
              className="mt-1 w-full resize-none rounded-md border bg-background px-2 py-1.5 text-sm"
            />

            <div className="mt-3 flex flex-col gap-1.5">
              {ACTIONS[chunk.data.status].map((action) => (
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

function ChunkCard({ chunk }: { chunk: KnowledgeChunkReview }) {
  return (
    <section className="rounded-xl border bg-card p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-xl font-semibold">{chunk.title ?? chunk.slug}</h1>
          <p className="mt-1 text-xs text-muted-foreground">
            <code>{chunk.slug}</code> · source <code>{chunk.source_slug}</code> ·{' '}
            lang <strong>{chunk.language}</strong>
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {chunk.confidence != null && (
            <span className="rounded-full border bg-background px-2 py-0.5 text-[10px] uppercase tracking-wide text-muted-foreground">
              confidence {Math.round(chunk.confidence * 100)}%
            </span>
          )}
          <AuthorityChip authority={chunk.authority} />
        </div>
      </div>

      <p className="mt-3 whitespace-pre-line text-sm leading-relaxed">{chunk.body}</p>

      {chunk.quote && (
        <blockquote className="mt-3 border-l-2 border-muted pl-3 text-xs italic text-muted-foreground">
          "{chunk.quote}"
        </blockquote>
      )}

      <div className="mt-4 grid grid-cols-1 gap-3 text-xs sm:grid-cols-2">
        {chunk.topic_tags && chunk.topic_tags.length > 0 && (
          <div>
            <div className="font-semibold uppercase tracking-wide text-muted-foreground">Topic tags</div>
            <div className="mt-1 flex flex-wrap gap-1">
              {chunk.topic_tags.map((tag) => (
                <span key={tag} className="rounded bg-muted px-1.5 py-0.5">
                  #{tag}
                </span>
              ))}
            </div>
          </div>
        )}
        {chunk.entity_refs && chunk.entity_refs.length > 0 && (
          <div>
            <div className="font-semibold uppercase tracking-wide text-muted-foreground">Linked entities</div>
            <ul className="mt-1 space-y-0.5">
              {chunk.entity_refs.map((r, i) => (
                <li key={i}>
                  <span className="rounded bg-muted px-1.5 py-0.5">
                    {r.type}:{r.slug}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}
        {chunk.applies_to_countries && chunk.applies_to_countries.length > 0 && (
          <div>
            <div className="font-semibold uppercase tracking-wide text-muted-foreground">Countries</div>
            <div className="mt-1">{chunk.applies_to_countries.join(', ')}</div>
          </div>
        )}
        {chunk.applies_to_aez_codes && chunk.applies_to_aez_codes.length > 0 && (
          <div>
            <div className="font-semibold uppercase tracking-wide text-muted-foreground">AEZ codes</div>
            <div className="mt-1 font-mono text-[11px]">{chunk.applies_to_aez_codes.join(', ')}</div>
          </div>
        )}
      </div>
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
