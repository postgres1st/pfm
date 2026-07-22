import {
  addConfiguration,
  filterSupportedServiceTypes,
} from './navigation.utils';
import { NAV_CONFIGURATION } from './navigation.constants';
import { GetUpdatesResponse, UpdateStatus } from 'types/updates.types';
import { ManagedServiceType, ServiceType } from 'types/services.types';
import { DEFAULT_SUPPORTED_SERVICE_TYPES } from 'lib/constants';

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

describe('filterSupportedServiceTypes', () => {
  const ALL: ServiceType[] = [
    ServiceType.mysql,
    ServiceType.mongodb,
    ServiceType.posgresql,
    ServiceType.proxysql,
    ServiceType.haproxy,
    ServiceType.valkey,
    ServiceType.external,
  ];

  it('keeps only the types present in the supported allowlist', () => {
    expect(
      filterSupportedServiceTypes(ALL, DEFAULT_SUPPORTED_SERVICE_TYPES)
    ).toEqual([
      ServiceType.posgresql,
      ServiceType.haproxy,
      ServiceType.external,
    ]);
  });

  it('drops unsupported DB types (MySQL, MongoDB, ProxySQL, Valkey)', () => {
    const result = filterSupportedServiceTypes(
      ALL,
      DEFAULT_SUPPORTED_SERVICE_TYPES
    );

    expect(result).not.toContain(ServiceType.mysql);
    expect(result).not.toContain(ServiceType.mongodb);
    expect(result).not.toContain(ServiceType.proxysql);
    expect(result).not.toContain(ServiceType.valkey);
  });

  it('honours a runtime override that re-enables a type', () => {
    const supported = [...DEFAULT_SUPPORTED_SERVICE_TYPES, ManagedServiceType.mysql];

    expect(filterSupportedServiceTypes(ALL, supported)).toContain(
      ServiceType.mysql
    );
  });

  it('returns an empty list when nothing is supported', () => {
    expect(filterSupportedServiceTypes(ALL, [])).toEqual([]);
  });

  it('drops the unspecified type and any unmapped inventory type', () => {
    // unspecified maps to '' and a future/unknown type maps to undefined — both
    // must be filtered out rather than leaking through as a matchless nav tree.
    const withOddballs = [
      ServiceType.posgresql,
      ServiceType.unspecified,
      'SERVICE_TYPE_FUTURE' as ServiceType,
    ];

    expect(
      filterSupportedServiceTypes(withOddballs, DEFAULT_SUPPORTED_SERVICE_TYPES)
    ).toEqual([ServiceType.posgresql]);
  });
});

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
