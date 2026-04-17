import { Link, createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { FlaskConical } from 'lucide-react';

import { api, type RemedySummary, type RemedyType } from '@/lib/api';
import { pickLocalised, type Locale } from '@/i18n';

export const Route = createFileRoute('/remedies')({
  component: RemediesListPage,
});

// Colour-code remedy types so a farmer can quickly filter visually —
// chemicals stand out destructively, organics read positive.
const TYPE_TONE: Record<RemedyType, string> = {
  chemical: 'bg-destructive/10 text-destructive',
  biological: 'bg-emerald-500/10 text-emerald-700',
  cultural: 'bg-sky-500/10 text-sky-700',
  resistant_variety: 'bg-amber-500/10 text-amber-700',
  mechanical: 'bg-muted text-muted-foreground',
  integrated: 'bg-violet-500/10 text-violet-700',
};

function RemediesListPage() {
  const { t, i18n } = useTranslation();
  const locale = i18n.language as Locale;
  const remedies = useQuery({
    queryKey: ['remedies'],
    queryFn: () => api.listRemedies(),
  });

  return (
    <div className="space-y-5">
      <header>
        <h1 className="flex items-center gap-2 text-2xl font-semibold">
          <FlaskConical className="h-6 w-6 text-primary" aria-hidden />
          {t('pathology.remedies_title')}
        </h1>
        <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
          {t('pathology.remedies_subtitle')}
        </p>
      </header>

      {remedies.isLoading && <p>{t('pathology.loading')}</p>}
      {remedies.isError && (
        <p role="alert" className="text-destructive">
          {remedies.error instanceof Error ? remedies.error.message : t('errors.generic')}
        </p>
      )}
      {remedies.data && remedies.data.items.length === 0 && (
        <p className="rounded-lg border bg-card p-6 text-center text-sm text-muted-foreground">
          {t('pathology.empty_published')}
        </p>
      )}
      {remedies.data && remedies.data.items.length > 0 && (
        <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {remedies.data.items.map((r) => (
            <RemedyCard key={r.slug} remedy={r} locale={locale} />
          ))}
        </ul>
      )}
    </div>
  );
}

function RemedyCard({ remedy, locale }: { remedy: RemedySummary; locale: Locale }) {
  const { t } = useTranslation();
  const name = pickLocalised(remedy.name, locale) ?? remedy.slug;
  return (
    <li>
      <Link
        to="/remedies/$slug"
        params={{ slug: remedy.slug }}
        className="block rounded-xl border bg-card p-4 transition-colors hover:border-primary"
      >
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="font-semibold">{name}</h2>
          <span
            className={
              'rounded-full px-2 py-0.5 text-[11px] capitalize ' + TYPE_TONE[remedy.type]
            }
          >
            {remedy.type.replace('_', ' ')}
          </span>
          {remedy.pre_harvest_interval_days != null && (
            <span className="rounded-full bg-primary/10 px-2 py-0.5 text-[11px] text-primary">
              {t('pathology.phi')} {remedy.pre_harvest_interval_days}d
            </span>
          )}
          {remedy.organic_compatible === true && (
            <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] text-emerald-700">
              {t('pathology.organic_compatible')}
            </span>
          )}
        </div>
        {remedy.active_ingredient && (
          <p className="mt-1 text-xs text-muted-foreground">AI: {remedy.active_ingredient}</p>
        )}
      </Link>
    </li>
  );
}
