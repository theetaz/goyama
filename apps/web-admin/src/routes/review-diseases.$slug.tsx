import { useState } from 'react';
import { Link, createFileRoute } from '@tanstack/react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, ChevronLeft, ExternalLink, ImagePlus, ShieldAlert, X } from 'lucide-react';

import {
  ApiError,
  api,
  getReviewer,
  type Disease,
  type FieldProvenance,
  type Media,
  type RecordStatus,
} from '@/lib/api';

export const Route = createFileRoute('/review-diseases/$slug')({
  component: DiseaseReviewDetailPage,
});

// Transitions UI mirrors the cultivation-step review page. The backend
// enforces the same set; this just gates which buttons we draw.
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

function DiseaseReviewDetailPage() {
  const { slug } = Route.useParams();
  const qc = useQueryClient();
  const [notes, setNotes] = useState('');

  const disease = useQuery({
    queryKey: ['disease-review', slug],
    queryFn: () => api.getDisease(slug),
  });

  const mutation = useMutation({
    mutationFn: (to: RecordStatus) =>
      api.updateDiseaseStatus(slug, {
        status: to,
        review_notes: notes.trim() || undefined,
      }),
    onSuccess: (updated) => {
      qc.setQueryData(['disease-review', slug], updated);
      qc.invalidateQueries({ queryKey: ['disease-review-queue'] });
      setNotes('');
    },
  });

  const reviewer = getReviewer();

  return (
    <div className="space-y-5">
      <Link
        to="/review-diseases"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ChevronLeft className="h-4 w-4" aria-hidden />
        Disease review
      </Link>

      {disease.isLoading && <p>Loading…</p>}
      {disease.isError && (
        <p className="text-destructive" role="alert">
          {disease.error instanceof Error ? disease.error.message : 'Failed to load record'}
        </p>
      )}

      {disease.data && (
        <article className="grid grid-cols-1 gap-4 lg:grid-cols-[1fr_340px]">
          <div className="space-y-4">
            <DiseaseCard disease={disease.data} />
            <ImageGallery diseaseSlug={disease.data.slug} />
            {disease.data.field_provenance && Object.keys(disease.data.field_provenance).length > 0 && (
              <ProvenancePanel provenance={disease.data.field_provenance} />
            )}
          </div>

          <aside className="h-fit rounded-xl border bg-card p-4">
            <h2 className="text-sm font-semibold">Review actions</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Current status: <strong className="capitalize">{disease.data.status.replace('_', ' ')}</strong>
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
              {ACTIONS[disease.data.status].map((action) => (
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

            {disease.data.reviewed_by && (
              <div className="mt-4 border-t pt-3 text-xs text-muted-foreground">
                <div>
                  Last change by <strong>{disease.data.reviewed_by}</strong>
                </div>
                {disease.data.reviewed_at && (
                  <div>at {new Date(disease.data.reviewed_at).toLocaleString()}</div>
                )}
                {disease.data.review_notes && (
                  <p className="mt-1 whitespace-pre-line text-foreground">
                    “{disease.data.review_notes}”
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

function DiseaseCard({ disease }: { disease: Disease }) {
  const name = disease.names?.en ?? disease.slug;
  return (
    <section className="rounded-xl border bg-card p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">{name}</h1>
          {disease.scientific_name && (
            <p className="mt-1 text-sm italic text-muted-foreground">{disease.scientific_name}</p>
          )}
          <div className="mt-1 text-xs text-muted-foreground">
            <code>{disease.slug}</code>
          </div>
        </div>
        <div className="flex flex-wrap gap-1.5 text-[11px]">
          <span className="rounded-full bg-muted px-2 py-0.5 capitalize text-muted-foreground">
            {disease.causal_organism}
          </span>
          {disease.severity && (
            <span
              className={
                'rounded-full px-2 py-0.5 capitalize ' +
                (disease.severity === 'high'
                  ? 'bg-destructive/10 text-destructive'
                  : 'bg-muted text-muted-foreground')
              }
            >
              severity · {disease.severity}
            </span>
          )}
        </div>
      </div>

      {disease.causal_species && (
        <p className="mt-3 text-sm">
          <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Causal species
          </span>{' '}
          {disease.causal_species}
        </p>
      )}

      {disease.description?.en && (
        <div className="mt-3 rounded-lg border bg-background p-3 text-sm leading-relaxed">
          {disease.description.en}
        </div>
      )}

      <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-3">
        <Facts label="Affected crops" items={disease.affected_crop_slugs} />
        <Facts label="Affected parts" items={disease.affected_parts} />
        <Facts label="Transmission" items={disease.transmission} />
      </div>

      {disease.confused_with && disease.confused_with.length > 0 && (
        <p className="mt-3 text-xs text-muted-foreground">
          Often confused with: {disease.confused_with.join(', ')}
        </p>
      )}

      {disease.aliases && disease.aliases.length > 0 && (
        <div className="mt-3">
          <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Aliases
          </span>
          <ul className="mt-1 flex flex-wrap gap-1.5">
            {disease.aliases.map((a) => (
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
        Every claim must trace back to an authoritative source before this disease
        record is promoted. Verify each quote below before clicking "published".
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

function ImageGallery({ diseaseSlug }: { diseaseSlug: string }) {
  const qc = useQueryClient();
  const reviewer = getReviewer();

  const images = useQuery({
    queryKey: ['disease-images', diseaseSlug],
    queryFn: () => api.listMediaForEntity('disease', diseaseSlug),
    retry: false,
  });

  const reviewedCount = (images.data?.items ?? []).filter(
    (m) => m.status === 'published',
  ).length;

  return (
    <section className="rounded-xl border bg-card p-5">
      <header className="flex items-center justify-between gap-2">
        <div>
          <h2 className="text-sm font-semibold">Reference images</h2>
          <p className="mt-0.5 text-xs text-muted-foreground">
            Phase-1 publish gate: ≥ 3 reviewed images per disease.{' '}
            <strong className={reviewedCount >= 3 ? 'text-primary' : 'text-destructive'}>
              {reviewedCount}/3 reviewed
            </strong>
          </p>
        </div>
      </header>

      <AttachImageForm diseaseSlug={diseaseSlug} disabled={!reviewer} />

      {images.isLoading && <p className="mt-3 text-sm">Loading images…</p>}
      {images.error instanceof ApiError && images.error.status === 503 && (
        <p className="mt-3 rounded-md border bg-muted/30 p-3 text-xs text-muted-foreground">
          Image storage requires DATABASE_URL on the API.
        </p>
      )}

      {images.data && images.data.items.length === 0 && (
        <p className="mt-3 rounded-md border border-dashed bg-background p-3 text-center text-xs text-muted-foreground">
          No images attached yet. Paste an external URL above (Wikimedia, DOA gallery, agronomist upload to a public host).
        </p>
      )}

      {images.data && images.data.items.length > 0 && (
        <ul className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2">
          {images.data.items.map((m) => (
            <ImageTile
              key={m.slug}
              media={m}
              onUpdated={() =>
                qc.invalidateQueries({ queryKey: ['disease-images', diseaseSlug] })
              }
              disabled={!reviewer}
            />
          ))}
        </ul>
      )}
    </section>
  );
}

function AttachImageForm({
  diseaseSlug,
  disabled,
}: {
  diseaseSlug: string;
  disabled: boolean;
}) {
  const qc = useQueryClient();
  const [url, setUrl] = useState('');
  const [credit, setCredit] = useState('');
  const [licence, setLicence] = useState('CC-BY-SA 4.0');

  const mutation = useMutation({
    mutationFn: () =>
      api.attachMediaToEntity('disease', diseaseSlug, {
        external_url: url.trim(),
        credit: credit.trim() || undefined,
        licence: licence.trim(),
        type: 'image',
      }),
    onSuccess: () => {
      setUrl('');
      setCredit('');
      qc.invalidateQueries({ queryKey: ['disease-images', diseaseSlug] });
    },
  });

  return (
    <form
      className="mt-3 grid grid-cols-1 gap-2 rounded-md border bg-background p-3 sm:grid-cols-[1fr_180px_120px_auto]"
      onSubmit={(e) => {
        e.preventDefault();
        if (url.trim()) mutation.mutate();
      }}
    >
      <input
        value={url}
        onChange={(e) => setUrl(e.target.value)}
        placeholder="https://commons.wikimedia.org/wiki/File:Rice_blast.jpg"
        type="url"
        required
        className="rounded-md border bg-background px-2 py-1.5 text-sm"
      />
      <input
        value={credit}
        onChange={(e) => setCredit(e.target.value)}
        placeholder="Credit (optional)"
        className="rounded-md border bg-background px-2 py-1.5 text-sm"
      />
      <input
        value={licence}
        onChange={(e) => setLicence(e.target.value)}
        placeholder="Licence"
        required
        className="rounded-md border bg-background px-2 py-1.5 text-sm"
      />
      <button
        type="submit"
        disabled={disabled || mutation.isPending || !url.trim()}
        className="inline-flex items-center justify-center gap-1 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground disabled:opacity-50"
      >
        <ImagePlus className="h-4 w-4" aria-hidden />
        {mutation.isPending ? 'Attaching…' : 'Attach'}
      </button>
      {mutation.isError && (
        <p className="col-span-full text-xs text-destructive" role="alert">
          {mutation.error instanceof ApiError ? mutation.error.problem.detail : 'Attach failed'}
        </p>
      )}
    </form>
  );
}

function ImageTile({
  media,
  onUpdated,
  disabled,
}: {
  media: Media;
  onUpdated: () => void;
  disabled: boolean;
}) {
  const mutation = useMutation({
    mutationFn: (status: RecordStatus) =>
      api.updateMediaStatus(media.slug, { status }),
    onSuccess: onUpdated,
  });

  const src = media.external_url || media.url;
  const isPublished = media.status === 'published';
  const isRejected = media.status === 'rejected';

  return (
    <li className="flex flex-col overflow-hidden rounded-md border bg-background">
      <a
        href={src}
        target="_blank"
        rel="noreferrer"
        className="relative block aspect-video bg-muted/30"
      >
        {src && (
          <img
            src={src}
            alt={media.credit || media.slug}
            className="absolute inset-0 h-full w-full object-cover"
            loading="lazy"
          />
        )}
        <span
          className={
            'absolute right-2 top-2 rounded-full px-2 py-0.5 text-[10px] capitalize ' +
            (isPublished
              ? 'bg-primary text-primary-foreground'
              : isRejected
                ? 'bg-destructive/80 text-destructive-foreground'
                : 'bg-card text-muted-foreground')
          }
        >
          {media.status.replace('_', ' ')}
        </span>
      </a>
      <div className="flex flex-1 flex-col gap-1 p-2 text-xs">
        {media.credit && <span className="text-muted-foreground">{media.credit}</span>}
        <span className="text-[10px] text-muted-foreground">{media.licence}</span>
        <div className="mt-auto flex gap-1.5 pt-2">
          {!isPublished && (
            <button
              type="button"
              disabled={disabled || mutation.isPending}
              onClick={() => mutation.mutate('published')}
              className="inline-flex flex-1 items-center justify-center gap-1 rounded-md bg-primary px-2 py-1 text-xs font-medium text-primary-foreground disabled:opacity-50"
            >
              <Check className="h-3 w-3" aria-hidden />
              Approve
            </button>
          )}
          {!isRejected && media.status !== 'published' && (
            <button
              type="button"
              disabled={disabled || mutation.isPending}
              onClick={() => mutation.mutate('rejected')}
              className="inline-flex flex-1 items-center justify-center gap-1 rounded-md border border-destructive/40 px-2 py-1 text-xs text-destructive disabled:opacity-50"
            >
              <X className="h-3 w-3" aria-hidden />
              Reject
            </button>
          )}
          {isPublished && (
            <button
              type="button"
              disabled={disabled || mutation.isPending}
              onClick={() => mutation.mutate('deprecated')}
              className="inline-flex flex-1 items-center justify-center rounded-md border px-2 py-1 text-xs text-muted-foreground disabled:opacity-50"
            >
              Deprecate
            </button>
          )}
        </div>
        {mutation.isError && (
          <p className="text-[11px] text-destructive" role="alert">
            {mutation.error instanceof ApiError ? mutation.error.problem.detail : 'Update failed'}
          </p>
        )}
      </div>
    </li>
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
