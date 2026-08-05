import React from 'react';
import { Navigate, createBrowserRouter } from 'react-router-dom';
import { Settings } from 'pages/settings';
import { Updates } from 'pages/updates';
import { UpdateClients } from 'pages/update-clients/UpdateClients';
import { MainWithNav } from 'components/main/MainWithNav';
import { NotFoundPage } from 'pages/not-found';
import { HelpCenter } from 'pages/help-center';
import Providers from 'Providers';
import { PMM_NEW_NAV_PATH, UPDATES_ENABLED } from 'lib/constants';
import { Redirect, SettingsRedirect } from 'components/redirect';
import { AlertsPage } from 'pages/alerting/status';

const router = createBrowserRouter(
  [
    {
      path: '',
      element: <Providers />,
      children: [
        {
          path: PMM_NEW_NAV_PATH,
          element: <MainWithNav />,
          children: [
            {
              path: '',
              element: <Navigate to="graph" />,
            },
            ...(UPDATES_ENABLED
              ? [
                  {
                    path: 'updates',
                    element: <Updates />,
                  },
                  {
                    path: 'updates/clients',
                    element: <UpdateClients />,
                  },
                ]
              : []),
            {
              path: 'help',
              element: <HelpCenter />,
            },
            {
              path: 'alerting',
              children: [
                {
                  path: 'status',
                  element: <AlertsPage />,
                },
              ],
            },
            {
              path: 'settings/:tab?',
              element: <Settings />,
            },
            // Fallback
            {
              path: 'graph/settings/:tab?',
              element: <SettingsRedirect />,
            },
            {
              path: 'graph/alerting/alerts',
              element: <Navigate to="/alerting/status" replace />,
            },
            // Grafana routes are handled at the Main component level
            {
              path: 'graph/*',
              element: <React.Fragment />,
            },
            {
              path: '*',
              element: <NotFoundPage />,
            },
          ],
        },
        // Provide fallback for /next/* paths to redirect to the root path
        {
          path: '/next/*',
          element: <Redirect />,
        },
        {
          path: '*',
          element: <div>Not found!</div>,
        },
      ],
    },
  ],
  {
    basename: '/pfm-ui',
  }
);

export default router;
