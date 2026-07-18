import { addConfiguration } from './navigation.utils';
import { NAV_CONFIGURATION } from './navigation.constants';
import { GetUpdatesResponse, UpdateStatus } from 'types/updates.types';

// The Updates feature is flag-disabled in the shipped app; force it on here to
// cover the enabled path (where addConfiguration touches the updates nav item).
vi.mock('lib/constants', async (importOriginal) => {
  const actual = await importOriginal<typeof import('lib/constants')>();
  return { ...actual, UPDATES_ENABLED: true };
});

const VERSION_INFO: GetUpdatesResponse = {
  lastCheck: '2026-07-07T00:00:00Z',
  latest: {
    version: '3.2.0',
    tag: '3.2.0',
    timestamp: null,
    releaseNotesText: '',
    releaseNotesUrl: '',
  },
  installed: { version: '3.1.0', fullVersion: '3.1.0', timestamp: null },
  latestNewsUrl: '',
  updateAvailable: true,
};

const getUpdatesChild = (node = NAV_CONFIGURATION) =>
  node.children?.find((c) => c.id === 'updates');

describe('addConfiguration (UPDATES_ENABLED=true)', () => {
  it('does not mutate the shared NAV_CONFIGURATION updates child', () => {
    // Baseline: the shared constant carries no dynamic state.
    expect(getUpdatesChild()?.secondaryText).toBeUndefined();
    expect(getUpdatesChild()?.badge).toBeUndefined();

    addConfiguration(UpdateStatus.Pending, VERSION_INFO);

    // The shared module-level constant must remain pristine across renders.
    expect(getUpdatesChild()?.secondaryText).toBeUndefined();
    expect(getUpdatesChild()?.badge).toBeUndefined();
  });

  it('returns a nav with computed updates secondaryText and badge', () => {
    const result = addConfiguration(UpdateStatus.Pending, VERSION_INFO);
    const updates = getUpdatesChild(result);

    expect(updates?.secondaryText).toBe('Update from v3.1.0 to v3.2.0');
    expect(updates?.badge).toEqual({ label: 'New' });
  });

  it('renders multi-digit version components without truncation', () => {
    // A version with a 2-digit component — the case the old slice(0, 5) broke.
    const installedVersion = '3.10.2';
    const result = addConfiguration(UpdateStatus.Pending, {
      ...VERSION_INFO,
      installed: {
        version: installedVersion,
        fullVersion: installedVersion,
        timestamp: null,
      },
    });

    // Guards against the old installed.version.slice(0, 5), which truncated
    // "3.10.2" to "3.10.".
    expect(getUpdatesChild(result)?.secondaryText).toBe(
      `Update from v${installedVersion} to v${VERSION_INFO.latest?.version}`
    );
  });
});
