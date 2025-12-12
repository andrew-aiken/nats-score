import { NavLink, useLocation } from 'react-router-dom'
import './NavBar.css'

export default function NavBar() {
  const location = useLocation()

  return (
    <nav className="navbar">
      <div className="nav-brand">
        <h1 className="logo">NATS Dashboard</h1>
      </div>
      
      <div className="nav-links">
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
      </div>
    </nav>
  )
}
