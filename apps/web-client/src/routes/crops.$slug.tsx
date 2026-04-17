import { Link, createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { ChevronLeft } from 'lucide-react';

import { api, type Range } from '@/lib/api';
import { pickLocalised, type Locale } from '@/i18n';

export const Route = createFileRoute('/crops/$slug')({
  component: CropDetailPage,
});

function CropDetailPage() {
  const { slug } = Route.useParams();
  const { t, i18n } = useTranslation();
  const locale = i18n.language as Locale;

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['crop', slug],
    queryFn: () => api.getCrop(slug),
  });

  return (
    <div className="space-y-6">
      <Link
        to="/"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ChevronLeft className="h-4 w-4" aria-hidden />
        {t('nav.explore')}
      </Link>

      {isLoading && <p>{t('crops.loading')}</p>}

      {isError && (
        <p className="text-destructive" role="alert">
          {error instanceof Error ? error.message : t('errors.generic')}
        </p>
      )}

      {data && (
        <article className="space-y-6">
          <header>
            <h1 className="text-3xl font-semibold">{pickLocalised(data.names, locale) ?? data.slug}</h1>
            {data.scientific_name && (
              <p className="mt-1 text-lg italic text-muted-foreground">{data.scientific_name}</p>
            )}
            <div className="mt-3 flex flex-wrap gap-2 text-xs">
              {data.category && <Chip label={t('crop_detail.category')} value={data.category} />}
              {data.family && <Chip label={t('crop_detail.family')} value={data.family} />}
              {data.life_cycle && <Chip value={data.life_cycle} />}
              {data.growth_habit && <Chip value={data.growth_habit} />}
              {data.default_season && <Chip value={data.default_season} />}
            </div>
          </header>

          {data.description && pickLocalised(data.description, locale) && (
            <section className="rounded-xl border bg-card p-5">
              <p className="leading-relaxed text-card-foreground">
                {pickLocalised(data.description, locale)}
              </p>
            </section>
          )}

          <dl className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <Field label={t('crop_detail.duration')} range={data.duration_days} />
            <Field label={t('crop_detail.elevation')} range={data.elevation_m} />
            <Field label={t('crop_detail.rainfall')} range={data.rainfall_mm} />
            <Field label={t('crop_detail.temperature')} range={data.temperature_c} />
            <Field label={t('crop_detail.soil_ph')} range={data.soil_ph} />
            <Field label={t('crop_detail.yield')} range={data.expected_yield_kg_per_acre} />
          </dl>

          {data.aliases && data.aliases.length > 0 && (
            <section>
              <h2 className="text-sm font-semibold text-muted-foreground">
                {t('crop_detail.also_known_as')}
              </h2>
              <ul className="mt-2 flex flex-wrap gap-2">
                {data.aliases.map((a) => (
                  <li key={a} className="rounded-md bg-muted px-2 py-1 text-xs">
                    {a}
                  </li>
                ))}
              </ul>
            </section>
          )}
        </article>
      )}
    </div>
  );
}

function Chip({ label, value }: { label?: string; value: string }) {
  return (
    <span className="rounded-full bg-muted px-2.5 py-1 text-xs capitalize text-muted-foreground">
      {label && <span className="mr-1 text-[0.7rem] uppercase tracking-wide opacity-60">{label}</span>}
      {value.replace(/_/g, ' ')}
    </span>
  );
}

function Field({ label, range }: { label: string; range?: Range }) {
  if (!range || (range.min == null && range.max == null)) return null;
  const { min, max, unit } = range;
  const value =
    min != null && max != null && min !== max
      ? `${min}–${max}`
      : String(min ?? max);
  return (
    <div className="rounded-lg border bg-card p-4">
      <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className="mt-1 text-lg font-medium">
        {value} {unit && <span className="text-sm text-muted-foreground">{unit}</span>}
      </dd>
    </div>
  );
}
