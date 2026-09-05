export const Messages = {
  fetchError: "Couldn't load current version information.",
  upToDate: 'This PGF WatchTower instance is up to date.',
  newUpdateAvailable: (version: string) =>
    `New update available: PGF WatchTower ${version}`,
  runningVersion: 'Running version:',
  newVersion: 'New version:',
  deprecationWarning:
    'Note: The in-app update button has been deprecated. All updates are now securely managed via the CLI.',
  lastChecked: 'Last checked:',
  home: 'PGF WatchTower home',
  checkNow: 'Check updates now',
  checking: 'Checking',
  howTo: 'How to',
  howToUpdateDocs: 'How to update docs',
  error: 'There was a problem during the update',

  deprecation: {
    heading: 'UI upgrades deprecated',
    paragraph1BeforeUpdateNow: ': This ',
    paragraph1AfterUpdateNow: ' button will be removed in PGF WatchTower 3.9.0.',
    viaIntro: 'After that, PGF WatchTower upgrades will only be available via\u00a0',
    docker: 'Docker',
    afterDocker: ' (recommended), ',
    podman: 'Podman',
    afterPodman: ', or ',
    helm: 'Helm',
    afterHelm: '.',
    reminder: 'Switch before then to keep upgrading PGF WatchTower to newer versions.',
  },
};
