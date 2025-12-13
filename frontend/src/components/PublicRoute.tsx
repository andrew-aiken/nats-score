import { Navigate } from 'react-router-dom'
import { isAuthenticated } from '../services/auth'

interface PublicRouteProps {
  children: React.ReactNode
}

/**
 * Wrapper for public routes (like login) that redirects 
 * authenticated users to the dashboard
 */
export default function PublicRoute({ children }: PublicRouteProps) {
  if (isAuthenticated()) {
    return <Navigate to="/" replace />
  }

  return <>{children}</>
}

