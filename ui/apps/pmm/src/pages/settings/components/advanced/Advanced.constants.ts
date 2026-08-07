import { UPDATES_ENABLED } from 'lib/constants';

import { Messages } from '../../Settings.messages';

export const SECONDS = 60;
export const MINUTES = 60;
export const HOURS = 24;
export const SECONDS_IN_DAY = SECONDS * MINUTES * HOURS;
export const MINUTES_IN_DAY = MINUTES * HOURS;
export const MIN_DAYS = 1;
export const MAX_DAYS = 3650;
export const MIN_STT_CHECK_INTERVAL = 0.1;
export const STT_CHECK_INTERVAL_STEP = 0.1;
export const DEFAULT_DATA_RETENTION = '86400s';

export const STT_CHECK_INTERVALS = [
  {
    label: Messages.advanced.sttRareIntervalLabel,
    name: 'rareInterval' as const,
  },
  {
    label: Messages.advanced.sttStandardIntervalLabel,
    name: 'standardInterval' as const,
  },
  {
    label: Messages.advanced.sttFrequentIntervalLabel,
    name: 'frequentInterval' as const,
  },
];

export const TECHNICAL_PREVIEW_DOC_URL =
  'https://docs.postgresfirst.com/pmm-feature-status';

export const FEATURE_MANAGEMENT_SETTINGS = [
  // Gated on the same UPDATES_ENABLED flag as the /updates route, the nav entry, the
  // polling provider and the header banner — this was the one surface it was never
  // applied to. PFMM ships no update source, and pfm-managed.service sets
  // PMM_ENABLE_UPDATES=false, which the settings API treats as immutable
  // (FailedPrecondition), so the toggle could only ever return an error.
  //
  // Hidden rather than removed: the form still round-trips updatesEnabled unchanged,
  // so re-enabling updates later is this flag and nothing else.
  ...(UPDATES_ENABLED
    ? [
        {
          name: 'updates' as const,
          label: Messages.advanced.updatesLabel,
          tooltip: Messages.advanced.updatesTooltip,
          link: Messages.advanced.updatesLink,
          testId: 'advanced-updates',
        },
      ]
    : []),
  {
    name: 'alerting' as const,
    label: Messages.advanced.alertingLabel,
    tooltip: Messages.advanced.alertingTooltip,
    link: Messages.advanced.alertingLink,
    testId: 'advanced-alerting',
  },
  {
    name: 'backup' as const,
    label: Messages.advanced.backupLabel,
    tooltip: Messages.advanced.backupTooltip,
    link: Messages.advanced.backupLink,
    testId: 'advanced-backup',
  },
  {
    name: 'enableInternalPgQan' as const,
    label: Messages.advanced.enableInternalPgQanLabel,
    tooltip: Messages.advanced.enableInternalPgQanTooltip,
    link: Messages.advanced.enableInternalPgQanLink,
    testId: 'enable-internal-pg-qan',
  },
];
