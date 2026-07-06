import type { PaletteMode, ThemeOptions } from '@mui/material';
import { alpha } from '@mui/material/styles';
import { deepmerge } from '@mui/utils';
import { pmmThemeOptions } from '@percona/percona-ui';

/**
 * Postgres1st brand theme.
 *
 * Layers a Postgres1st brand on top of the `@percona/percona-ui` design system
 * (no fork): we take `pmmThemeOptions(mode)` and deep-merge our overrides so the
 * base's component styles survive.
 *
 * Overrides:
 *  - `palette.primary` — periwinkle purple; `palette.secondary` — amber (logo).
 *    Semantic colors (error/warning/success/info) keep their meaning.
 *  - `MuiLink` — the base hard-codes links to its own `text.accent1` blue; we
 *    point them at `primary.main` so links follow the brand in both modes.
 *  - `MuiChip` status variants — the base renders success/warning/info/error
 *    chips with unreadable low-contrast text (e.g. bright green ~2.3:1). We add
 *    soft tinted badges (pale same-hue fill + a dark/light readable same-hue
 *    text) for those four colors, per WCAG AA. Brand chips (primary/secondary)
 *    are untouched and stay solid.
 *
 * Recolors the PFMM shell only — Grafana dashboards keep Grafana's palette.
 */

// Readable soft-badge colors per status color, per mode: [color, tint-hue, text].
const STATUS_SOFT = {
  light: [
    ['success', '#1B7A2F', '#12561F'],
    ['warning', '#B7791F', '#6E4A0C'],
    ['info', '#0073FF', '#08428F'],
    ['error', '#B42B21', '#7C2620'],
  ],
  dark: [
    ['success', '#3FB85C', '#BEEDCB'],
    ['warning', '#F5B94D', '#F8DCA1'],
    ['info', '#479DFF', '#BBD9FF'],
    ['error', '#EA5449', '#F8C0BB'],
  ],
} as const;

export const postgres1stThemeOptions = (mode: PaletteMode): ThemeOptions => {
  const base = pmmThemeOptions(mode);
  const isLight = mode === 'light';
  const primaryMain = isLight ? '#653DF4' : '#B6B2FF';

  const chipVariants = STATUS_SOFT[mode].map(([color, hue, text]) => ({
    props: { color: color as 'success' | 'warning' | 'info' | 'error' },
    style: {
      backgroundColor: alpha(hue, isLight ? 0.14 : 0.26),
      borderColor: 'transparent',
      color: text,
      '& .MuiChip-label': { color: text },
      '& .MuiChip-icon': { color: text },
      '& .MuiChip-deleteIcon': { color: text, opacity: 0.7 },
    },
  }));

  return deepmerge(base, {
    palette: {
      primary: isLight
        ? { main: primaryMain, light: '#9B81F8', dark: '#472BAB', contrastText: '#FFFFFF' }
        : { main: primaryMain, light: '#CCC9FF', dark: '#8986BF', contrastText: '#000000' },
      secondary: {
        main: '#F5B94D',
        light: '#F8CE82',
        dark: '#AC8236',
        contrastText: '#000000',
      },
    },
    components: {
      MuiLink: {
        styleOverrides: {
          root: { color: primaryMain, textDecorationColor: primaryMain },
        },
      },
      MuiChip: {
        variants: chipVariants,
      },
    },
  } satisfies ThemeOptions);
};
