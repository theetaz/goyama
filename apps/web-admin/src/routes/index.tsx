import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';

import { api } from '@/lib/api';

export const Route = createFileRoute('/')({
  component: Dashboard,
});

function Dashboard() {
  const health = useQuery({ queryKey: ['health'], queryFn: () => api.health() });
  const crops = useQuery({ queryKey: ['crops-count'], queryFn: () => api.listCrops({ limit: 1 }) });

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold">Dashboard</h1>
        <p className="text-sm text-muted-foreground">
          Corpus coverage + API health snapshot. Row 1 of the full review-queue UI.
        </p>
      </header>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <Stat
          label="API service"
          value={health.data?.status ?? (health.isLoading ? '…' : 'down')}
          sub={health.data?.version ? `v${health.data.version}` : ''}
          intent={health.data?.status === 'ok' ? 'ok' : 'warn'}
        />
        <Stat
          label="Uptime (s)"
          value={health.data ? String(health.data.uptime_sec) : '—'}
        />
        <Stat
          label="Crops in corpus"
          value={crops.data ? String(crops.data.count) : '—'}
          sub="status=draft"
        />
      </div>

      <section className="rounded-lg border bg-card p-5">
        <h2 className="text-base font-semibold">Next up</h2>
        <ul className="mt-3 space-y-1 text-sm text-muted-foreground">
          <li>· Build record-review screens for crops/diseases/pests.</li>
          <li>· Connect OTP / TOTP auth for staff roles.</li>
          <li>· Hook up the translations console.</li>
          <li>· Wire the advisory broadcaster.</li>
        </ul>
      </section>
    </div>
  );
}

function Stat({
  label,
  value,
  sub,
  intent,
}: {
  label: string;
  value: string;
  sub?: string;
  intent?: 'ok' | 'warn';
}) {
  return (
    <div className="rounded-lg border bg-card p-5">
      <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </div>
      <div
        className={`mt-2 text-3xl font-semibold ${intent === 'warn' ? 'text-destructive' : ''}`}
      >
        {value}
      </div>
      {sub && <div className="mt-1 text-xs text-muted-foreground">{sub}</div>}
    </div>
  );
}
