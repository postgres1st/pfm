import { describe, it, expect } from 'vitest';
import { pmmThemeOptions } from '@percona/percona-ui';
import { postgres1stThemeOptions } from './theme';

const primary = (mode: 'light' | 'dark') =>
  postgres1stThemeOptions(mode).palette?.primary as { main: string };
const secondary = (mode: 'light' | 'dark') =>
  postgres1stThemeOptions(mode).palette?.secondary as { main: string };

describe('postgres1stThemeOptions', () => {
  it('applies the Postgres1st blue primary (light)', () => {
    expect(primary('light').main).toBe('#0E7ABE');
  });

  it('applies a lighter blue primary in dark mode', () => {
    expect(primary('dark').main).toBe('#5EAEE0');
  });

  it('applies the amber secondary', () => {
    expect(secondary('light').main).toBe('#F5B94D');
  });

  it('preserves design-system semantic colors (error) from the base theme', () => {
    const baseError = (pmmThemeOptions('light').palette?.error as { main: string })
      ?.main;
    const brandError = (
      postgres1stThemeOptions('light').palette?.error as { main: string }
    )?.main;
    expect(brandError).toBe(baseError);
  });
});
