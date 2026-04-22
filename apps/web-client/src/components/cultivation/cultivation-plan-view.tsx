import { Fragment, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  AlertTriangle, CalendarDays, CloudRain, Coins, Droplets, ExternalLink,
  Flame, Leaf, Scissors, Scissors as ScissorsIcon, Sprout, SunMedium,
  TestTubeDiagonal, Truck, Wheat,
} from 'lucide-react';

import type {
  CultivationActivity,
  CultivationActivityInput,
  CultivationEconomics,
  CultivationPestRisk,
  CultivationPlan,
} from '@/lib/api';
import { pickLocalised, type Locale } from '@/i18n';
import { cn } from '@/lib/utils';

import { AuthorityChip } from './authority-chip';

const WEATHER_ICON: Record<string, typeof SunMedium> = {
  dry: SunMedium,
  mixed: CloudRain,
  rainy: CloudRain,
};

// Maps each activity_type to a lucide icon. Farmer-friendly symbols:
// seeds look like sprouts, harvest looks like scissors, chemicals get
// the test-tube glyph. Keeps the week strip readable without depending
// on colour alone.
const ACTIVITY_ICON: Record<string, typeof Sprout> = {
  land_prep: CalendarDays,
  basal_fertilizer: TestTubeDiagonal,
  seed_sowing: Sprout,
  transplanting: Sprout,
  herbicide_pre: Flame,
  herbicide_post: Flame,
  top_dressing: TestTubeDiagonal,
  irrigation: Droplets,
  weed_control: Leaf,
  pest_monitoring: AlertTriangle,
  pollination_support: Leaf,
  seed_harvest: Wheat,
  harvest: Scissors,
  post_harvest: Truck,
};

const RISK_TONE: Record<CultivationPestRisk['risk'], string> = {
  low: 'bg-emerald-500/10 text-emerald-700 border-emerald-500/30 dark:text-emerald-400',
  moderate: 'bg-amber-500/10 text-amber-700 border-amber-500/30 dark:text-amber-400',
  high: 'bg-destructive/10 text-destructive border-destructive/40',
};

export function CultivationPlanView({
  plan,
  locale,
}: {
  plan: CultivationPlan;
  locale: Locale;
}) {
  const { t } = useTranslation();

  // Re-bucket activities and pest risks by week so the timeline strip
  // and per-week drill-down don't need to re-iterate the source lists.
  const weeks = useMemo(() => buildWeeks(plan), [plan]);

  const title = pickLocalised(plan.title, locale) ?? plan.slug;
  const summary = pickLocalised(plan.summary, locale);

  return (
    <article className="space-y-6">
      <header className="space-y-2">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <h2 className="text-xl font-semibold">{title}</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              {t('plan.season')}: <strong className="capitalize text-foreground">{plan.season.replace('_', ' ')}</strong>
              {plan.duration_weeks != null && (
                <> · {plan.duration_weeks} {t('plan.weeks')}</>
              )}
              {plan.aez_codes && plan.aez_codes.length > 0 && (
                <> · {t('plan.aez_count', { count: plan.aez_codes.length })}</>
              )}
            </p>
          </div>
          <AuthorityChip authority={plan.authority} size="md" />
        </div>

        {summary && (
          <p className="text-sm leading-relaxed text-muted-foreground">{summary}</p>
        )}

        {plan.source_document_url && (
          <a
            href={plan.source_document_url}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
          >
            {plan.source_document_title ?? t('plan.source')}
            <ExternalLink className="h-3 w-3" aria-hidden />
          </a>
        )}
      </header>

      <TimelineStrip weeks={weeks} locale={locale} />
      <FertilizerSchedule activities={plan.activities} locale={locale} />
      <PestRiskTable risks={plan.pest_risks} locale={locale} />
      {plan.economics.length > 0 && (
        <EconomicsPanel economics={plan.economics[0]} locale={locale} />
      )}
    </article>
  );
}

// ─── timeline strip ────────────────────────────────────────────────────────

interface WeekBucket {
  weekIdx: number;
  activities: CultivationActivity[];
  risks: CultivationPestRisk[];
  weatherHint?: string;
}

function buildWeeks(plan: CultivationPlan): WeekBucket[] {
  const total = plan.duration_weeks ?? Math.max(
    ...plan.activities.map((a) => a.week_idx),
    ...plan.pest_risks.map((p) => p.week_idx),
    1,
  );
  const byWeek = new Map<number, WeekBucket>();
  for (let i = 1; i <= total; i++) {
    byWeek.set(i, { weekIdx: i, activities: [], risks: [] });
  }
  for (const a of plan.activities) {
    const b = byWeek.get(a.week_idx);
    if (!b) continue;
    b.activities.push(a);
    if (a.weather_hint) b.weatherHint = a.weather_hint;
  }
  for (const r of plan.pest_risks) {
    const b = byWeek.get(r.week_idx);
    if (!b) continue;
    b.risks.push(r);
  }
  return Array.from(byWeek.values());
}

