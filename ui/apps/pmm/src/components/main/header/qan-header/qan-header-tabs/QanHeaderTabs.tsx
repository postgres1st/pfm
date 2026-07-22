import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import { Messages } from '../QanHeader.messages';
import { FC } from 'react';
import { Link } from 'react-router-dom';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';

// Real-time QAN is MongoDB-only, which this PostgreSQL-only build does not
// support, so only the stored-metrics (historical) tab is shown.
const QanHeaderTabs: FC = () => {
  return (
    <Tabs value="historical">
      <Tab
        value="historical"
        label={Messages.tabStoredMetrics}
        component={Link}
        to={`${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-qan/pmm-query-analytics`}
        data-testid="qan-header-tabs-historical-tab"
      />
    </Tabs>
  );
};

export default QanHeaderTabs;
