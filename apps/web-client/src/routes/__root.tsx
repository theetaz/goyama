import { Link, Outlet, createRootRouteWithContext } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/router-devtools';
import type { QueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Bug, FlaskConical, Leaf, Map as MapIcon, ShieldAlert } from 'lucide-react';

import { supportedLocales, setLocale, type Locale } from '@/i18n';

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  component: RootLayout,
});

function RootLayout() {
  const { t, i18n } = useTranslation();
  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="border-b bg-card">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
          <Link to="/" className="flex items-center gap-2 text-primary">
            <Leaf className="h-6 w-6" aria-hidden />
            <span className="text-lg font-semibold">{t('app.name')}</span>
          </Link>
          <nav className="flex items-center gap-2 text-sm">
            <Link
              to="/"
              activeProps={{ className: 'font-semibold text-primary' }}
              activeOptions={{ exact: true }}
              className="rounded-md px-3 py-2 hover:bg-muted"
            >
              {t('nav.explore')}
            </Link>
            <Link
              to="/map"
              activeProps={{ className: 'font-semibold text-primary' }}
              className="flex items-center gap-1 rounded-md px-3 py-2 hover:bg-muted"
            >
              <MapIcon className="h-4 w-4" aria-hidden />
              {t('nav.map')}
            </Link>
            <Link
              to="/diseases"
              activeProps={{ className: 'font-semibold text-primary' }}
              className="flex items-center gap-1 rounded-md px-3 py-2 hover:bg-muted"
            >
              <ShieldAlert className="h-4 w-4" aria-hidden />
              {t('nav.diseases')}
            </Link>
            <Link
              to="/pests"
              activeProps={{ className: 'font-semibold text-primary' }}
              className="flex items-center gap-1 rounded-md px-3 py-2 hover:bg-muted"
            >
              <Bug className="h-4 w-4" aria-hidden />
              {t('nav.pests')}
            </Link>
            <Link
              to="/remedies"
              activeProps={{ className: 'font-semibold text-primary' }}
              className="flex items-center gap-1 rounded-md px-3 py-2 hover:bg-muted"
            >
              <FlaskConical className="h-4 w-4" aria-hidden />
              {t('nav.remedies')}
            </Link>
            <LocalePicker current={i18n.language as Locale} />
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-6">
        <Outlet />
      </main>
      <footer className="mt-12 border-t py-6 text-center text-xs text-muted-foreground">
        <span>{t('app.tagline')}</span>
      </footer>
      <TanStackRouterDevtools position="bottom-right" />
    </div>
  );
}

function LocalePicker({ current }: { current: Locale }) {
  const { t } = useTranslation();
  return (
    <label className="flex items-center gap-2 text-sm">
      <span className="sr-only">{t('locale_picker.label')}</span>
      <select
        value={current}
        onChange={(e) => setLocale(e.target.value as Locale)}
        className="rounded-md border bg-background px-2 py-1"
      >
        {supportedLocales.map((l) => (
          <option key={l.code} value={l.code}>
            {l.native}
          </option>
        ))}
      </select>
    </label>
  );
}
