import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { isAuthenticated, isAdmin, isObserver } from '../services/auth'

export default function ProtectedRoute() {
  const location = useLocation()

  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }

  const role = isAdmin() ? 'admin' : isObserver() ? 'observer' : 'team'
  const path = location.pathname
  const inAdminSection = path.startsWith('/admin')
  const inObserverSection = path.startsWith('/observer')

  // Redirect admin users to /admin if they try to access non-admin pages
  if (role === 'admin' && !inAdminSection) {
    return <Navigate to="/admin" replace />
  }

  // Redirect observer users to /observer if they try to access other pages
  if (role === 'observer' && !inObserverSection) {
    return <Navigate to="/observer" replace />
  }

  // Redirect team users away from admin/observer-only routes
  if (role === 'team' && (inAdminSection || inObserverSection)) {
    return <Navigate to="/" replace />
  }

  return <Outlet />
}
