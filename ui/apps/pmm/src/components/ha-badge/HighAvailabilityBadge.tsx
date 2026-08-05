import Chip from '@mui/material/Chip';
import { FC } from 'react';
import { HIGH_AVAILABILITY_BADGE_HEALTH } from './HighAvailabilityBadge.constants';
import {
  HEALTH_CHIP_COLOR,
  HEALTH_EMPHASIS,
} from './HighAvailabilityBadge.styles';
import Stack from '@mui/material/Stack';
import { HighAvailabilityBadgeProps } from './HighAvailabilityBadge.types';

const HighAvailabilityBadge: FC<HighAvailabilityBadgeProps> = ({
  health,
  sx,
  ...props
}) => {
  // Upstream added an explicit "unknown" health that renders no badge.
  if (health === 'unknown') {
    return null;
  }

  return (
    <Stack flex={8} alignItems="flex-start">
      <Chip
        data-testid="ha-badge"
        color={HEALTH_CHIP_COLOR[health]}
        variant="outlined"
        label={HIGH_AVAILABILITY_BADGE_HEALTH[health]}
        // Per-state emphasis first, then any caller sx so callers can still
        // override without silently dropping the health font weight / border.
        sx={[HEALTH_EMPHASIS[health], ...(Array.isArray(sx) ? sx : [sx])]}
        {...props}
      />
    </Stack>
  );
};

export default HighAvailabilityBadge;
