import { Link, createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { ShieldAlert } from 'lucide-react';

import { api, type DiseaseSummary } from '@/lib/api';
import { pickLocalised, type Locale } from '@/i18n';

export const Route = createFileRoute('/diseases')({
  component: DiseasesListPage,
});

function DiseasesListPage() {
  const { t, i18n } = useTranslation();
  const locale = i18n.language as Locale;
  const diseases = useQuery({
    queryKey: ['diseases'],
    queryFn: () => api.listDiseases(),
  });

  return (
    <div className="space-y-5">
      <header>
        <h1 className="flex items-center gap-2 text-2xl font-semibold">
          <ShieldAlert className="h-6 w-6 text-primary" aria-hidden />
          {t('pathology.diseases_title')}
        </h1>
        <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
          {t('pathology.diseases_subtitle')}
        </p>
      </header>

      {diseases.isLoading && <p>{t('pathology.loading')}</p>}
      {diseases.isError && (
        <p role="alert" className="text-destructive">
          {diseases.error instanceof Error ? diseases.error.message : t('errors.generic')}
        </p>
      )}
      {diseases.data && diseases.data.items.length === 0 && (
        <p className="rounded-lg border bg-card p-6 text-center text-sm text-muted-foreground">
          {t('pathology.empty_published')}
        </p>
      )}
      {diseases.data && diseases.data.items.length > 0 && (
        <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {diseases.data.items.map((d) => (
            <DiseaseCard key={d.slug} disease={d} locale={locale} />
          ))}
        </ul>
      )}
    </div>
  );
}

function DiseaseCard({ disease, locale }: { disease: DiseaseSummary; locale: Locale }) {
  const name = pickLocalised(disease.names, locale) ?? disease.slug;
  return (
    <li>
      <Link
        to="/diseases/$slug"
        params={{ slug: disease.slug }}
        className="block rounded-xl border bg-card p-4 transition-colors hover:border-primary"
      >
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="font-semibold">{name}</h2>
          <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] capitalize text-muted-foreground">
            {disease.causal_organism}
          </span>
          {disease.severity && (
            <span
              className={
                'rounded-full px-2 py-0.5 text-[11px] capitalize ' +
                (disease.severity === 'high'
                  ? 'bg-destructive/10 text-destructive'
                  : 'bg-muted text-muted-foreground')
              }
            >
              {disease.severity}
            </span>
          )}
        </div>
        {disease.scientific_name && (
          <p className="mt-1 text-xs italic text-muted-foreground">{disease.scientific_name}</p>
        )}
        {disease.affected_crop_slugs && disease.affected_crop_slugs.length > 0 && (
          <p className="mt-2 text-xs text-muted-foreground">
            {disease.affected_crop_slugs.slice(0, 5).join(', ')}
            {disease.affected_crop_slugs.length > 5 && '…'}
          </p>
        )}
      </Link>
    </li>
  );
}
