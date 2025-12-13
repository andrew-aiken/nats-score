import { createBrowserRouter } from 'react-router-dom'
import App from '../App'
import ProtectedRoute from '../components/ProtectedRoute'
import PublicRoute from '../components/PublicRoute'
import DashboardView from '../views/DashboardView'
import HealthCheckView from '../views/HealthCheckView'
import SettingsView from '../views/SettingsView'
import AuthCallbackView from '../views/AuthCallbackView'
import LoginView from '../views/LoginView'

export const router = createBrowserRouter([
  {
    path: '/login',
    element: (
      <PublicRoute>
        <LoginView />
      </PublicRoute>
    )
  },
  {
    path: '/auth/callback',
    element: <AuthCallbackView />
  },
  {
    path: '/',
    element: <ProtectedRoute />,
    children: [
      {
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
    ]
  }
])
