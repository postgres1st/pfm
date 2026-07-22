import { fireEvent, render, screen } from '@testing-library/react';
import { wrapWithRouter } from 'utils/testUtils';
import QanHeaderTabs from './QanHeaderTabs';
import { Route, Routes } from 'react-router-dom';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';

const TabsTestComponent = () => (
  <>
    <QanHeaderTabs />
    <Routes>
      <Route path="/" element={<div>Home</div>} />
      <Route
        path={`${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-qan/pmm-query-analytics`}
        element={<div data-testid="historical-tab-content">Historical</div>}
      />
    </Routes>
  </>
);

describe('QanHeaderTabs', () => {
  it('should render only the historical tab (real-time QAN is unsupported)', () => {
    render(wrapWithRouter(<TabsTestComponent />));

    expect(
      screen.getByTestId('qan-header-tabs-historical-tab')
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId('qan-header-tabs-real-time-tab')
    ).not.toBeInTheDocument();
  });

  it('should navigate to historical tab', async () => {
    render(wrapWithRouter(<TabsTestComponent />));

    fireEvent.click(screen.getByTestId('qan-header-tabs-historical-tab'));

    expect(screen.getByTestId('historical-tab-content')).toBeInTheDocument();
  });
});