function TimelineStrip({ weeks, locale }: { weeks: WeekBucket[]; locale: Locale }) {
  const { t } = useTranslation();
  const weeksWithContent = weeks.filter((w) => w.activities.length > 0 || w.risks.length > 0);

  return (
    <section aria-labelledby="timeline-heading" className="space-y-3">
      <h3 id="timeline-heading" className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        {t('plan.timeline')}
      </h3>
      <div className="overflow-x-auto rounded-xl border bg-card">
        <div className="flex min-w-max">
          {weeks.map((w) => (
            <WeekCell key={w.weekIdx} week={w} hasContent={weeksWithContent.includes(w)} />
          ))}
        </div>
      </div>
      <div className="space-y-3">
        {weeksWithContent.map((w) => (
          <WeekDetail key={w.weekIdx} week={w} locale={locale} />
        ))}
      </div>
    </section>
  );
}

function WeekCell({ week, hasContent }: { week: WeekBucket; hasContent: boolean }) {
  const Weather = week.weatherHint ? WEATHER_ICON[week.weatherHint] : null;
  const highRisk = week.risks.some((r) => r.risk === 'high');
  return (
    <div
      className={cn(
        'flex w-14 flex-shrink-0 flex-col items-center border-r py-2 text-[10px]',
        !hasContent && 'opacity-40',
      )}
    >
      <span className="font-mono text-muted-foreground">W{week.weekIdx}</span>
      <div className="mt-1 flex h-6 items-center gap-0.5">
        {week.activities.slice(0, 3).map((a, i) => {
          const Icon = ACTIVITY_ICON[a.activity] ?? ScissorsIcon;
          return <Icon key={i} className="h-3 w-3 text-primary" aria-hidden />;
        })}
        {week.activities.length > 3 && (
          <span className="text-[9px] text-muted-foreground">+{week.activities.length - 3}</span>
        )}
      </div>
      <div className="mt-0.5 flex items-center gap-0.5">
        {Weather && <Weather className="h-3 w-3 text-muted-foreground" aria-hidden />}
        {highRisk && (
          <span
            aria-hidden
            className="h-1.5 w-1.5 rounded-full bg-destructive"
          />
        )}
      </div>
    </div>
  );
}

function WeekDetail({ week, locale }: { week: WeekBucket; locale: Locale }) {
  const { t } = useTranslation();
  return (
    <details className="group rounded-xl border bg-card">
      <summary className="flex cursor-pointer list-none items-center justify-between gap-2 px-4 py-2.5 text-sm font-medium">
        <span className="flex items-center gap-2">
          <span className="font-mono text-xs text-muted-foreground">W{week.weekIdx}</span>
          <span>
            {week.activities
              .map((a) => pickLocalised(a.title, locale) ?? a.activity.replace(/_/g, ' '))
              .join(' · ')}
          </span>
        </span>
        <span className="flex items-center gap-1.5">
          {week.risks.slice(0, 3).map((r, i) => (
            <span
              key={i}
              className={cn(
                'rounded-full border px-1.5 py-0.5 text-[10px] capitalize',
                RISK_TONE[r.risk],
              )}
            >
              {r.risk}
            </span>
          ))}
        </span>
      </summary>
      <div className="space-y-3 border-t px-4 py-3">
        {week.activities.map((a, i) => (
          <ActivityRow key={`${a.activity}-${a.order_in_week ?? 0}-${i}`} activity={a} locale={locale} />
        ))}
        {week.risks.length > 0 && (
          <ul className="mt-2 space-y-1.5 border-t pt-2">
            {week.risks.map((r, i) => (
              <li key={i} className="flex flex-wrap items-start gap-2 text-xs">
                <span className={cn('rounded-full border px-2 py-0.5 capitalize', RISK_TONE[r.risk])}>
                  {r.risk}
                </span>
                <span className="text-foreground">
                  <strong>{r.disease_slug ?? r.pest_slug}</strong>
                  {pickLocalised(r.notes, locale) && (
                    <span className="text-muted-foreground"> — {pickLocalised(r.notes, locale)}</span>
                  )}
                </span>
              </li>
            ))}
          </ul>
        )}
        {week.activities.every((a) => !a.inputs || a.inputs.length === 0) &&
          week.risks.length === 0 && (
            <p className="text-xs text-muted-foreground">{t('plan.week_no_details')}</p>
          )}
      </div>
    </details>
  );
}

