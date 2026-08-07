// PFMM ships no update source: pfm-managed.service sets PMM_ENABLE_UPDATES=false and
// the settings API treats an env-set value as immutable, so an Updates toggle could
// only ever return FailedPrecondition. It is hidden behind the same UPDATES_ENABLED
// flag as the /updates route, the nav entry and the header banner.
//
// Both paths are covered: the shipped build (flag off, toggle absent) and the
// re-enabled build (flag on, toggle present), so turning updates back on later is
// tested rather than assumed.
describe('FEATURE_MANAGEMENT_SETTINGS (UPDATES_ENABLED=false — shipped)', () => {
  beforeEach(() => {
    vi.resetModules();
  });

  it('omits the updates toggle', async () => {
    const { FEATURE_MANAGEMENT_SETTINGS } =
      await import('./Advanced.constants');

    expect(FEATURE_MANAGEMENT_SETTINGS.map((s) => s.name)).not.toContain(
      'updates'
    );
  });

  it('still exposes the other feature toggles', async () => {
    const { FEATURE_MANAGEMENT_SETTINGS } =
      await import('./Advanced.constants');
    const names = FEATURE_MANAGEMENT_SETTINGS.map((s) => s.name);

    // Guards against the spread accidentally swallowing its neighbours.
    expect(names).toContain('alerting');
    expect(names).toContain('backup');
    expect(names.length).toBeGreaterThan(1);
  });
});

describe('FEATURE_MANAGEMENT_SETTINGS (UPDATES_ENABLED=true — re-enabled)', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.doMock('lib/constants', async (importOriginal) => {
      const actual = await importOriginal<typeof import('lib/constants')>();
      return { ...actual, UPDATES_ENABLED: true };
    });
  });

  afterEach(() => {
    vi.doUnmock('lib/constants');
  });

  it('restores the updates toggle, testId intact', async () => {
    const { FEATURE_MANAGEMENT_SETTINGS } =
      await import('./Advanced.constants');
    const updates = FEATURE_MANAGEMENT_SETTINGS.find(
      (s) => s.name === 'updates'
    );

    expect(updates).toBeDefined();
    expect(updates?.testId).toBe('advanced-updates');
  });
});
