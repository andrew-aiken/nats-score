import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { isAuthenticated, isAdmin } from '../services/auth'

export default function ProtectedRoute() {
  const location = useLocation()
  
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }

  const userIsAdmin = isAdmin()
  const isAdminSection = location.pathname.startsWith('/admin')

  // Redirect admin users to /admin if they try to access non-admin pages
  if (userIsAdmin && !isAdminSection) {
    return <Navigate to="/admin" replace />
  }

  // Redirect non-admin users away from admin route
  if (!userIsAdmin && isAdminSection) {
    return <Navigate to="/" replace />
  }

  return <Outlet />
}

