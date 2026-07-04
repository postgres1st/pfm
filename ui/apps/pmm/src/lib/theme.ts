import type { PaletteMode, ThemeOptions } from '@mui/material';
import { pmmThemeOptions } from '@percona/percona-ui';

/**
 * Postgres1st brand theme.
 *
 * Layers a Postgres1st palette on top of the `@percona/percona-ui` design
 * system (no fork): we take the package's `pmmThemeOptions(mode)` and override
 * only the brand accent slots — `primary` (periwinkle purple) and `secondary`
 * (amber), drawn from the logo.
 *
 * Semantic colors (error / warning / success / info) are intentionally left as
 * the design-system defaults so their UX meaning is preserved. Non-standard
 * PaletteColor extras (hover/selected/focus) are spread through from the base.
 *
 * This recolors the PFMM shell only — Grafana dashboards keep Grafana's palette
 * (the shell↔Grafana sync carries light/dark mode, not brand colors).
 */
export const postgres1stThemeOptions = (mode: PaletteMode): ThemeOptions => {
  const base = pmmThemeOptions(mode);
  const isLight = mode === 'light';

  return {
    ...base,
    palette: {
      ...base.palette,
      primary: {
        ...base.palette?.primary,
        ...(isLight
          ? { main: '#653DF4', light: '#9B81F8', dark: '#472BAB', contrastText: '#FFFFFF' }
          : { main: '#B6B2FF', light: '#CCC9FF', dark: '#8986BF', contrastText: '#000000' }),
      },
      secondary: {
        ...base.palette?.secondary,
        ...(isLight
          ? { main: '#F5A623', light: '#F8C165', dark: '#AC7419', contrastText: '#000000' }
          : { main: '#F5B94D', light: '#F8CE82', dark: '#AC8236', contrastText: '#000000' }),
      },
    },
  };
};
