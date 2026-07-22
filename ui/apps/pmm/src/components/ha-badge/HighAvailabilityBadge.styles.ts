import { ChipProps } from '@mui/material/Chip';
import { SxProps, Theme } from '@mui/material/styles';
import { HAHealth } from 'types/ha.types';

// Colours come from the theme's WCAG-AA `MuiChip` soft variants via the `color`
// prop — hard-coding semantic palette values here (e.g. `warning.main` as
// outlined-chip text) failed AA on light.
export const HEALTH_CHIP_COLOR: Record<HAHealth, ChipProps['color']> = {
  healthy: 'default',
  degraded: 'warning',
  critical: 'error',
  unreachable: 'error',
  unknown: 'default',
};

// Per-state emphasis sx (the `color` above already supplies the AA soft tint).
// critical/unreachable additionally get a solid same-hue border + heavier weight
// so an at-risk cluster stays noticeable at a glance in the sidebar, without a
// non-AA solid fill.
export const HEALTH_EMPHASIS: Record<HAHealth, SxProps<Theme>> = {
  healthy: { fontWeight: 400 },
  degraded: { fontWeight: 500 },
  critical: { fontWeight: 600, border: '1px solid', borderColor: 'error.main' },
  unreachable: {
    fontWeight: 600,
    border: '1px solid',
    borderColor: 'error.main',
  },
  unknown: { fontWeight: 400 },
};
