import { AgentUpdateSeverity, GetAgentVersionItem } from 'types/agent.types';

export const getAgentVersions = async (): Promise<GetAgentVersionItem[]> => [
  {
    agentId: 'watchtower-server',
    version: '3.0.0',
    nodeName: 'watchtower-server',
    severity: AgentUpdateSeverity.UP_TO_DATE,
  },
];
