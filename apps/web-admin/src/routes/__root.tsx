import { Link, Outlet, createRootRouteWithContext } from '@tanstack/react-router';
import type { QueryClient } from '@tanstack/react-query';
import { LayoutDashboard, Leaf } from 'lucide-react';

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  component: AdminLayout,
});

// Only routes that exist today. Add Diseases / Pests / Translations / Users as
// their route files land under src/routes/.
const nav = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard, exact: true },
  { to: '/crops', label: 'Crops', icon: Leaf, exact: false },
] as const;

function AdminLayout() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="grid grid-cols-[220px_1fr]">
        <aside className="sticky top-0 h-screen border-r bg-card">
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
        </aside>
        <main className="p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
