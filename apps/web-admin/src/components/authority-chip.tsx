import { AlertCircle, BookOpen, CheckCircle2, Lightbulb, Sparkles, UserCheck } from 'lucide-react';

import type { AuthorityLevel } from '@/lib/api';

const LABEL: Record<AuthorityLevel, string> = {
  doa_official: 'DOA official',
  peer_reviewed: 'Peer-reviewed',
  regional_authority: 'Regional authority',
  practitioner_report: 'Practitioner report',
  inferred_by_analogy: 'Promising practice',
  agent_synthesis: 'Agent synthesis',
};

const META: Record<AuthorityLevel, { icon: typeof CheckCircle2; tone: string }> = {
  doa_official: { icon: CheckCircle2, tone: 'bg-primary/10 border-primary/30 text-primary' },
  peer_reviewed: { icon: BookOpen, tone: 'bg-primary/10 border-primary/30 text-primary' },
  regional_authority: {
    icon: UserCheck,
    tone: 'bg-amber-500/10 border-amber-500/30 text-amber-700 dark:text-amber-400',
  },
  practitioner_report: {
    icon: Lightbulb,
    tone: 'bg-amber-500/10 border-amber-500/30 text-amber-700 dark:text-amber-400',
  },
  inferred_by_analogy: {
    icon: AlertCircle,
    tone: 'bg-amber-500/10 border-amber-500/30 text-amber-700 dark:text-amber-400',
  },
  agent_synthesis: {
    icon: Sparkles,
    tone: 'bg-muted border-muted-foreground/30 text-muted-foreground',
  },
};

/**
 * AuthorityChip is the admin-side mirror of the web-client chip.
 * Same six-band colour mapping; different styling primitives so the
 * admin UI is always honest about which records may drive farmer
 * recommendations vs. which should remain advisory.
 */
export function AuthorityChip({ authority }: { authority: AuthorityLevel }) {
  const meta = META[authority];
  const Icon = meta.icon;
  return (
    <span
      className={
        'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium ' +
        meta.tone
      }
    >
      <Icon className="h-3 w-3" aria-hidden />
      {LABEL[authority]}
    </span>
  );
}
