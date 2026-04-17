import { Link, createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { ChevronLeft, ExternalLink } from 'lucide-react';

import { api } from '@/lib/api';
import { pickLocalised, type Locale } from '@/i18n';

export const Route = createFileRoute('/pests/$slug')({
  component: PestDetailPage,
});

interface ProvenanceEntry {
  source_id?: string;
  source_url?: string;
}

function PestDetailPage() {
  const { slug } = Route.useParams();
  const { t, i18n } = useTranslation();
  const locale = i18n.language as Locale;

  const pest = useQuery({
    queryKey: ['pest', slug],
    queryFn: () => api.getPest(slug),
  });

  return (
    <div className="space-y-5">
      <Link
        to="/pests"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ChevronLeft className="h-4 w-4" aria-hidden />
        {t('pathology.pests_title')}
      </Link>

      {pest.isLoading && <p>{t('pathology.loading')}</p>}
      {pest.isError && (
        <p role="alert" className="text-destructive">
          {pest.error instanceof Error ? pest.error.message : t('errors.generic')}
        </p>
      )}

      {pest.data && (
        <article className="space-y-4">
          <header>
            <h1 className="text-3xl font-semibold">
              {pickLocalised(pest.data.names, locale) ?? pest.data.slug}
            </h1>
            {pest.data.scientific_name && (
              <p className="mt-1 text-base italic text-muted-foreground">{pest.data.scientific_name}</p>
            )}
            <div className="mt-3 flex flex-wrap gap-1.5 text-[11px]">
              <span className="rounded-full bg-muted px-2 py-0.5 capitalize text-muted-foreground">
                {pest.data.kingdom}
              </span>
            </div>
          </header>

          {pickLocalised(pest.data.description, locale) && (
            <section className="rounded-xl border bg-card p-5 leading-relaxed">
              {pickLocalised(pest.data.description, locale)}
            </section>
          )}

          {pickLocalised(pest.data.economic_threshold, locale) && (
            <section className="rounded-xl border border-primary/30 bg-primary/5 p-4 text-sm">
              <div className="text-xs font-semibold uppercase tracking-wide text-primary">
                {t('pathology.economic_threshold')}
              </div>
              <p className="mt-1">{pickLocalised(pest.data.economic_threshold, locale)}</p>
            </section>
          )}

          <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
            <Facts label={t('pathology.affected_crops')} items={pest.data.affected_crop_slugs} />
            <Facts label={t('pathology.life_stages')} items={pest.data.life_stages} />
            <Facts label={t('pathology.feeding_type')} items={pest.data.feeding_type} />
          </div>

          {pest.data.aliases && pest.data.aliases.length > 0 && (
            <section>
              <h2 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                {t('pathology.also_known_as')}
              </h2>
              <ul className="mt-2 flex flex-wrap gap-2">
                {pest.data.aliases.map((a) => (
                  <li key={a} className="rounded-md bg-muted px-2 py-1 text-xs">
                    {a}
                  </li>
                ))}
              </ul>
            </section>
          )}

          <SourceFooter provenance={pest.data.field_provenance} />
        </article>
      )}
    </div>
  );
}

function Facts({ label, items }: { label: string; items?: string[] }) {
  if (!items || items.length === 0) return null;
  return (
    <div className="rounded-lg border bg-card p-3 text-xs">
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

function SourceFooter({ provenance }: { provenance?: Record<string, unknown> }) {
  const { t } = useTranslation();
  if (!provenance) return null;
  const entries = Object.values(provenance).filter(
    (v): v is ProvenanceEntry => typeof v === 'object' && v != null,
  );
  const bySource = new Map<string, string>();
  for (const e of entries) {
    if (e.source_id && e.source_url && !bySource.has(e.source_id)) {
      bySource.set(e.source_id, e.source_url);
    }
  }
  if (bySource.size === 0) return null;
  return (
    <footer className="rounded-xl border bg-muted/30 p-4 text-xs text-muted-foreground">
      <div className="font-semibold uppercase tracking-wide">{t('pathology.source')}</div>
      <ul className="mt-1.5 flex flex-wrap gap-2">
        {Array.from(bySource.entries()).map(([id, url]) => (
          <li key={id}>
            <a
              href={url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 text-primary hover:underline"
            >
              {id}
              <ExternalLink className="h-3 w-3" aria-hidden />
            </a>
          </li>
        ))}
      </ul>
    </footer>
  );
}
