import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { handleCallback } from '../services/auth'
import './AuthCallbackView.css'

export default function AuthCallbackView() {
  const navigate = useNavigate()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const credentials = handleCallback()

    if (credentials) {
      // Successfully extracted credentials, redirect to dashboard
      navigate('/', { replace: true })
    } else {
      // Missing credentials in URL
      setError('Authentication failed. Missing credentials in callback.')
    }
  }, [navigate])

  if (error) {
    return (
      <div className="auth-callback">
        <div className="auth-callback-error">
          <h2>Authentication Error</h2>
          <p>{error}</p>
          <button onClick={() => navigate('/login', { replace: true })}>
            Try Again
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="auth-callback">
      <div className="auth-callback-loading">
        <div className="spinner"></div>
        <p>Completing authentication...</p>
      </div>
    </div>
  )
}
