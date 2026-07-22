import { AdvisorFamily, AdvisorInterval } from 'types/advisors.types';
import { ManagedServiceType, ServiceType } from 'types/services.types';

export const PMM_TITLE = 'Postgres1st Monitoring and Management';

// Feature flags — flip to true to re-enable.
export const TOUR_ENABLED = false;
export const UPDATES_ENABLED = false;
// todo: remove completely in a follow up to reduce current scope
export const PMM_NEW_NAV_PATH = '';
export const GRAFANA_SUB_PATH = '/graph';
export const PMM_BASE_PATH = `/pmm-ui${PMM_NEW_NAV_PATH}`;
export const PMM_NEW_NAV_GRAFANA_PATH = `${PMM_NEW_NAV_PATH}${GRAFANA_SUB_PATH}`;
export const PMM_HOME_URL = `${GRAFANA_SUB_PATH}/d/pmm-home`;
export const PMM_LOGIN_URL = `${GRAFANA_SUB_PATH}/login`;
export const PMM_SETTINGS_URL = `${PMM_BASE_PATH}/settings`;
export const PMM_NEW_NAV_UPDATES_PATH = `${PMM_NEW_NAV_PATH}/updates`;
export const PMM_SUPPORT_URL = 'https://docs.postgresfirst.com/pmm_documentation';
export const PMM_DOCS_UPDATES_URL = 'https://docs.postgresfirst.com/pmm-upgrade';
export const PMM_DOCS_UPDATE_CLIENT_URL = 'https://docs.postgresfirst.com/pmm-upgrade-agent';
export const PMM_NEW_NAV_HOME_URL = `${PMM_NEW_NAV_PATH}/graph/d/pmm-home`;

export const INTERVALS_MS = {
  // 5 mins
  SERVICE_TYPES: 300000,
};

export const ADVISOR_FAMILY: Record<AdvisorFamily, string> = {
  [AdvisorFamily.mysql]: 'MySQL',
  [AdvisorFamily.postgresql]: 'PostgreSQL',
  [AdvisorFamily.mongodb]: 'MongoDB',
  [AdvisorFamily.unspecified]: 'Unspecified',
};

export const ADVISOR_INTERVAL: Record<AdvisorInterval, string> = {
  [AdvisorInterval.standard]: 'Standard',
  [AdvisorInterval.rare]: 'Rare',
  [AdvisorInterval.frequent]: 'Frequent',
  [AdvisorInterval.unspecified]: 'Unspecified',
};

// Maps the inventory ServiceType enum (SERVICE_TYPE_*) to the lowercase model
// identifier the server reports in `supported_service_types` (e.g. "postgresql").
export const SERVICE_TYPE_MODEL_ID: Record<ServiceType, ManagedServiceType | ''> =
  {
    [ServiceType.unspecified]: '',
    [ServiceType.mysql]: ManagedServiceType.mysql,
    [ServiceType.mongodb]: ManagedServiceType.mongodb,
    [ServiceType.posgresql]: ManagedServiceType.postgresql,
    [ServiceType.proxysql]: ManagedServiceType.proxysql,
    [ServiceType.haproxy]: ManagedServiceType.haproxy,
    [ServiceType.valkey]: ManagedServiceType.valkey,
    [ServiceType.external]: ManagedServiceType.external,
  };

// Fallback used only until the server settings (and their runtime
// `supportedServiceTypes` allowlist) have loaded. Mirrors the backend's shipped
// default so the sidebar shows the PostgreSQL-first set without flicker; once
// settings arrive, the live value (which honours PFM_DB_TYPES) takes over.
export const DEFAULT_SUPPORTED_SERVICE_TYPES: string[] = [
  ManagedServiceType.postgresql,
  ManagedServiceType.haproxy,
  ManagedServiceType.external,
];

// 5 seconds
export const SHOW_UPDATE_INFO_DELAY_MS = 5000;
// 1 hour
export const SHOW_UPDATE_MODAL_AFTER_MS = 60 * 60 * 1000;

export const DOCS_URLS = {
  qan: 'https://docs.postgresfirst.com/QAN',
  forums: 'https://docs.postgresfirst.com/PMM3_forums',
};

export const TIME_FORMAT = 'yyyy-MM-dd HH:mm:ss';
