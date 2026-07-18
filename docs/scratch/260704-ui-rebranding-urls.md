# Percona URL replacements — review list

_Generated 2026-07-04 for the UI rebrand (branch `ui-rebranding`)._

Per decision D5, all Percona-domain links were replaced with **placeholder** Postgres1st domains and MUST be reviewed once real infra exists. URL **paths were preserved** (only the domain changed) so each link stays traceable to its original purpose. Many will 404 until Postgres1st docs/support/blog/forum infrastructure is live.

## Domain mapping applied

| Original domain | Placeholder | Notes |
|---|---|---|
| `per.co.na` | `docs.postgresfirst.com` | Percona docs short-link redirector |
| `www.percona.com` | `www.postgresfirst.com` | marketing/blog/contact |
| `percona.com` (bare) | `postgresfirst.com` | incl. some Grafana `/d/...` dashboard links |
| `docs.percona.com` | `docs.postgresfirst.com` | product docs |
| `forums.percona.com` | `forums.postgresfirst.com` | community forum |
| `/percona-blog/feed` | `/postgresfirst-blog/feed` | RSS panel feed |
| `github.com/percona/pmm` | `github.com/postgres1st/pfm` | handled in plugin.json (placeholder repo) |

## Every occurrence (file:line → original URL)

```
dashboards/dashboards/Experimental/DB_Cluster_Summary.json:83:https://forums.percona.com/c/percona-monitoring-and-management-pmm/pmm-unofficial-dashboards-and-plugins/67
dashboards/dashboards/Experimental/DB_Cluster_Summary.json:83:https://per.co.na/dbaas
dashboards/dashboards/Insight/Prometheus_Exporters_Overview.json:1051:https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/
dashboards/dashboards/Insight/Prometheus_Exporters_Overview.json:1137:https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/
dashboards/dashboards/Insight/Prometheus_Exporters_Overview.json:121:https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/
dashboards/dashboards/Insight/Prometheus_Exporters_Overview.json:1314:https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/
dashboards/dashboards/Insight/Prometheus_Exporters_Overview.json:1407:https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/
dashboards/dashboards/Insight/Prometheus_Exporters_Overview.json:201:https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/
dashboards/dashboards/Insight/Prometheus_Exporters_Overview.json:280:https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/
dashboards/dashboards/Insight/Prometheus_Exporters_Overview.json:367:https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/
dashboards/dashboards/Insight/Prometheus_Exporters_Overview.json:470:https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/
dashboards/dashboards/Insight/Prometheus_Exporters_Overview.json:576:https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/
dashboards/dashboards/Insight/Prometheus_Exporters_Overview.json:694:https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/
dashboards/dashboards/Insight/Prometheus_Exporters_Overview.json:804:https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/
dashboards/dashboards/Insight/Prometheus_Exporters_Overview.json:894:https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/
dashboards/dashboards/Insight/Prometheus_Exporters_Overview.json:966:https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/
dashboards/dashboards/Kubernetes (experimental)/Databases_on_Kubernetes.json:41:https://docs.percona.com
dashboards/dashboards/Kubernetes (experimental)/Databases_on_Kubernetes.json:53:https://www.percona.com/about/contact
dashboards/dashboards/MongoDB/MongoDB_ReplSet_Summary.json:598:https://per.co.na/mongo-repstate
dashboards/dashboards/MySQL/MySQL_InnoDB_Compression_Details.json:1451:https://www.percona.com/blog/2015/06/02/80-ram-tune-innodb_buffer_pool_size/
dashboards/dashboards/MySQL/MySQL_InnoDB_Compression_Details.json:1542:https://www.percona.com/blog/2015/06/02/80-ram-tune-innodb_buffer_pool_size/
dashboards/dashboards/MySQL/MySQL_InnoDB_Details.json:13159:https://www.percona.com/blog/2008/11/21/how-to-calculate-a-good-innodb-log-file-size/
dashboards/dashboards/MySQL/MySQL_InnoDB_Details.json:13164:https://per.co.na/innodb_log_file_size
dashboards/dashboards/MySQL/MySQL_InnoDB_Details.json:136:https://www.percona.com/blog/2015/06/02/80-ram-tune-innodb_buffer_pool_size/
dashboards/dashboards/MySQL/MySQL_InnoDB_Details.json:21591:https://www.percona.com/blog/2017/05/09/mariadb-handler_icp_-counters-what-they-are-and-how-to-use-them/
dashboards/dashboards/MySQL/MySQL_InnoDB_Details.json:227:https://www.percona.com/blog/2015/06/02/80-ram-tune-innodb_buffer_pool_size/
dashboards/dashboards/MySQL/MySQL_InnoDB_Details.json:6325:https://www.percona.com/blog/2015/06/02/80-ram-tune-innodb_buffer_pool_size/
dashboards/dashboards/MySQL/MySQL_InnoDB_Details.json:6415:https://www.percona.com/blog/2015/06/02/80-ram-tune-innodb_buffer_pool_size/
dashboards/dashboards/MySQL/MySQL_Instances_Compare.json:1016:https://www.percona.com/blog/2014/05/29/how-mysql-queries-and-questions-are-measured/
dashboards/dashboards/MySQL/MySQL_Instances_Compare.json:2504:https://per.co.na/mysql_internal_memory_overview
dashboards/dashboards/MySQL/MySQL_Instances_Compare.json:513:https://www.percona.com/blog/2015/06/02/80-ram-tune-innodb_buffer_pool_size/
dashboards/dashboards/MySQL/MySQL_Instances_Compare.json:609:https://www.percona.com/blog/2015/06/02/80-ram-tune-innodb_buffer_pool_size/
dashboards/dashboards/MySQL/MySQL_Instances_Overview.json:2710:https://www.percona.com/blog/2014/05/29/how-mysql-queries-and-questions-are-measured/
dashboards/dashboards/MySQL/MySQL_Instances_Overview.json:2954:https://www.percona.com/blog/2014/05/29/how-mysql-queries-and-questions-are-measured/
dashboards/dashboards/MySQL/MySQL_Instances_Overview.json:3732:https://www.percona.com/blog/2014/05/29/how-mysql-queries-and-questions-are-measured/
dashboards/dashboards/MySQL/MySQL_Instances_Overview.json:418:https://www.percona.com/blog/2015/06/02/80-ram-tune-innodb_buffer_pool_size/
dashboards/dashboards/MySQL/MySQL_Instance_Summary.json:1801:https://www.percona.com/blog/2014/05/29/how-mysql-queries-and-questions-are-measured/
dashboards/dashboards/MySQL/MySQL_Instance_Summary.json:2179:https://per.co.na/mysql_internal_memory_overview
dashboards/dashboards/MySQL/MySQL_Instance_Summary.json:305:https://www.percona.com/blog/2015/06/02/80-ram-tune-innodb_buffer_pool_size/
dashboards/dashboards/MySQL/MySQL_Instance_Summary.json:395:https://www.percona.com/blog/2015/06/02/80-ram-tune-innodb_buffer_pool_size/
dashboards/dashboards/MySQL/MySQL_MyISAM_Aria_Details.json:671:https://per.co.na/aria-storage-engine
dashboards/dashboards/MySQL/MySQL_MyISAM_Aria_Details.json:723:https://per.co.na/aria-system-variables
dashboards/dashboards/MySQL/MySQL_Query_Response_Time_Details.json:116:https://per.co.na/query_response_time
dashboards/dashboards/MySQL/MySQL_Query_Response_Time_Details.json:245:https://per.co.na/query_response_time
dashboards/dashboards/MySQL/MySQL_Query_Response_Time_Details.json:366:https://per.co.na/5_6_diagnostics_response_time_distribution
dashboards/dashboards/MySQL/MySQL_Query_Response_Time_Details.json:393:https://per.co.na/5_7_diagnostics_response_time_distribution_logging
dashboards/dashboards/MySQL/MySQL_Query_Response_Time_Details.json:393:https://per.co.na/5_7_diagnostics_response_time_distribution_read
dashboards/dashboards/MySQL/MySQL_Query_Response_Time_Details.json:393:https://per.co.na/5_7_diagnostics_response_time_distribution_write
dashboards/dashboards/MySQL/MySQL_Query_Response_Time_Details.json:433:https://per.co.na/5_7_diagnostics_response_time_distribution_logging
dashboards/dashboards/MySQL/MySQL_Query_Response_Time_Details.json:574:https://per.co.na/5_7_diagnostics_response_time_distribution_read
dashboards/dashboards/MySQL/MySQL_Query_Response_Time_Details.json:733:https://per.co.na/5_7_diagnostics_response_time_distribution_write
dashboards/dashboards/MySQL/MySQL_Query_Response_Time_Details.json:76:https://per.co.na/query_response_time
dashboards/dashboards/MySQL/MySQL_Table_Details.json:451:https://per.co.na/5_6_diagnostics_user_stat
dashboards/dashboards/MySQL/MySQL_User_Details.json:81:https://per.co.na/8_0_diagnostics_user_stat
dashboards/dashboards/PMM Health/Environments_Overview.json:133:https://www.percona.com/pmm
dashboards/dashboards/PMM Health/Environments_Overview.json:88:https://forums.percona.com/c/percona-monitoring-and-management-pmm/pmm-unofficial-dashboards-and-plugins/67
dashboards/dashboards/PostgreSQL/PostgreSQL_Overview_Extended.json:200:https://per.co.na/pg-max-conn
dashboards/dashboards/Valkey/Valkey_Overview.json:62:https://docs.percona.com/valkey/index.html
dashboards/dashboards/Valkey/Valkey_Overview.json:62:https://docs.percona.com/valkey/index.html
dashboards/dashboards/Valkey/Valkey_Overview.json:62:https://forums.percona.com/c/valkey/
dashboards/dashboards/Valkey/Valkey_Overview.json:62:https://forums.percona.com/c/valkey/
dashboards/dashboards/Valkey/Valkey_Overview.json:62:https://www.percona.com/about/contact?utm_campaign=7075599-Product
dashboards/dashboards/Valkey/Valkey_Overview.json:62:https://www.percona.com/about/contact?utm_campaign=7075599-Product
dashboards/dashboards/Valkey/Valkey_Overview.json:62:https://www.percona.com/blog/category/valkey/
dashboards/dashboards/Valkey/Valkey_Overview.json:62:https://www.percona.com/blog/category/valkey/
ui/apps/pmm-compat/src/lib/utils/variables.test.ts:101:https://percona.com/d/postgresql-instance-overview/postgresql-instances-overview?var-empty=&var-empty-old=None&var-value=Value
ui/apps/pmm-compat/src/lib/utils/variables.test.ts:102:https://percona.com/d/postgresql-instance-overview/postgresql-instances-overview?var-value=Value
ui/apps/pmm-compat/src/lib/utils/variables.test.ts:109:https://percona.com/d/postgresql-instance-overview/postgresql-instances-overview?var-all=
ui/apps/pmm-compat/src/lib/utils/variables.test.ts:110:https://percona.com/d/postgresql-instance-overview/postgresql-instances-overview?var-value=Value
ui/apps/pmm-compat/src/lib/utils/variables.test.ts:117:https://percona.com/d/postgresql-instance-overview/postgresql-instances-overview?var-all=
ui/apps/pmm-compat/src/lib/utils/variables.test.ts:118:https://percona.com/d/postgresql-instance-overview/postgresql-instances-overview?var-value=Value
ui/apps/pmm-compat/src/lib/utils/variables.test.ts:34:https://percona.com
ui/apps/pmm-compat/src/lib/utils/variables.test.ts:46:https://percona.com
ui/apps/pmm-compat/src/lib/utils/variables.test.ts:94:https://percona.com/d/postgresql-instance-overview/postgresql-instances-overview
ui/apps/pmm/src/components/main/header/qan-header/qan-header-actions/QanHeaderActions.test.tsx:11:https://www.percona.com/
ui/apps/pmm/src/components/main/header/qan-header/qan-header-actions/QanHeaderActions.test.tsx:73:https://www.percona.com/
ui/apps/pmm/src/components/main/update-modal/UpdateModal.test.tsx:36:https://per.co.na/pmm/3.1.0
ui/apps/pmm/src/contexts/tour/steps/product.steps.tsx:100:https://per.co.na/backup_management
ui/apps/pmm/src/contexts/tour/steps/product.steps.tsx:124:https://per.co.na/configure
ui/apps/pmm/src/contexts/tour/steps/product.steps.tsx:57:https://per.co.na/alert_templates
ui/apps/pmm/src/contexts/tour/steps/product.steps.tsx:75:https://per.co.na/advisors
ui/apps/pmm/src/lib/constants.ts:14:https://per.co.na/pmm_documentation
ui/apps/pmm/src/lib/constants.ts:15:https://per.co.na/pmm-upgrade
ui/apps/pmm/src/lib/constants.ts:16:https://per.co.na/pmm-upgrade-agent
ui/apps/pmm/src/lib/constants.ts:54:https://per.co.na/QAN
ui/apps/pmm/src/lib/constants.ts:55:https://per.co.na/PMM3_forums
ui/apps/pmm/src/pages/help-center/HelpCenter.constants.ts:123:https://per.co.na/pmm3_feedback
ui/apps/pmm/src/pages/help-center/HelpCenter.constants.ts:33:https://per.co.na/pmm_documentation
ui/apps/pmm/src/pages/help-center/HelpCenter.constants.ts:48:https://www.percona.com/about/contact?utm_campaign=7075599-Product
ui/apps/pmm/src/pages/help-center/HelpCenter.constants.ts:63:https://per.co.na/PMM3_forum
ui/apps/pmm/src/pages/rta/selection/RealtimeSelection.constants.ts:4:https://per.co.na/QAN
ui/apps/pmm/src/pages/rta/selection/RealtimeSelection.constants.ts:5:https://per.co.na/PMM3_forums
ui/apps/pmm/src/pages/settings/components/advanced/Advanced.constants.ts:29:https://per.co.na/pmm-feature-status
ui/apps/pmm/src/pages/settings/Settings.messages.ts:19:https://per.co.na/data_retention
ui/apps/pmm/src/pages/settings/Settings.messages.ts:21:https://per.co.na/telemetry
ui/apps/pmm/src/pages/settings/Settings.messages.ts:28:https://per.co.na/updates
ui/apps/pmm/src/pages/settings/Settings.messages.ts:37:https://per.co.na/advisors
ui/apps/pmm/src/pages/settings/Settings.messages.ts:43:https://per.co.na/azure_monitoring
ui/apps/pmm/src/pages/settings/Settings.messages.ts:47:https://per.co.na/roles_permissions
ui/apps/pmm/src/pages/settings/Settings.messages.ts:55:https://per.co.na/alerting
ui/apps/pmm/src/pages/settings/Settings.messages.ts:59:https://per.co.na/backup_management
ui/apps/pmm/src/pages/settings/Settings.messages.ts:63:https://per.co.na/qan-pmm-server
ui/apps/pmm/src/pages/settings/Settings.messages.ts:76:https://per.co.na/metrics_resolution
ui/apps/pmm/src/pages/settings/Settings.messages.ts:97:https://per.co.na/ssh_key
ui/apps/pmm/src/pages/updates/update-card/UpdateCard.constants.ts:2:https://per.co.na/pmm/upgrade-docker
ui/apps/pmm/src/pages/updates/update-card/UpdateCard.constants.ts:5:https://per.co.na/pmm/upgrade-podman
ui/apps/pmm/src/pages/updates/update-card/UpdateCard.constants.ts:7:https://per.co.na/pmm/upgrade-helm
ui/apps/pmm/src/utils/testUtils.tsx:38:https://per.co.na/pmm/3.0.0
```
