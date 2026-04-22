import { useState } from 'react';
import { Link, Outlet, createRootRouteWithContext } from '@tanstack/react-router';
import type { QueryClient } from '@tanstack/react-query';
import { Bug, CalendarRange, ClipboardCheck, FlaskConical, LayoutDashboard, Leaf, Shell, Sparkles, UserCircle2 } from 'lucide-react';

import { getReviewer, setReviewer } from '@/lib/api';

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  component: AdminLayout,
});

// Only routes that exist today. Add Diseases / Pests / Translations / Users as
// their route files land under src/routes/.
const nav = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard, exact: true },
  { to: '/crops', label: 'Crops', icon: Leaf, exact: false },
  { to: '/review', label: 'Cultivation review', icon: ClipboardCheck, exact: false },
  { to: '/review-diseases', label: 'Disease review', icon: Bug, exact: false },
  { to: '/review-pests', label: 'Pest review', icon: Shell, exact: false },
  { to: '/review-remedies', label: 'Remedy review', icon: FlaskConical, exact: false },
  { to: '/review-plans', label: 'Plan review', icon: CalendarRange, exact: false },
  { to: '/review-knowledge', label: 'Knowledge review', icon: Sparkles, exact: false },
] as const;

function AdminLayout() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="grid grid-cols-[220px_1fr]">
        <aside className="sticky top-0 flex h-screen flex-col border-r bg-card">
          <div className="px-4 py-5">
            <div className="text-sm font-semibold">Goyama Admin</div>
            <div className="text-xs text-muted-foreground">internal tooling</div>
          </div>
          <nav className="flex flex-col gap-0.5 px-2 text-sm">
            {nav.map((item) => (
              <Link
                key={item.to}
                to={item.to}
                activeProps={{ className: 'bg-primary/10 text-primary' }}
                activeOptions={{ exact: item.exact }}
                className="flex items-center gap-2 rounded-md px-3 py-2 hover:bg-muted"
              >
                <item.icon className="h-4 w-4" aria-hidden />
                <span>{item.label}</span>
              </Link>
            ))}
          </nav>
          <ReviewerCard />
        </aside>
        <main className="p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}

function ReviewerCard() {
  const [value, setValue] = useState<string>(getReviewer());
  function commit(next: string) {
    setReviewer(next);
    setValue(next);
  }
  return (
    <div className="mt-auto border-t px-4 py-4 text-xs">
      <label className="flex items-center gap-1.5 font-medium text-muted-foreground">
        <UserCircle2 className="h-3.5 w-3.5" aria-hidden />
        Reviewer identity
      </label>
      <input
        type="email"
        placeholder="you@goyama.lk"
        value={value}
        onChange={(e) => commit(e.target.value)}
        className="mt-1 w-full rounded-md border bg-background px-2 py-1"
      />
      <p className="mt-1.5 text-[11px] leading-snug text-muted-foreground">
        Stamped onto every status change as <code>X-Goyama-Reviewer</code>. Placeholder
        until staff SSO lands.
      </p>
    </div>
  );
}
