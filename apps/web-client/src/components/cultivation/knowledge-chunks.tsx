import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { ExternalLink, FileText, Sparkles } from 'lucide-react';

import type {
  AuthorityLevel,
  KnowledgeChunk,
  KnowledgeResponse,
  KnowledgeSource,
} from '@/lib/api';

import { AuthorityChip } from './authority-chip';

/**
 * KnowledgeChunks lists every chunk for an entity grouped by authority
 * band, so DOA-official guidance always renders above cross-regional
 * analogies. Each chunk carries its own authority chip in case the
 * reader scans out of order.
 */
export function KnowledgeChunks({ data }: { data: KnowledgeResponse | undefined }) {
  const { t } = useTranslation();

  const groups = useMemo(() => groupByAuthority(data?.chunks ?? []), [data]);
  const sourceBySlug = useMemo(() => {
    const m = new Map<string, KnowledgeSource>();
    for (const s of data?.sources ?? []) m.set(s.slug, s);
    return m;
  }, [data]);

  if (!data || data.chunks.length === 0) {
    return null;
  }

  return (
    <section aria-labelledby="knowledge-heading" className="space-y-4">
      <header className="flex items-center gap-2">
        <Sparkles className="h-5 w-5 text-primary" aria-hidden />
        <h2 id="knowledge-heading" className="text-xl font-semibold">
          {t('knowledge.heading')}
        </h2>
      </header>
      <p className="text-xs text-muted-foreground">{t('knowledge.subhead')}</p>

      <div className="space-y-5">
        {AUTHORITY_ORDER.map((authority) => {
          const chunks = groups.get(authority);
          if (!chunks || chunks.length === 0) return null;
          return (
            <div key={authority} className="space-y-2">
              <div className="flex items-center gap-2">
                <AuthorityChip authority={authority} size="md" />
                <span className="text-xs text-muted-foreground">
                  {chunks.length}
                  {chunks.length === 1 ? ' ' + t('knowledge.note_singular') : ' ' + t('knowledge.note_plural')}
                </span>
              </div>
              <ul className="space-y-2">
                {chunks.map((c) => (
                  <ChunkCard key={c.slug} chunk={c} source={sourceBySlug.get(c.source_slug)} />
                ))}
              </ul>
            </div>
          );
        })}
      </div>
    </section>
  );
}

function ChunkCard({
  chunk,
  source,
}: {
  chunk: KnowledgeChunk;
  source: KnowledgeSource | undefined;
}) {
  const { t } = useTranslation();
  return (
    <li className="rounded-xl border bg-card p-4">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          {chunk.title && <h3 className="font-semibold leading-tight">{chunk.title}</h3>}
          {source && (
            <p className="mt-0.5 inline-flex items-center gap-1 text-xs text-muted-foreground">
              <FileText className="h-3 w-3" aria-hidden />
              {source.display_name}
              {source.publisher && source.publisher !== source.display_name && (
                <span> · {source.publisher}</span>
              )}
            </p>
          )}
        </div>
        {chunk.confidence != null && (
          <span className="rounded-full border bg-background px-2 py-0.5 text-[10px] uppercase tracking-wide text-muted-foreground">
            {t('knowledge.confidence')} {(chunk.confidence * 100).toFixed(0)}%
          </span>
        )}
      </div>

      <p className="mt-2 whitespace-pre-line text-sm leading-relaxed text-foreground">
        {chunk.body}
      </p>

      {chunk.quote && (
        <blockquote className="mt-3 border-l-2 border-muted pl-3 text-xs italic text-muted-foreground">
          "{chunk.quote}"
        </blockquote>
      )}

      <div className="mt-3 flex flex-wrap items-center gap-2 text-[11px]">
        {chunk.topic_tags?.slice(0, 5).map((tag) => (
          <span key={tag} className="rounded bg-muted px-1.5 py-0.5 text-muted-foreground">
            #{tag}
          </span>
        ))}
        {chunk.applies_to_countries && chunk.applies_to_countries.length > 0 && (
          <span className="text-muted-foreground">
            {t('knowledge.applies_to')}: {chunk.applies_to_countries.join(', ')}
          </span>
        )}
        {source?.url && (
          <a
            href={source.url}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1 text-primary hover:underline"
          >
            {t('knowledge.view_source')}
            <ExternalLink className="h-3 w-3" aria-hidden />
          </a>
        )}
      </div>
    </li>
  );
}

// Order authoritative bands above advisory ones so the reader sees the
// DOA guidance first, regardless of how the backend ordered the items.
const AUTHORITY_ORDER: AuthorityLevel[] = [
  'doa_official',
  'peer_reviewed',
  'regional_authority',
  'practitioner_report',
  'inferred_by_analogy',
  'agent_synthesis',
];

function groupByAuthority(chunks: KnowledgeChunk[]) {
  const groups = new Map<AuthorityLevel, KnowledgeChunk[]>();
  for (const c of chunks) {
    const arr = groups.get(c.authority) ?? [];
    arr.push(c);
    groups.set(c.authority, arr);
  }
  return groups;
}
