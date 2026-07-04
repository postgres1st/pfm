export const Messages = {
  title: 'Settings',
  tabs: {
    ssh: 'SSH key',
    metrics: 'Metrics resolution',
    advanced: 'Advanced settings',
  },
  advanced: {
    validation: {
      required: 'Required field',
      retentionRange: (min: number, max: number) =>
        `Value should be in the range from ${min} to ${max}`,
      intervalMin: (min: number) => `Min ${min}`,
    },
    retentionLabel: 'Data retention',
    retentionTooltip:
      'How long PFMM keeps collected data. Older data is automatically deleted.',
    retentionUnits: 'days',
    retentionLink: 'https://docs.postgresfirst.com/data_retention',
    telemetryLabel: 'Telemetry',
    telemetryLink: 'https://docs.postgresfirst.com/telemetry',
    telemetryDialogLink: 'What we collect',
    telemetryTooltip:
      'Sends anonymous usage statistics to help improve PFMM. No personal or database content is collected.',
    telemetrySummaryTitle:
      'We gather and send the following information to Postgres1st:',
    updatesLabel: 'Check for updates',
    updatesLink: 'https://docs.postgresfirst.com/updates',
    updatesTooltip:
      'Option to check new versions and ability to update PFMM from UI.',
    advisorsLabel: 'Advisors',
    sttRareIntervalLabel: 'Rare',
    sttStandardIntervalLabel: 'Standard',
    sttFrequentIntervalLabel: 'Frequent',
    sttCheckIntervalTooltip:
      'How often Advisor checks run. Lower values catch issues faster but increase resource usage.',
    advisorsLink: 'https://docs.postgresfirst.com/advisors',
    advisorsTooltip:
      'Run automated checks to identify potential database performance and configuration issues.',
    azureDiscoverLabel: 'Microsoft Azure monitoring',
    azureDiscoverTooltip:
      'Option to enable/disable Microsoft Azure DB instances discovery and monitoring',
    azureDiscoverLink: 'https://docs.postgresfirst.com/azure_monitoring',
    accessControl: 'Access control',
    accessControlTooltip:
      'Restrict data visibility based on user roles and labels.',
    accessControlLink: 'https://docs.postgresfirst.com/roles_permissions',
    publicAddressLabel: 'Public address',
    publicAddressTooltip:
      'The address or hostname PFMM Server will be accessible at.',
    publicAddressPlaceholder: 'https://...',
    publicAddressButton: 'Get from browser',
    alertingLabel: 'Postgres1st Alerting',
    alertingTooltip: 'Option to enable/disable Postgres1st Alerting features.',
    alertingLink: 'https://docs.postgresfirst.com/alerting',
    backupLabel: 'Backup Management',
    backupTooltip:
      'Enable scheduled and on-demand backups for supported databases.',
    backupLink: 'https://docs.postgresfirst.com/backup_management',
    enableInternalPgQanLabel: 'QAN for PFMM Server',
    enableInternalPgQanTooltip:
      "Displays queries from PFMM Server's internal PostgreSQL database in Query Analytics (QAN). Enable to troubleshoot PFMM Server's database performance alongside your monitored instances.",
    enableInternalPgQanLink: 'https://docs.postgresfirst.com/qan-pmm-server',
    featureManagementLabel: 'Feature management',
    featureManagementDescription:
      'Enable or disable core PFMM capabilities. Turning off unused features can help conserve system resources and simplify your navigation menu.',
    technicalPreviewLegend: 'Technical preview features',
    technicalPreviewDescription: 'These are technical preview features, ',
    technicalPreviewWarning: 'not recommended',
    technicalPreviewDescriptionSuffix:
      ' to be used in production environments. Read more about feature status',
    technicalPreviewLinkText: 'here.',
  },
  metrics: {
    label: 'Metrics resolution',
    link: 'https://docs.postgresfirst.com/metrics_resolution',
    options: {
      rare: 'Rare',
      standard: 'Standard',
      frequent: 'Frequent',
      custom: 'Custom',
    },
    intervals: {
      low: 'Low',
      medium: 'Medium',
      high: 'High',
    },
    tooltip:
      'How often PFMM collects metrics, in seconds. Lower values provide more detail but use more resources.',
    validation: {
      required: 'Required',
      minMax: (min: number, max: number) => `Must be between ${min} and ${max}`,
    },
  },
  ssh: {
    label: 'SSH key',
    link: 'https://docs.postgresfirst.com/ssh_key',
    tooltip:
      'Paste your public SSH key (ssh-rsa format) to enable SSH access to PFMM Server.',
    placeholder: 'ssh-rsa AAAA...',
    validation: {
      invalidFormat: 'Enter a valid SSH public key (e.g. ssh-rsa, ssh-ed25519)',
    },
  },
  service: {
    success: 'Settings updated',
  },
  tooltipLinkText: 'Read more',
  unauthorized: 'Insufficient access permissions.',
  applyChanges: 'Apply changes',
  applying: 'Applying...',
};
