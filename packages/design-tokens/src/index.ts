/**
 * JS/TS export of the CropDoc design tokens.
 * apps/mobile consumes these via NativeWind; web apps import the CSS directly.
 */

export const tokens = {
  colors: {
    primary: 'oklch(0.55 0.16 142)',
    primaryForeground: 'oklch(0.99 0.01 142)',
    accent: 'oklch(0.75 0.15 80)',
    accentForeground: 'oklch(0.22 0.05 80)',
    destructive: 'oklch(0.60 0.22 27)',
    success: 'oklch(0.65 0.15 155)',
    warning: 'oklch(0.78 0.15 70)',
    background: 'oklch(0.98 0.005 100)',
    foreground: 'oklch(0.22 0.02 260)',
    muted: 'oklch(0.95 0.01 100)',
    mutedForeground: 'oklch(0.50 0.02 260)',
    border: 'oklch(0.90 0.01 100)',
    card: 'oklch(1.00 0 0)',
    soil: 'oklch(0.35 0.06 45)',
  },
  radius: {
    sm: '0.5rem',
    md: '0.75rem',
    lg: '1rem',
  },
  motion: {
    micro: '150ms',
    element: '250ms',
    route: '400ms',
    easeStandard: 'cubic-bezier(0.22, 1, 0.36, 1)',
  },
  typography: {
    fontFamilySans: 'Inter, "Noto Sans Sinhala", "Noto Sans Tamil", system-ui, sans-serif',
    fontSizes: {
      xs: '0.75rem',
      sm: '0.875rem',
      base: '1rem',
      lg: '1.125rem',
      xl: '1.375rem',
      '2xl': '1.75rem',
      '3xl': '2.25rem',
    },
  },
  /** Agro-ecological zone colours for map overlays. */
  aez: {
    wet: 'oklch(0.60 0.14 150)',
    intermediate: 'oklch(0.75 0.12 95)',
    dry: 'oklch(0.70 0.12 60)',
  },
} as const;

export type Tokens = typeof tokens;
