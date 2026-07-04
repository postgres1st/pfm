export const Messages = {
  fetchError: "Couldn't load current version information.",
  upToDate: 'This PFMM instance is up to date.',
  newUpdateAvailable: (version: string) =>
    `New update available: PFMM ${version}`,
  runningVersion: 'Running version:',
  newVersion: 'New version:',
  lastChecked: 'Last checked:',
  home: 'PFMM home',
  checkNow: 'Check updates now',
  checking: 'Checking',
  updateNow: 'Update now',
  error: 'There was a problem during the update',

  deprecation: {
    heading: 'UI upgrades deprecated',
    paragraph1BeforeUpdateNow: ': This ',
    paragraph1AfterUpdateNow: ' button will be removed in PFMM 3.9.0.',
    viaIntro: 'After that, PFMM upgrades will only be available via\u00a0',
    docker: 'Docker',
    afterDocker: ' (recommended), ',
    podman: 'Podman',
    afterPodman: ', or ',
    helm: 'Helm',
    afterHelm: '.',
    reminder: 'Switch before then to keep upgrading PFMM to newer versions.',
  },
};
