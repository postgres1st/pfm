import { ChipProps } from '@mui/material/Chip';
import { HAHealth } from 'types/ha.types';

// Colours come from the theme's WCAG-AA `MuiChip` soft variants via the `color`
// prop — hard-coding semantic palette values here (e.g. `warning.main` as
// outlined-chip text) failed AA on light. Emphasis for the worst states is
// conveyed by font weight instead of a louder (and less readable) fill.
export const HEALTH_CHIP_COLOR: Record<HAHealth, ChipProps['color']> = {
  healthy: 'default',
  degraded: 'warning',
  critical: 'error',
  down: 'error',
};

export const HEALTH_FONT_WEIGHT: Record<HAHealth, number> = {
  healthy: 400,
  degraded: 500,
  critical: 600,
  down: 600,
};
