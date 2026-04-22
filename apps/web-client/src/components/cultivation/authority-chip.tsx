import { useTranslation } from 'react-i18next';
import { AlertCircle, BookOpen, CheckCircle2, Lightbulb, Sparkles, UserCheck } from 'lucide-react';

import type { AuthorityLevel } from '@/lib/api';
import { cn } from '@/lib/utils';

/**
 * AuthorityChip renders an honesty-first badge alongside any agronomic
 * claim. Colour + icon makes the distinction between a DOA-validated
 * recommendation and a cross-regional analogy instantly scannable.
 *
 * Only `doa_official` and `peer_reviewed` render in the primary colour;
 * every lower-authority band gets a warning-tone treatment so farmers
 * know to treat them as advisory, not official dose.
 */
export function AuthorityChip({
  authority,
  size = 'sm',
  className,
}: {
  authority: AuthorityLevel;
  size?: 'sm' | 'md';
  className?: string;
}) {
  const { t } = useTranslation();
  const meta = AUTHORITY_META[authority];
  const Icon = meta.icon;
  const padding = size === 'md' ? 'px-2.5 py-1 text-xs' : 'px-2 py-0.5 text-[11px]';
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full border font-medium',
        padding,
        meta.classes,
        className,
      )}
      title={t(`authority.${authority}_tooltip`)}
    >
      <Icon className={size === 'md' ? 'h-3.5 w-3.5' : 'h-3 w-3'} aria-hidden />
      {t(`authority.${authority}`)}
    </span>
  );
}

const AUTHORITY_META: Record<
  AuthorityLevel,
  { icon: typeof CheckCircle2; classes: string }
> = {
  doa_official: {
    icon: CheckCircle2,
    classes: 'bg-primary/10 border-primary/30 text-primary',
  },
  peer_reviewed: {
    icon: BookOpen,
    classes: 'bg-primary/10 border-primary/30 text-primary',
  },
  regional_authority: {
    icon: UserCheck,
    classes: 'bg-amber-500/10 border-amber-500/30 text-amber-700 dark:text-amber-400',
  },
  practitioner_report: {
    icon: Lightbulb,
    classes: 'bg-amber-500/10 border-amber-500/30 text-amber-700 dark:text-amber-400',
  },
  inferred_by_analogy: {
    icon: AlertCircle,
    classes: 'bg-amber-500/10 border-amber-500/30 text-amber-700 dark:text-amber-400',
  },
  agent_synthesis: {
    icon: Sparkles,
    classes: 'bg-muted border-muted-foreground/30 text-muted-foreground',
  },
};
