import { useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import {
  AlertTriangle,
  ExternalLink,
  MapPin,
  MessageCircleQuestion,
  Send,
  Sparkles,
} from 'lucide-react';

import {
  ApiError,
  api,
  type AskHit,
  type AskRequest,
  type AskResponse,
} from '@/lib/api';
import { AuthorityChip } from '@/components/cultivation/authority-chip';
import { cn } from '@/lib/utils';

export const Route = createFileRoute('/ask')({
  component: AskPage,
});

function AskPage() {
  const { t } = useTranslation();
  const [question, setQuestion] = useState('');
  const [crop, setCrop] = useState('');
  const [location, setLocation] = useState<{ lat: number; lng: number } | null>(null);
  const [locError, setLocError] = useState<string | null>(null);

  const crops = useQuery({
    queryKey: ['crops-for-ask-picker'],
    queryFn: () => api.listCrops({ limit: 200 }),
  });

  const ask = useMutation({
    mutationFn: (body: AskRequest) => api.ask(body),
  });

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!question.trim()) return;
    ask.mutate({
      question: question.trim(),
      crop: crop || undefined,
      lat: location?.lat,
      lng: location?.lng,
      k: 6,
    });
  }

  function requestLocation() {
    setLocError(null);
    if (!navigator.geolocation) {
      setLocError(t('ask.location_unsupported'));
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (pos) => setLocation({ lat: pos.coords.latitude, lng: pos.coords.longitude }),
      (err) => setLocError(err.message),
      { enableHighAccuracy: false, timeout: 5000 },
    );
  }

  return (
    <div className="space-y-6">
      <header>
        <h1 className="flex items-center gap-2 text-2xl font-semibold">
          <MessageCircleQuestion className="h-6 w-6 text-primary" aria-hidden />
          {t('ask.title')}
        </h1>
        <p className="mt-1 max-w-3xl text-sm text-muted-foreground">{t('ask.subtitle')}</p>
      </header>

      <form
        onSubmit={onSubmit}
        className="space-y-3 rounded-xl border bg-card p-4"
      >
        <label className="block">
          <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            {t('ask.question_label')}
          </span>
          <textarea
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            rows={3}
            required
            placeholder={t('ask.question_placeholder')}
            className="mt-1 w-full resize-none rounded-md border bg-background px-3 py-2 text-sm"
          />
        </label>

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <label className="flex flex-col gap-1 text-sm">
            <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              {t('ask.crop_label')}
            </span>
            <select
              value={crop}
              onChange={(e) => setCrop(e.target.value)}
              className="rounded-md border bg-background px-3 py-1.5"
            >
              <option value="">{t('ask.crop_any')}</option>
              {crops.data?.items.map((c) => (
                <option key={c.slug} value={c.slug}>
                  {c.names?.en ?? c.slug}
                </option>
              ))}
            </select>
          </label>

          <div className="flex flex-col gap-1 text-sm">
            <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              {t('ask.location_label')}
            </span>
            {location ? (
              <div className="flex items-center justify-between rounded-md border bg-background px-3 py-1.5">
                <span className="inline-flex items-center gap-1 text-xs">
                  <MapPin className="h-3.5 w-3.5 text-primary" aria-hidden />
                  {location.lat.toFixed(3)}, {location.lng.toFixed(3)}
                </span>
                <button
                  type="button"
                  onClick={() => setLocation(null)}
                  className="text-xs text-muted-foreground hover:text-foreground"
                >
                  {t('ask.location_clear')}
                </button>
              </div>
            ) : (
              <button
                type="button"
                onClick={requestLocation}
                className="inline-flex items-center justify-center gap-1.5 rounded-md border bg-background px-3 py-1.5 text-sm hover:bg-muted"
              >
                <MapPin className="h-3.5 w-3.5" aria-hidden />
                {t('ask.location_share')}
              </button>
            )}
            {locError && <span className="text-[11px] text-destructive">{locError}</span>}
          </div>
        </div>

        <div className="flex items-center justify-between gap-2">
          <p className="text-[11px] text-muted-foreground">{t('ask.tip')}</p>
          <button
            type="submit"
            disabled={ask.isPending || !question.trim()}
            className="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
          >
            {ask.isPending ? (
              <>
                <Sparkles className="h-4 w-4 animate-pulse" aria-hidden />
                {t('ask.searching')}
              </>
            ) : (
              <>
                <Send className="h-4 w-4" aria-hidden />
                {t('ask.submit')}
              </>
            )}
          </button>
        </div>
      </form>

      {ask.isError && <AskError error={ask.error} />}
      {ask.data && <AskResults data={ask.data} />}
    </div>
  );
}