function ActivityRow({ activity, locale }: { activity: CultivationActivity; locale: Locale }) {
  const { t } = useTranslation();
  const Icon = ACTIVITY_ICON[activity.activity] ?? ScissorsIcon;
  const title = pickLocalised(activity.title, locale) ?? activity.activity.replace(/_/g, ' ');
  const body = pickLocalised(activity.body, locale);
  const dap = formatDap(activity.dap_min, activity.dap_max, t);
  return (
    <div>
      <div className="flex items-center gap-2 text-sm">
        <Icon className="h-4 w-4 text-primary" aria-hidden />
        <strong>{title}</strong>
        {dap && <span className="text-[11px] text-muted-foreground">· {dap}</span>}
      </div>
      {body && <p className="mt-0.5 pl-6 text-xs leading-relaxed text-muted-foreground">{body}</p>}
      {activity.inputs && activity.inputs.length > 0 && (
        <ul className="mt-1.5 flex flex-wrap gap-1.5 pl-6">
          {activity.inputs.map((input, i) => (
            <li key={i}>
              <InputPill input={input} locale={locale} />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function formatDap(min: number | undefined, max: number | undefined, t: (key: string, options?: Record<string, unknown>) => string): string | null {
  if (min == null && max == null) return null;
  const fmt = (n: number) => (n < 0 ? t('cultivation.dap_before', { n: -n }) : t('cultivation.dap_after', { n }));
  if (min != null && max != null && min !== max) return `${fmt(min)} → ${fmt(max)}`;
  return fmt(min ?? max!);
}

function InputPill({ input, locale }: { input: CultivationActivityInput; locale: Locale }) {
  const name = input.name ? pickLocalised(input.name, locale) : undefined;
  const parts: string[] = [];
  if (input.amount != null) {
    const unit = input.unit ? ` ${input.unit}` : '';
    const per = input.per_unit_area ? `/${input.per_unit_area.replace('_', ' ')}` : '';
    parts.push(`${input.amount}${unit}${per}`);
  }
  return (
    <span className="inline-flex items-center gap-1 rounded-md border bg-background px-2 py-0.5 text-[11px]">
      <span className="uppercase tracking-wide text-muted-foreground">{input.type}</span>
      {name && <span className="text-foreground">{name}</span>}
      {parts.length > 0 && <span className="text-muted-foreground">· {parts.join(' ')}</span>}
    </span>
  );
}

// ─── fertilizer schedule table ────────────────────────────────────────────

function FertilizerSchedule({
  activities,
  locale,
}: {
  activities: CultivationActivity[];
  locale: Locale;
}) {
  const { t } = useTranslation();
  const fertRows = activities.filter((a) =>
    a.activity === 'basal_fertilizer' || a.activity === 'top_dressing',
  );
  if (fertRows.length === 0) return null;

  return (
    <section aria-labelledby="fertilizer-heading" className="space-y-3">
      <h3
        id="fertilizer-heading"
        className="text-sm font-semibold uppercase tracking-wide text-muted-foreground"
      >
        {t('plan.fertilizer_schedule')}
      </h3>
      <div className="overflow-x-auto rounded-xl border bg-card">
        <table className="w-full text-sm">
          <thead className="border-b bg-muted/30 text-left text-xs uppercase tracking-wide text-muted-foreground">
            <tr>
              <th className="px-4 py-2">{t('plan.col_stage')}</th>
              <th className="px-4 py-2">{t('plan.col_week')}</th>
              <th className="px-4 py-2">{t('plan.col_dap')}</th>
              <th className="px-4 py-2">{t('plan.col_inputs')}</th>
            </tr>
          </thead>
          <tbody>
            {fertRows.map((a, i) => (
              <tr key={i} className="border-b last:border-0">
                <td className="px-4 py-2 capitalize">{a.activity.replace('_', ' ')}</td>
                <td className="px-4 py-2 font-mono text-xs">W{a.week_idx}</td>
                <td className="px-4 py-2 font-mono text-xs">
                  {formatDap(a.dap_min, a.dap_max, t) ?? '—'}
                </td>
                <td className="px-4 py-2">
                  <ul className="flex flex-wrap gap-1">
                    {(a.inputs ?? []).map((input, j) => (
                      <li key={j}>
                        <InputPill input={input} locale={locale} />
                      </li>
                    ))}
                  </ul>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

// ─── pest risk table ──────────────────────────────────────────────────────

function PestRiskTable({ risks, locale }: { risks: CultivationPestRisk[]; locale: Locale }) {
  const { t } = useTranslation();
  if (risks.length === 0) return null;
  return (
    <section aria-labelledby="pest-risk-heading" className="space-y-3">
      <h3
        id="pest-risk-heading"
        className="text-sm font-semibold uppercase tracking-wide text-muted-foreground"
      >
        {t('plan.pest_risks')}
      </h3>
      <div className="overflow-x-auto rounded-xl border bg-card">
        <table className="w-full text-sm">
          <thead className="border-b bg-muted/30 text-left text-xs uppercase tracking-wide text-muted-foreground">
            <tr>
              <th className="px-4 py-2">{t('plan.col_week')}</th>
              <th className="px-4 py-2">{t('plan.col_agent')}</th>
              <th className="px-4 py-2">{t('plan.col_risk')}</th>
              <th className="px-4 py-2">{t('plan.col_treatment')}</th>
            </tr>
          </thead>
          <tbody>
            {risks.map((r, i) => {
              const notes = pickLocalised(r.notes, locale);
              return (
                <tr key={i} className="border-b align-top last:border-0">
                  <td className="px-4 py-2 font-mono text-xs">W{r.week_idx}</td>
                  <td className="px-4 py-2 font-mono text-xs">
                    {r.disease_slug ?? r.pest_slug}
                  </td>
                  <td className="px-4 py-2">
                    <span className={cn('rounded-full border px-2 py-0.5 text-xs capitalize', RISK_TONE[r.risk])}>
                      {r.risk}
                    </span>
                  </td>
                  <td className="px-4 py-2 text-xs text-muted-foreground">{notes ?? '—'}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
}

// ─── economics ────────────────────────────────────────────────────────────

function EconomicsPanel({
  economics,
  locale,
}: {
  economics: CultivationEconomics;
  locale: Locale;
}) {
  const { t } = useTranslation();
  const money = (v: number | undefined) =>
    v == null
      ? '—'
      : new Intl.NumberFormat('en-LK', { maximumFractionDigits: 0 }).format(v);
  return (
    <section aria-labelledby="economics-heading" className="space-y-3">
      <header className="flex items-center gap-2">
        <Coins className="h-5 w-5 text-primary" aria-hidden />
        <h3 id="economics-heading" className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
          {t('plan.economics')} · {economics.reference_year} · {t('plan.per_unit_area')} {economics.unit_area}
        </h3>
      </header>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <Kpi label={t('plan.gross_revenue')} value={money(economics.gross_revenue)} currency={economics.currency} />
        <Kpi
          label={t('plan.net_with_family_labour')}
          value={money(economics.net_revenue_with_family_labour)}
          currency={economics.currency}
          accent
        />
        <Kpi
          label={t('plan.net_without_family_labour')}
          value={money(economics.net_revenue_without_family_labour)}
          currency={economics.currency}
        />
      </div>
      {economics.cost_lines && economics.cost_lines.length > 0 && (
        <div className="overflow-x-auto rounded-xl border bg-card">
          <table className="w-full text-sm">
            <thead className="border-b bg-muted/30 text-left text-xs uppercase tracking-wide text-muted-foreground">
              <tr>
                <th className="px-4 py-2">{t('plan.col_category')}</th>
                <th className="px-4 py-2">{t('plan.col_label')}</th>
                <th className="px-4 py-2 text-right">{t('plan.col_amount')}</th>
              </tr>
            </thead>
            <tbody>
              {economics.cost_lines.map((line, i) => (
                <tr key={i} className="border-b last:border-0">
                  <td className="px-4 py-2 capitalize">{line.category.replace('_', ' ')}</td>
                  <td className="px-4 py-2 text-muted-foreground">
                    {pickLocalised(line.label, locale) ?? <Fragment />}
                    {line.notes && <span className="block text-[11px] opacity-70">{line.notes}</span>}
                  </td>
                  <td className="px-4 py-2 text-right font-mono tabular-nums">
                    {money(line.amount)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function Kpi({
  label,
  value,
  currency,
  accent,
}: {
  label: string;
  value: string;
  currency: string;
  accent?: boolean;
}) {
  return (
    <div
      className={cn(
        'rounded-xl border p-4',
        accent ? 'border-primary/40 bg-primary/5' : 'bg-card',
      )}
    >
      <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className="mt-1 text-xl font-semibold tabular-nums">
        {value} <span className="text-xs text-muted-foreground">{currency}</span>
      </dd>
    </div>
  );
}
