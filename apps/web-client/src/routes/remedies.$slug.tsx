import { Link, createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { AlertTriangle, ChevronLeft, ExternalLink } from 'lucide-react';

import { api } from '@/lib/api';
import { pickLocalised, type Locale } from '@/i18n';

export const Route = createFileRoute('/remedies/$slug')({
  component: RemedyDetailPage,
});

interface ProvenanceEntry {
  source_id?: string;
  source_url?: string;
}

function RemedyDetailPage() {
  const { slug } = Route.useParams();
  const { t, i18n } = useTranslation();
  const locale = i18n.language as Locale;

  const remedy = useQuery({
    queryKey: ['remedy', slug],
    queryFn: () => api.getRemedy(slug),
  });

  return (
    <div className="space-y-5">
      <Link
        to="/remedies"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ChevronLeft className="h-4 w-4" aria-hidden />
        {t('pathology.remedies_title')}
      </Link>

      {remedy.isLoading && <p>{t('pathology.loading')}</p>}
      {remedy.isError && (
        <p role="alert" className="text-destructive">
          {remedy.error instanceof Error ? remedy.error.message : t('errors.generic')}
        </p>
      )}

      {remedy.data && (
        <article className="space-y-4">
          <header>
            <h1 className="text-3xl font-semibold">
              {pickLocalised(remedy.data.name, locale) ?? remedy.data.slug}
            </h1>
            <div className="mt-3 flex flex-wrap gap-1.5 text-[11px]">
              <span className="rounded-full bg-muted px-2 py-0.5 capitalize text-muted-foreground">
                {t('pathology.remedy_type')} · {remedy.data.type.replace('_', ' ')}
              </span>
              {remedy.data.effectiveness && (
                <span className="rounded-full bg-muted px-2 py-0.5 capitalize text-muted-foreground">
                  {remedy.data.effectiveness}
                </span>
              )}
              {remedy.data.cost_tier && (
                <span className="rounded-full bg-muted px-2 py-0.5 capitalize text-muted-foreground">
                  {remedy.data.cost_tier} cost
                </span>
              )}
              {remedy.data.organic_compatible === true && (
                <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-emerald-700">
                  {t('pathology.organic_compatible')}
                </span>
              )}
            </div>
          </header>

          {remedy.data.type === 'chemical' && (
            <div className="flex items-start gap-2 rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">
              <AlertTriangle className="mt-0.5 h-5 w-5 flex-shrink-0" aria-hidden />
              <p className="leading-relaxed">{t('pathology.chemical_warning')}</p>
            </div>
          )}

          {pickLocalised(remedy.data.description, locale) && (
            <section className="rounded-xl border bg-card p-5 leading-relaxed">
              {pickLocalised(remedy.data.description, locale)}
            </section>
          )}

          {remedy.data.type === 'chemical' && (
            <section className="rounded-xl border bg-card p-4">
              <dl className="grid grid-cols-1 gap-y-1.5 text-sm md:grid-cols-2">
                <Row label={t('pathology.active_ingredient')} value={remedy.data.active_ingredient} />
                <Row label={t('pathology.concentration')} value={remedy.data.concentration} />
                <Row label={t('pathology.formulation')} value={remedy.data.formulation} />
                <Row
                  label={t('pathology.application_method')}
                  value={remedy.data.application_method?.replace(/_/g, ' ')}
                />
                <Row
                  label={t('pathology.phi')}
                  value={
                    remedy.data.pre_harvest_interval_days != null
                      ? `${remedy.data.pre_harvest_interval_days} days`
                      : undefined
                  }
                />
                <Row
                  label={t('pathology.re_entry')}
                  value={
                    remedy.data.re_entry_interval_hours != null
                      ? `${remedy.data.re_entry_interval_hours} hours`
                      : undefined
                  }
                />
                <Row label={t('pathology.who_hazard')} value={remedy.data.who_hazard_class} />
              </dl>
              {remedy.data.dosage && (
                <p className="mt-3 text-sm">
                  <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                    {t('pathology.dosage')}
                  </span>{' '}
                  {remedy.data.dosage}
                </p>
              )}
              {pickLocalised(remedy.data.safety_notes, locale) && (
                <p className="mt-3 rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
                  <strong>{t('pathology.safety')}:</strong>{' '}
                  {pickLocalised(remedy.data.safety_notes, locale)}
                </p>
              )}
            </section>
          )}

          {pickLocalised(remedy.data.instructions, locale) && (
            <section className="rounded-xl border bg-card p-5">
              <h2 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                {t('pathology.instructions')}
              </h2>
              <p className="mt-2 leading-relaxed">
                {pickLocalised(remedy.data.instructions, locale)}
              </p>
            </section>
          )}

          <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
            <Facts label={t('pathology.targets_diseases')} items={remedy.data.target_disease_slugs} />
            <Facts label={t('pathology.targets_pests')} items={remedy.data.target_pest_slugs} />
            <Facts label={t('pathology.applicable_crops')} items={remedy.data.applicable_crop_slugs} />
          </div>

          <SourceFooter provenance={remedy.data.field_provenance} />
        </article>
      )}
    </div>
  );
}

function Row({ label, value }: { label: string; value?: string }) {
  if (!value) return null;
  return (
    <div className="flex gap-2">
      <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd>{value}</dd>
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