function AskError({ error }: { error: unknown }) {
  const { t } = useTranslation();
  const isApi = error instanceof ApiError;
  const msg = isApi ? error.problem.detail : t('errors.generic');
  const disabled = isApi && error.status === 503;
  return (
    <div
      role="alert"
      className={cn(
        'flex items-start gap-2 rounded-xl border p-4 text-sm',
        disabled ? 'border-amber-500/40 bg-amber-500/5 text-amber-800 dark:text-amber-300'
                 : 'border-destructive/40 bg-destructive/5 text-destructive',
      )}
    >
      <AlertTriangle className="mt-0.5 h-4 w-4 flex-shrink-0" aria-hidden />
      <div>
        <div className="font-medium">
          {disabled ? t('ask.disabled_title') : t('ask.error_title')}
        </div>
        <p className="mt-0.5 text-xs">{msg}</p>
      </div>
    </div>
  );
}

function AskResults({ data }: { data: AskResponse }) {
  const { t } = useTranslation();
  if (data.count === 0) {
    return (
      <div className="rounded-xl border bg-card p-6 text-center text-sm text-muted-foreground">
        {t('ask.empty')}
      </div>
    );
  }
  return (
    <section className="space-y-4" aria-live="polite">
      <header className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
        <span>{t('ask.results_count', { count: data.count })}</span>
        {data.used_district && (
          <span className="inline-flex items-center gap-1 rounded-full border bg-card px-2 py-0.5">
            <MapPin className="h-3 w-3" aria-hidden />
            {data.used_district}
          </span>
        )}
        {data.used_aez_codes?.map((aez) => (
          <span key={aez} className="rounded-full border bg-card px-2 py-0.5 font-mono">
            AEZ {aez}
          </span>
        ))}
        {data.used_crop && (
          <span className="rounded-full border bg-card px-2 py-0.5">crop: {data.used_crop}</span>
        )}
        <span className="ml-auto text-[11px] opacity-70">{t('ask.embedder')} {data.embedder}</span>
      </header>

      <ol className="space-y-3">
        {data.hits.map((h, i) => (
          <HitCard key={h.slug} hit={h} rank={i + 1} />
        ))}
      </ol>

      <p className="rounded-xl border bg-muted/30 p-3 text-[11px] leading-relaxed text-muted-foreground">
        {data.disclaimer}
      </p>
    </section>
  );
}

function HitCard({ hit, rank }: { hit: AskHit; rank: number }) {
  const { t } = useTranslation();
  const scorePct = Math.max(0, Math.min(1, hit.score)) * 100;
  return (
    <li className="rounded-xl border bg-card p-4">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-mono text-xs text-muted-foreground">#{rank}</span>
            {hit.title && <h3 className="font-semibold">{hit.title}</h3>}
            <AuthorityChip authority={hit.authority} />
          </div>
          {hit.source && (
            <p className="mt-0.5 text-xs text-muted-foreground">
              {hit.source.display_name}
              {hit.source.publisher && hit.source.publisher !== hit.source.display_name && (
                <> · {hit.source.publisher}</>
              )}
            </p>
          )}
        </div>
        <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
          <div className="flex w-20 items-center gap-1">
            <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
              <div
                aria-hidden
                className="h-full bg-primary"
                style={{ width: `${scorePct}%` }}
              />
            </div>
            <span className="tabular-nums">{scorePct.toFixed(0)}%</span>
          </div>
        </div>
      </div>

      <p className="mt-2 whitespace-pre-line text-sm leading-relaxed">{hit.body}</p>

      {hit.quote && (
        <blockquote className="mt-2 border-l-2 border-muted pl-3 text-xs italic text-muted-foreground">
          "{hit.quote}"
        </blockquote>
      )}

      <div className="mt-3 flex flex-wrap items-center gap-2 text-[11px]">
        {hit.topic_tags?.slice(0, 4).map((tag) => (
          <span key={tag} className="rounded bg-muted px-1.5 py-0.5 text-muted-foreground">
            #{tag}
          </span>
        ))}
        {hit.source?.url && (
          <a
            href={hit.source.url}
            target="_blank"
            rel="noreferrer"
            className="ml-auto inline-flex items-center gap-1 text-primary hover:underline"
          >
            {t('knowledge.view_source')}
            <ExternalLink className="h-3 w-3" aria-hidden />
          </a>
        )}
      </div>
    </li>
  );
}
