import { NavLink, useLocation, useNavigate } from 'react-router-dom'
import { clearCredentials, getCredentials, getTeamIdFromJwt, isAdmin } from '../services/auth'
import './NavBar.css'

export default function NavBar() {
  const location = useLocation()
  const navigate = useNavigate()

  // Get team ID from JWT
  const creds = getCredentials()
  const teamId = creds ? getTeamIdFromJwt(creds.jwt) : null
  const userIsAdmin = isAdmin()

  const handleLogout = () => {
    clearCredentials()
    navigate('/login', { replace: true })
  }

  return (
    <nav className="navbar">
      <div className="nav-brand">
        <h1 className="logo">NATS Dashboard</h1>
      </div>
      
      <div className="nav-links">
        {userIsAdmin ? (
          <NavLink 
            to="/admin" 
            className={`nav-link ${location.pathname === '/admin' ? 'active' : ''}`}
          >
            <span className="nav-icon">🔧</span>
            Admin
          </NavLink>
        ) : (
          <>
            <NavLink 
              to="/" 
              className={`nav-link ${location.pathname === '/' ? 'active' : ''}`}
            >
              <span className="nav-icon">📊</span>
              Messages
            </NavLink>
            <NavLink 
              to="/health" 
              className={`nav-link ${location.pathname === '/health' ? 'active' : ''}`}
            >
              <span className="nav-icon">💚</span>
              Health Check
            </NavLink>
            <NavLink 
              to="/settings" 
              className={`nav-link ${location.pathname === '/settings' ? 'active' : ''}`}
            >
              <span className="nav-icon">⚙️</span>
              Settings
            </NavLink>
          </>
        )}
      </div>

      <div className="nav-user">
        {teamId && <span className="team-id">{userIsAdmin ? 'Admin' : `Team ${teamId}`}</span>}
        <button className="logout-btn" onClick={handleLogout}>
          Logout
        </button>
      </div>
    </nav>
  )
}
