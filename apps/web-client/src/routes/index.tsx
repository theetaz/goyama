import { useState } from 'react';
import { Link, createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Search } from 'lucide-react';

import { api, type CropSummary } from '@/lib/api';
import { pickLocalised, type Locale } from '@/i18n';
import { cn } from '@/lib/utils';

export const Route = createFileRoute('/')({
  component: HomePage,
});

const categoryFilters = [
  { value: '', labelKey: 'crops.filter_all' },
  { value: 'field_crop', labelKey: 'crops.filter_field_crop' },
  { value: 'vegetable', labelKey: 'crops.filter_vegetable' },
  { value: 'fruit', labelKey: 'crops.filter_fruit' },
  { value: 'spice', labelKey: 'crops.filter_spice' },
  { value: 'plantation', labelKey: 'crops.filter_plantation' },
  { value: 'medicinal', labelKey: 'crops.filter_medicinal' },
] as const;

function HomePage() {
  const { t, i18n } = useTranslation();
  const locale = i18n.language as Locale;
  const [query, setQuery] = useState('');
  const [category, setCategory] = useState<string>('');

  const crops = useQuery({
    queryKey: ['crops', { category, query }],
    queryFn: () => api.listCrops({ category: category || undefined, q: query || undefined, limit: 100 }),
  });

  return (
    <div className="space-y-6">
      <section className="rounded-2xl bg-gradient-to-br from-primary/10 via-secondary to-accent/10 p-6 md:p-8">
        <h1 className="text-2xl font-semibold md:text-3xl">{t('home.welcome')}</h1>
        <p className="mt-2 max-w-2xl text-sm text-muted-foreground md:text-base">
          {t('home.subtitle')}
        </p>
      </section>

      <section className="space-y-4">
        <div className="flex items-center gap-3">
          <h2 className="text-xl font-semibold">{t('crops.title')}</h2>
          {crops.isSuccess && (
            <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
              {crops.data.count}
            </span>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {categoryFilters.map((f) => (
            <button
              key={f.value || 'all'}
              type="button"
              onClick={() => setCategory(f.value)}
              className={cn(
                'rounded-full border px-3 py-1.5 text-sm transition-colors duration-micro',
                category === f.value
                  ? 'border-primary bg-primary text-primary-foreground'
                  : 'border-border bg-background hover:bg-muted',
              )}
            >
              {t(f.labelKey)}
            </button>
          ))}
        </div>

        <div className="relative max-w-xl">
          <Search
            className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
            aria-hidden
          />
          <input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t('crops.search_placeholder')}
            className="w-full rounded-lg border bg-background py-2.5 pl-10 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            aria-label={t('crops.search_placeholder')}
          />
        </div>

        {crops.isLoading && <p className="text-sm text-muted-foreground">{t('crops.loading')}</p>}
        {crops.isError && (
          <p className="text-sm text-destructive" role="alert">
            {t('errors.api_unreachable')}
          </p>
        )}
        {crops.isSuccess && crops.data.items.length === 0 && (
          <p className="text-sm text-muted-foreground">{t('crops.empty')}</p>
        )}
        {crops.isSuccess && crops.data.items.length > 0 && (
          <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {crops.data.items.map((c) => (
              <CropCard key={c.slug} crop={c} locale={locale} />
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function CropCard({ crop, locale }: { crop: CropSummary; locale: Locale }) {
  const displayName = pickLocalised(crop.names, locale) ?? crop.slug;
  return (
    <li>
      <Link
        to="/crops/$slug"
        params={{ slug: crop.slug }}
        className="group block rounded-xl border bg-card p-4 transition-shadow duration-element ease-standard hover:shadow-md"
      >
        <div className="flex items-start justify-between gap-2">
          <h3 className="text-lg font-medium text-card-foreground group-hover:text-primary">
            {displayName}
          </h3>
          {crop.category && (
            <span className="rounded-full bg-muted px-2 py-0.5 text-xs capitalize text-muted-foreground">
              {crop.category.replace('_', ' ')}
            </span>
          )}
        </div>
        {crop.scientific_name && (
          <p className="mt-1 text-xs italic text-muted-foreground">{crop.scientific_name}</p>
        )}
      </Link>
    </li>
  );
}
