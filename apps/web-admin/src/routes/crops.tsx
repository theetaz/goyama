import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';

import { api } from '@/lib/api';

export const Route = createFileRoute('/crops')({
  component: CropsTable,
});

function CropsTable() {
  const [q, setQ] = useState('');
  const crops = useQuery({
    queryKey: ['admin-crops', q],
    queryFn: () => api.listCrops({ q: q || undefined, limit: 200 }),
  });

  return (
    <div className="space-y-4">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Crops</h1>
          <p className="text-sm text-muted-foreground">
            {crops.data?.count ?? 0} records in the v0.0.1-drafts corpus.
          </p>
        </div>
        <input
          type="search"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Filter by slug, name, or alias"
          className="w-64 rounded-md border bg-background px-3 py-2 text-sm"
        />
      </header>

      <div className="overflow-x-auto rounded-lg border bg-card">
        <table className="w-full text-sm">
          <thead className="border-b bg-muted/50 text-left text-xs uppercase tracking-wide text-muted-foreground">
            <tr>
              <th className="px-4 py-2">Slug</th>
              <th className="px-4 py-2">Scientific name</th>
              <th className="px-4 py-2">Category</th>
              <th className="px-4 py-2">Names</th>
            </tr>
          </thead>
          <tbody>
            {crops.isLoading && (
              <tr>
                <td colSpan={4} className="px-4 py-6 text-center text-muted-foreground">
                  Loading…
                </td>
              </tr>
            )}
            {crops.isError && (
              <tr>
                <td colSpan={4} className="px-4 py-6 text-center text-destructive">
                  API unreachable. Start the Go service on :8080.
                </td>
              </tr>
            )}
            {crops.data?.items.map((c) => (
              <tr key={c.slug} className="border-b last:border-0 hover:bg-muted/30">
                <td className="px-4 py-2 font-mono text-xs">{c.slug}</td>
                <td className="px-4 py-2 italic text-muted-foreground">{c.scientific_name ?? '—'}</td>
                <td className="px-4 py-2">
                  {c.category && (
                    <span className="rounded-full bg-muted px-2 py-0.5 text-xs capitalize">
                      {c.category.replace(/_/g, ' ')}
                    </span>
                  )}
                </td>
                <td className="px-4 py-2 text-xs">
                  {c.names &&
                    Object.entries(c.names)
                      .map(([k, v]) => `${k}:${v}`)
                      .join(' · ')}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
