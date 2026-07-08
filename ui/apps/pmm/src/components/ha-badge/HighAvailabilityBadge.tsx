import Chip from '@mui/material/Chip';
import { FC } from 'react';
import { HIGH_AVAILABILITY_BADGE_HEALTH } from './HighAvailabilityBadge.constants';
import {
  HEALTH_CHIP_COLOR,
  HEALTH_FONT_WEIGHT,
} from './HighAvailabilityBadge.styles';
import Stack from '@mui/material/Stack';
import { HighAvailabilityBadgeProps } from './HighAvailabilityBadge.types';

const HighAvailabilityBadge: FC<HighAvailabilityBadgeProps> = ({
  health,
  ...props
}) => (
  <Stack flex={8} alignItems="flex-start">
    <Chip
      data-testid="ha-badge"
      color={HEALTH_CHIP_COLOR[health]}
      variant="outlined"
      label={HIGH_AVAILABILITY_BADGE_HEALTH[health]}
      sx={{ fontWeight: HEALTH_FONT_WEIGHT[health] }}
      {...props}
    />
  </Stack>
);

export default HighAvailabilityBadge;
