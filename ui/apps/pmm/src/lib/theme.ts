import type { PaletteMode, ThemeOptions } from '@mui/material';
import { pmmThemeOptions } from '@percona/percona-ui';

/**
 * Postgres1st brand theme.
 *
 * Layers a Postgres1st palette on top of the `@percona/percona-ui` design
 * system (no fork): we take the package's `pmmThemeOptions(mode)` and override
 * only the brand accent slots — `primary` (purple) and `secondary` (magenta),
 * drawn from the logo (purple / magenta / blue / amber).
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
          ? { main: '#6B2FB3', light: '#9387FE', dark: '#421CBB', contrastText: '#FFFFFF' }
          : { main: '#B6B2FF', light: '#D6D5FF', dark: '#9387FE', contrastText: '#000000' }),
      },
      secondary: {
        ...base.palette?.secondary,
        ...(isLight
          ? { main: '#D6338A', light: '#F062A8', dark: '#A61E67', contrastText: '#FFFFFF' }
          : { main: '#F062A8', light: '#F894C4', dark: '#D6338A', contrastText: '#000000' }),
      },
    },
  };
};
