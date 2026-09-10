import type { NatsCredentials } from '../types'

const STORAGE_KEY = 'nats_credentials'
const AUTH_SERVER = 'https://localhost'

/**
 * Login using a username and password
 * Returns credentials on success, throws error on failure
 */
export async function login(username: string, password: string): Promise<NatsCredentials> {
  const response = await fetch(`${AUTH_SERVER}/auth/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ username, password }),
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Authentication failed' }))
    throw new Error(error.error || 'Authentication failed')
  }

  const credentials: NatsCredentials = await response.json()
  saveCredentials(credentials)
  return credentials
}

/**
 * Get cached credentials from localStorage
 * Returns null if no credentials or if expired
 */
export function getCredentials(): NatsCredentials | null {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (!stored) {
    return null
  }

  try {
    const credentials: NatsCredentials = JSON.parse(stored)
    // console.log(foo(credentials.jwt))
    if (isTokenExpired(credentials.jwt)) {
      clearCredentials()
      return null
    }
    return credentials
  } catch {
    clearCredentials()
    return null
  }
}

/**
 * Extract team ID from JWT's name claim
 * Returns null if token is invalid or name claim is missing
 */
export function getTeamIdFromJwt(jwt: string): string | null {
  try {
    const parts = jwt.split('.')
    if (parts.length !== 3) return null
    const payload = parts[1]
    const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/'))
    const claims = JSON.parse(decoded)
    return claims.name || null
  } catch {
    return null
  }
}

/**
 * Decode JWT and check exp claim
 * Returns true if token is expired or invalid
 */
export function isTokenExpired(jwt: string): boolean {
  try {
    // JWT format: header.payload.signature
    const parts = jwt.split('.')
    if (parts.length !== 3) {
      return true
    }

    // Decode the payload (base64url)
    const payload = parts[1]
    const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/'))
    const claims = JSON.parse(decoded)

    // Check exp claim
    if (!claims.exp) {
      // No expiration claim - consider valid
      return false
    }

    // exp is in seconds, Date.now() is in milliseconds
    const expirationTime = claims.exp * 1000
    const now = Date.now()

    // Add a 30 second buffer to handle clock skew
    return now >= expirationTime - 30000
  } catch {
    // If we can't decode the token, consider it expired
    return true
  }
}

/**
 * Save credentials to localStorage
 */
export function saveCredentials(credentials: NatsCredentials): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(credentials))
}

/**
 * Clear stored credentials from localStorage
 */
export function clearCredentials(): void {
  localStorage.removeItem(STORAGE_KEY)
}

/**
 * Check if user is authenticated (has valid credentials)
 */
export function isAuthenticated(): boolean {
  return getCredentials() !== null
}

/**
 * Check if the current user is an admin based on JWT name claim
 * Admin users have the name claim set to "admin"
 */
export function isAdmin(): boolean {
  const creds = getCredentials()
  if (!creds) {
    return false
  }
  const teamId = getTeamIdFromJwt(creds.jwt)
  return teamId === 'admin'
}

/**
 * Check if the current user is an observer based on JWT name claim
 * Observers have the name claim set to "observer"
 */
export function isObserver(): boolean {
  const creds = getCredentials()
  if (!creds) {
    return false
  }
  const teamId = getTeamIdFromJwt(creds.jwt)
  return teamId === 'observer'
}
