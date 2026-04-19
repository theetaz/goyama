import { useMemo, useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { ArrowDown, ArrowUp, Coins, Minus } from 'lucide-react';

import { ApiError, api, type MarketPrice } from '@/lib/api';

export const Route = createFileRoute('/markets')({
  component: MarketsPage,
});

const KNOWN_MARKETS = [
  { code: 'dambulla-dec', label: 'Dambulla DEC' },
  { code: 'meegoda-dec', label: 'Meegoda DEC' },
  { code: 'welisara-dec', label: 'Welisara DEC' },
] as const;

function MarketsPage() {
  const { t } = useTranslation();
  const [market, setMarket] = useState<string>(KNOWN_MARKETS[0].code);

  const latest = useQuery({
    queryKey: ['market-latest', market],
    queryFn: () => api.latestMarketPrices(market),
    retry: false,
  });

  // Pull a 7-day window for trend calculation. The current implementation
  // does the trend math client-side off the same payload — a separate
  // /v1/market-prices/series endpoint would be cleaner once we have more
  // than one market actively reporting.
  const window = useQuery({
    queryKey: ['market-window', market],
    queryFn: () => api.listMarketPrices({ market, limit: 500 }),
    retry: false,
  });

  const trends = useMemo(() => buildTrends(window.data?.items ?? []), [window.data]);

  return (
    <div className="space-y-5">
      <header>
        <h1 className="flex items-center gap-2 text-2xl font-semibold">
          <Coins className="h-6 w-6 text-primary" aria-hidden />
          {t('markets.title')}
        </h1>
        <p className="mt-1 max-w-3xl text-sm text-muted-foreground">{t('markets.subtitle')}</p>
      </header>

      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-sm">
          <span className="text-muted-foreground">{t('markets.market_label')}</span>
          <select
            value={market}
            onChange={(e) => setMarket(e.target.value)}
            className="rounded-md border bg-background px-3 py-1.5"
          >
            {KNOWN_MARKETS.map((m) => (
              <option key={m.code} value={m.code}>
                {m.label}
              </option>
            ))}
          </select>
        </label>
        {latest.data?.items[0] && (
          <p className="text-sm text-muted-foreground">
            {t('markets.observed_on')}{' '}
            <strong className="text-foreground">{latest.data.items[0].observed_on}</strong>
          </p>
        )}
      </div>

      {latest.isLoading && <p>{t('markets.loading')}</p>}
      {latest.error instanceof ApiError && latest.error.status === 503 && (
        <p className="rounded-lg border bg-card p-6 text-sm text-muted-foreground">
          {t('markets.disabled')}
        </p>
      )}
      {latest.error instanceof ApiError && latest.error.status === 404 && (
        <p className="rounded-lg border bg-card p-6 text-sm text-muted-foreground">
          {t('markets.empty')}
        </p>
      )}

      {latest.data && latest.data.items.length > 0 && (
        <div className="overflow-x-auto rounded-xl border bg-card">
          <table className="w-full text-sm">
            <thead className="border-b bg-muted/50 text-left text-xs uppercase tracking-wide text-muted-foreground">
              <tr>
                <th className="px-4 py-2">{t('markets.col_commodity')}</th>
                <th className="px-4 py-2">{t('markets.col_grade')}</th>
                <th className="px-4 py-2 text-right">{t('markets.col_min')}</th>
                <th className="px-4 py-2 text-right">{t('markets.col_avg')}</th>
                <th className="px-4 py-2 text-right">{t('markets.col_max')}</th>
                <th className="px-4 py-2 text-right">{t('markets.col_trend')}</th>
              </tr>
            </thead>
            <tbody>
              {latest.data.items.map((p) => {
                const trend = trends.get(trendKey(p));
                return (
                  <tr key={`${p.commodity_label}-${p.grade ?? ''}`} className="border-b last:border-0">
                    <td className="px-4 py-2 font-medium">{p.commodity_label}</td>
                    <td className="px-4 py-2 text-muted-foreground">{p.grade ?? '—'}</td>
                    <td className="px-4 py-2 text-right tabular-nums">{fmtLKR(p.price_lkr_per_kg_min)}</td>
                    <td className="px-4 py-2 text-right tabular-nums font-semibold">
                      {fmtLKR(p.price_lkr_per_kg_avg)}
                    </td>
                    <td className="px-4 py-2 text-right tabular-nums">{fmtLKR(p.price_lkr_per_kg_max)}</td>
                    <td className="px-4 py-2 text-right">
                      <Trend pct={trend} />
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      <p className="text-xs text-muted-foreground">{t('markets.disclaimer')}</p>
    </div>
  );
}

function fmtLKR(v: number | undefined): string {
  if (v == null) return '—';
  return new Intl.NumberFormat('en-LK', { maximumFractionDigits: 0 }).format(v);
}

function trendKey(p: { commodity_label: string; grade?: string }): string {
  return `${p.commodity_label}::${p.grade ?? ''}`;
}

/**
 * For each (commodity, grade), compute % change between the two most-recent
 * observation dates. Returns null when only one data point exists.
 */
function buildTrends(rows: MarketPrice[]): Map<string, number | null> {
  const grouped = new Map<string, MarketPrice[]>();
  for (const r of rows) {
    const k = trendKey(r);
    const arr = grouped.get(k) ?? [];
    arr.push(r);
    grouped.set(k, arr);
  }
  const out = new Map<string, number | null>();
  for (const [k, arr] of grouped) {
    arr.sort((a, b) => (a.observed_on < b.observed_on ? 1 : -1));
    const today = arr[0]?.price_lkr_per_kg_avg;
    const previous = arr[1]?.price_lkr_per_kg_avg;
    if (today != null && previous != null && previous > 0) {
      out.set(k, ((today - previous) / previous) * 100);
    } else {
      out.set(k, null);
    }
  }
  return out;
}

function Trend({ pct }: { pct: number | null | undefined }) {
  if (pct == null) {
    return (
      <span className="inline-flex items-center gap-1 text-xs text-muted-foreground">
        <Minus className="h-3 w-3" aria-hidden />
      </span>
    );
  }
  if (Math.abs(pct) < 0.5) {
    return (
      <span className="inline-flex items-center gap-1 text-xs text-muted-foreground">
        <Minus className="h-3 w-3" aria-hidden />
        flat
      </span>
    );
  }
  const up = pct > 0;
  return (
    <span
      className={
        'inline-flex items-center gap-1 text-xs tabular-nums ' +
        (up ? 'text-destructive' : 'text-emerald-600')
      }
    >
      {up ? <ArrowUp className="h-3 w-3" aria-hidden /> : <ArrowDown className="h-3 w-3" aria-hidden />}
      {pct > 0 ? '+' : ''}
      {pct.toFixed(1)}%
    </span>
  );
}
