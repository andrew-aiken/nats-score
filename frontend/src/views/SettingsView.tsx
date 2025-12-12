import { useState, useEffect } from 'react'
import { useNatsStore } from '../services/nats'
import { toast } from '../services/toast'
import ConnectionStatus from '../components/ConnectionStatus'
import './SettingsView.css'

// Check schema from settings.settings
interface CheckConfig {
  name?: string
  type: string
  description?: string
  score_weight?: number
  mutable_fields?: string[]
}

interface Settings {
  checks: Record<string, CheckConfig>
}

// User values from settings.1.settings
// Format: {"icmp":{"host":"10.9.9.9","username":"foobar"}}
type UserSettings = Record<string, Record<string, string>>

export default function SettingsView() {
  const status = useNatsStore(state => state.status)
  const error = useNatsStore(state => state.error)
  const connect = useNatsStore(state => state.connect)
  const getKvValue = useNatsStore(state => state.getKvValue)
  const putKvValue = useNatsStore(state => state.putKvValue)

  const [settings, setSettings] = useState<Settings | null>(null)
  const [fieldValues, setFieldValues] = useState<UserSettings>({})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState<string | null>(null)

  // Connect to NATS if not already connected
  useEffect(() => {
    if (status === 'disconnected') {
      connect().catch(err => console.error('Connection failed:', err))
    }
  }, [status, connect])

  // Fetch settings schema and user values when connected
  useEffect(() => {
    const fetchSettings = async () => {
      if (status !== 'connected') return

      setLoading(true)
      try {
        // Fetch check schema from settings.settings
        const schemaValue = await getKvValue('settings', 'settings')
        if (schemaValue) {
          const parsed = JSON.parse(schemaValue) as Settings
          setSettings(parsed)
          
          // Fetch user values from settings.1.settings
          try {
            const userValue = await getKvValue('settings', '1.settings')
            if (userValue) {
              const userSettings = JSON.parse(userValue) as UserSettings
              setFieldValues(userSettings)
            } else {
              // Initialize empty values for each check
              const initialValues: UserSettings = {}
              for (const [key, check] of Object.entries(parsed.checks)) {
                if (check.mutable_fields && check.mutable_fields.length > 0) {
                  initialValues[key] = {}
                  for (const field of check.mutable_fields) {
                    initialValues[key][field] = ''
                  }
                }
              }
              setFieldValues(initialValues)
            }
          } catch {
            // If 1.settings doesn't exist yet, initialize empty values
            const initialValues: UserSettings = {}
            for (const [key, check] of Object.entries(parsed.checks)) {
              if (check.mutable_fields && check.mutable_fields.length > 0) {
                initialValues[key] = {}
                for (const field of check.mutable_fields) {
                  initialValues[key][field] = ''
                }
              }
            }
            setFieldValues(initialValues)
          }
        }
      } catch (err) {
        console.error('Failed to fetch settings:', err)
        toast.error('Failed to load settings', err instanceof Error ? err.message : 'Unknown error')
      } finally {
        setLoading(false)
      }
    }

    fetchSettings()
  }, [status, getKvValue])

  const handleFieldChange = (checkKey: string, field: string, value: string) => {
    setFieldValues(prev => ({
      ...prev,
      [checkKey]: {
        ...prev[checkKey],
        [field]: value
      }
    }))
  }

  const handleSaveCheck = async (checkKey: string) => {
    if (!settings) return

    setSaving(checkKey)
    try {
      // Save to 1.settings with format: {"checkKey":{"field":"value"}}
      const updatedUserSettings: UserSettings = {
        ...fieldValues,
        [checkKey]: fieldValues[checkKey] || {}
      }

      await putKvValue('settings', '1.settings', JSON.stringify(updatedUserSettings))
      toast.success('Settings saved', `Updated ${settings.checks[checkKey].name || checkKey}`)
    } catch (err) {
      console.error('Failed to save settings:', err)
      toast.error('Failed to save settings', err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setSaving(null)
    }
  }

  const renderCheckCard = (checkKey: string, check: CheckConfig) => {
    const displayName = check.name || checkKey
    const mutableFields = check.mutable_fields || []
    const checkValues = fieldValues[checkKey] || {}

    return (
      <div key={checkKey} className="check-card">
        <div className="check-header">
          <h3 className="check-name">{displayName}</h3>
          {check.type && <span className="check-type">{check.type}</span>}
        </div>
        
        {check.description && (
          <p className="check-description">{check.description}</p>
        )}

        {mutableFields.length > 0 ? (
          <div className="check-fields">
            {mutableFields.map(field => (
              <div key={field} className="field-group">
                <label className="field-label" htmlFor={`${checkKey}-${field}`}>
                  {field}
                </label>
                <input
                  id={`${checkKey}-${field}`}
                  type="text"
                  className="field-input"
                  value={checkValues[field] || ''}
                  onChange={e => handleFieldChange(checkKey, field, e.target.value)}
                  placeholder={`Enter ${field}`}
                />
              </div>
            ))}
            
            <button 
              className="save-button"
              onClick={() => handleSaveCheck(checkKey)}
              disabled={saving === checkKey}
            >
              {saving === checkKey ? 'Saving...' : 'Save'}
            </button>
          </div>
        ) : (
          <p className="no-fields">No configurable fields</p>
        )}
      </div>
    )
  }

  return (
    <div className="settings-view">
      <div className="view-header">
        <div className="header-left">
          <ConnectionStatus status={status} error={error} />
        </div>
        <div className="header-right">
          <h2 className="page-title">Settings</h2>
        </div>
      </div>

      <div className="view-content">
        {loading ? (
          <div className="loading-state">
            <div className="loading-spinner"></div>
            <p>Loading settings...</p>
          </div>
        ) : settings ? (
          <div className="checks-grid">
            {Object.entries(settings.checks).map(([key, check]) => 
              renderCheckCard(key, check)
            )}
          </div>
        ) : (
          <div className="empty-state">
            <p>No settings found</p>
            <p className="empty-hint">Make sure the "settings" KV bucket exists with a "settings" key</p>
          </div>
        )}
      </div>
    </div>
  )
}
