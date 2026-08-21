import { NavLink, useLocation, useNavigate } from 'react-router-dom'
import { clearCredentials, getCredentials, getTeamIdFromJwt, isAdmin, isObserver } from '../services/auth'
import { useNatsStore } from '../services/nats'
import './NavBar.css'

export default function NavBar() {
  const location = useLocation()
  const navigate = useNavigate()
  const disconnect = useNatsStore(state => state.disconnect)

  // Get team ID from JWT
  const creds = getCredentials()
  const teamId = creds ? getTeamIdFromJwt(creds.jwt) : null
  const userIsAdmin = isAdmin()
  const userIsObserver = isObserver()

  const handleLogout = async () => {
    await disconnect()
    clearCredentials()
    navigate('/login', { replace: true })
  }

  return (
    <nav className="navbar">
      <div className="nav-brand">
        <h1 className="logo">Score Dashboard</h1>
      </div>

      <div className="nav-links">
        {userIsAdmin ? (
          <>
            <NavLink
              to="/admin/scores"
              className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}
            >
              <span className="nav-icon">📈</span>
              Scores
            </NavLink>
            <NavLink
              to="/admin"
              end
              className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}
            >
              <span className="nav-icon">🔧</span>
              Admin
            </NavLink>
          </>
        ) : userIsObserver ? (
          <NavLink
            to="/observer"
            className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}
          >
            <span className="nav-icon">📈</span>
            Scores
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
        {teamId && (
          <span className="team-id">
            {userIsAdmin ? 'Admin' : userIsObserver ? 'Observer' : `Team ${teamId}`}
          </span>
        )}
        <button className="logout-btn" onClick={handleLogout}>
          Logout
        </button>
      </div>
    </nav>
  )
}
