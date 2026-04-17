import { Link, createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Bug } from 'lucide-react';

import { api, type PestSummary } from '@/lib/api';
import { pickLocalised, type Locale } from '@/i18n';

export const Route = createFileRoute('/pests')({
  component: PestsListPage,
});

function PestsListPage() {
  const { t, i18n } = useTranslation();
  const locale = i18n.language as Locale;
  const pests = useQuery({
    queryKey: ['pests'],
    queryFn: () => api.listPests(),
  });

  return (
    <div className="space-y-5">
      <header>
        <h1 className="flex items-center gap-2 text-2xl font-semibold">
          <Bug className="h-6 w-6 text-primary" aria-hidden />
          {t('pathology.pests_title')}
        </h1>
        <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
          {t('pathology.pests_subtitle')}
        </p>
      </header>

      {pests.isLoading && <p>{t('pathology.loading')}</p>}
      {pests.isError && (
        <p role="alert" className="text-destructive">
          {pests.error instanceof Error ? pests.error.message : t('errors.generic')}
        </p>
      )}
      {pests.data && pests.data.items.length === 0 && (
        <p className="rounded-lg border bg-card p-6 text-center text-sm text-muted-foreground">
          {t('pathology.empty_published')}
        </p>
      )}
      {pests.data && pests.data.items.length > 0 && (
        <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {pests.data.items.map((p) => (
            <PestCard key={p.slug} pest={p} locale={locale} />
          ))}
        </ul>
      )}
    </div>
  );
}

function PestCard({ pest, locale }: { pest: PestSummary; locale: Locale }) {
  const name = pickLocalised(pest.names, locale) ?? pest.slug;
  return (
    <li>
      <Link
        to="/pests/$slug"
        params={{ slug: pest.slug }}
        className="block rounded-xl border bg-card p-4 transition-colors hover:border-primary"
      >
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="font-semibold">{name}</h2>
          <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] capitalize text-muted-foreground">
            {pest.kingdom}
          </span>
          {pest.feeding_type && pest.feeding_type.length > 0 && (
            <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] capitalize text-muted-foreground">
              {pest.feeding_type.slice(0, 2).join(', ').replace(/_/g, ' ')}
            </span>
          )}
        </div>
        {pest.scientific_name && (
          <p className="mt-1 text-xs italic text-muted-foreground">{pest.scientific_name}</p>
        )}
        {pest.affected_crop_slugs && pest.affected_crop_slugs.length > 0 && (
          <p className="mt-2 text-xs text-muted-foreground">
            {pest.affected_crop_slugs.slice(0, 5).join(', ')}
            {pest.affected_crop_slugs.length > 5 && '…'}
          </p>
        )}
      </Link>
    </li>
  );
}
