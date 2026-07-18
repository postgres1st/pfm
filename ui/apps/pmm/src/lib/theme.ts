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
  const primaryMain = isLight ? '#0E7ABE' : '#5EAEE0';

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
      // Light-mode muted text is only ~3.5:1 (fails AA); darken to ~5:1 (AA)
      // while staying visibly secondary. Dark mode is already AA/AAA.
      ...(isLight
        ? { text: { secondary: 'rgba(40, 39, 39, 0.7)' } }
        : // The base dark theme puts background.paper (#282727) DARKER than
          // background.default (#3D3C3C), so cards look sunken — the reverse of
          // light (paper #FFF above default #F6F5F5). Darken the page below the
          // paper so Cards/Papers read as raised, consistent with light.
          { background: { default: '#1B1A1A' } }),
      primary: isLight
        ? { main: primaryMain, light: '#62A9D5', dark: '#0A5585', contrastText: '#FFFFFF' }
        : { main: primaryMain, light: '#8EC6E9', dark: '#4682A8', contrastText: '#000000' },
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
      // The base alert paints from palette[severity].surface + contrastText,
      // which resolves to loud SOLID fills in dark (vs soft tints in light).
      // Render soft-tinted alerts consistently in both modes: a same-hue tint
      // background with readable same-hue text (dark shade on light / light
      // shade on dark) — matching the status chips.
      MuiAlert: {
        styleOverrides: {
          root: // eslint-disable-next-line @typescript-eslint/no-explicit-any
          ({ theme, ownerState }: any) => {
            const sev = ownerState.color ?? ownerState.severity ?? 'info';
            const pal = theme.palette[sev] ?? theme.palette.info;
            const light = theme.palette.mode === 'light';
            const text = light ? pal.dark : pal.light;
            return {
              ...theme.typography.body1,
              minHeight: 40,
              padding: '6px 12px',
              alignItems: 'center',
              borderRadius: 6,
              color: text,
              backgroundColor: alpha(pal.main, light ? 0.12 : 0.22),
              border: `1px solid ${alpha(pal.main, light ? 0.28 : 0.42)}`,
            };
          },
          icon: // eslint-disable-next-line @typescript-eslint/no-explicit-any
          ({ theme, ownerState }: any) => {
            const sev = ownerState.color ?? ownerState.severity ?? 'info';
            const pal = theme.palette[sev] ?? theme.palette.info;
            const light = theme.palette.mode === 'light';
            return {
              color: `${light ? pal.dark : pal.light} !important`,
              marginRight: 10,
              padding: '7px 0',
            };
          },
          // The base `message` slot colors text with palette[severity].contrastText,
          // which for dark warning is a dark brown (#493408) — unreadable on the
          // soft tint. Use the same readable same-hue shade as root/icon.
          message: // eslint-disable-next-line @typescript-eslint/no-explicit-any
          ({ theme, ownerState }: any) => {
            const sev = ownerState.color ?? ownerState.severity ?? 'info';
            const pal = theme.palette[sev] ?? theme.palette.info;
            const light = theme.palette.mode === 'light';
            const text = light ? pal.dark : pal.light;
            return {
              color: text,
              padding: '7px 0',
              '& .MuiLink-root': {
                color: 'inherit',
                textDecorationColor: 'inherit',
                '&:focus-visible': { outlineColor: 'currentColor' },
              },
            };
          },
        },
      },
    },
  } satisfies ThemeOptions);
};
