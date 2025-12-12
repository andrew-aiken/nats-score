import { createBrowserRouter } from 'react-router-dom'
import App from '../App'
import DashboardView from '../views/DashboardView'
import HealthCheckView from '../views/HealthCheckView'
import SettingsView from '../views/SettingsView'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <App />,
    children: [
      {
        index: true,
        element: <DashboardView />
      },
      {
        path: 'health',
        element: <HealthCheckView />
      },
      {
        path: 'settings',
        element: <SettingsView />
      }
    ]
  }
])
