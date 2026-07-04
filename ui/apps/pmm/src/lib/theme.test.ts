import { describe, it, expect } from 'vitest';
import { pmmThemeOptions } from '@percona/percona-ui';
import { postgres1stThemeOptions } from './theme';

const primary = (mode: 'light' | 'dark') =>
  postgres1stThemeOptions(mode).palette?.primary as { main: string };
const secondary = (mode: 'light' | 'dark') =>
  postgres1stThemeOptions(mode).palette?.secondary as { main: string };

describe('postgres1stThemeOptions', () => {
  it('applies the Postgres1st periwinkle primary (light)', () => {
    expect(primary('light').main).toBe('#653DF4');
  });

  it('applies a lighter periwinkle primary in dark mode', () => {
    expect(primary('dark').main).toBe('#B6B2FF');
  });

  it('applies the amber secondary', () => {
    expect(secondary('light').main).toBe('#F5A623');
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
