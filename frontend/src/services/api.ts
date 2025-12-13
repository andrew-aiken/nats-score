import { getCredentials } from './auth'

const API_SERVER = 'http://localhost:3000'

/**
 * Fetch mutable fields for all checks from the server
 * Returns a map of check names to their mutable field arrays
 */
export async function getMutableFields(): Promise<Record<string, string[]>> {
  const creds = getCredentials()
  if (!creds) {
    throw new Error('Not authenticated')
  }

  const response = await fetch(`${API_SERVER}/api/checks/mutable-fields`, {
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${creds.jwt}`,
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Failed to fetch mutable fields' }))
    throw new Error(error.error || 'Failed to fetch mutable fields')
  }

  return response.json()
}

/**
 * Fetch team settings via the server API
 * The team number is determined from the user's JWT role on the server
 */
export async function getTeamSettings(): Promise<Record<string, Record<string, string>>> {
  const creds = getCredentials()
  if (!creds) {
    throw new Error('Not authenticated')
  }

  const response = await fetch(`${API_SERVER}/api/settings`, {
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${creds.jwt}`,
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Failed to fetch settings' }))
    throw new Error(error.error || 'Failed to fetch settings')
  }

  return response.json()
}

/**
 * Update team settings via the server API
 * The team number is determined from the user's JWT role on the server
 * @param settings - Map of check names to field values
 */
export async function updateTeamSettings(settings: Record<string, Record<string, string>>): Promise<{ success: boolean; team: string }> {
  const creds = getCredentials()
  if (!creds) {
    throw new Error('Not authenticated')
  }

  const response = await fetch(`${API_SERVER}/api/settings`, {
    method: 'PUT',
    headers: {
      'Authorization': `Bearer ${creds.jwt}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(settings),
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Failed to update settings' }))
    throw new Error(error.error || 'Failed to update settings')
  }

  return response.json()
}

