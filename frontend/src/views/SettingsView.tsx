import { useState, useEffect } from 'react'
import { getMutableFields, getTeamSettings, updateTeamSettings } from '../services/api'
import { toast, useNotificationPreferenceStore } from '../services/toast'
import './SettingsView.css'

// Mutable fields map from the API
type MutableFieldsMap = Record<string, string[]>

// User values from team settings
// Format: {"icmp":{"host":"10.9.9.9","username":"foobar"}}
type UserSettings = Record<string, Record<string, string>>

const getSanitizedSettingsPayload = (settings: UserSettings): UserSettings => {
  const sanitizedSettings: UserSettings = {}

  for (const [checkKey, fields] of Object.entries(settings)) {
    const sanitizedFields = Object.fromEntries(
      Object.entries(fields).filter(([, value]) => value.trim() !== '')
    )

    if (Object.keys(sanitizedFields).length > 0) {
      sanitizedSettings[checkKey] = sanitizedFields
    }
  }

  return sanitizedSettings
}

export default function SettingsView() {
  const [mutableFields, setMutableFields] = useState<MutableFieldsMap | null>(null)
  const [fieldValues, setFieldValues] = useState<UserSettings>({})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const soundEnabled = useNotificationPreferenceStore(s => s.soundEnabled)
  const setSoundEnabled = useNotificationPreferenceStore(s => s.setSoundEnabled)

  // Fetch mutable fields and team settings from API on mount
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        // Fetch mutable fields schema
        const fields = await getMutableFields()
        setMutableFields(fields)

        // Fetch current team settings
        try {
          const settings = await getTeamSettings()
          if (settings && Object.keys(settings).length > 0) {
            setFieldValues(settings)
          } else {
            // Initialize empty values for each check with mutable fields
            const initialValues: UserSettings = {}
            for (const [key, fieldList] of Object.entries(fields)) {
              if (fieldList.length > 0) {
                initialValues[key] = {}
                for (const field of fieldList) {
                  initialValues[key][field] = ''
                }
              }
            }
            setFieldValues(initialValues)
          }
        } catch {
          // If settings fetch fails, initialize empty values
          const initialValues: UserSettings = {}
          for (const [key, fieldList] of Object.entries(fields)) {
            if (fieldList.length > 0) {
              initialValues[key] = {}
              for (const field of fieldList) {
                initialValues[key][field] = ''
              }
            }
          }
          setFieldValues(initialValues)
        }
      } catch (err) {
        console.error('Failed to fetch settings:', err)
        toast.error('Failed to load settings', err instanceof Error ? err.message : 'Unknown error')
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [])

  const handleFieldChange = (checkKey: string, field: string, value: string) => {
    setFieldValues(prev => ({
      ...prev,
      [checkKey]: {
        ...prev[checkKey],
        [field]: value
      }
    }))
  }

  const handleSaveAll = async () => {
    setSaving(true)
    try {
      // Save via API - team number is determined from JWT on server
      const updatedUserSettings = getSanitizedSettingsPayload(fieldValues)

      await updateTeamSettings(updatedUserSettings)
      toast.success('Settings saved', "")
    } catch (err) {
      console.error('Failed to save settings:', err)
      toast.error('Failed to save settings', err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setSaving(false)
    }
  }

  const renderCheckCard = (checkKey: string, fields: string[]) => {
    const checkValues = fieldValues[checkKey] || {}

    return (
      <div key={checkKey} className="check-card">
        <div className="check-header">
          <h3 className="check-name">{checkKey}</h3>
        </div>

        {fields.length > 0 ? (
          <div className="check-fields">
            {fields.map(field => (
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
          <label className="sound-toggle">
            <span className="sound-toggle-label">Notification sounds</span>
            <button
              type="button"
              role="switch"
              aria-checked={soundEnabled}
              className={`toggle-switch ${soundEnabled ? 'toggle-switch--on' : ''}`}
              onClick={() => setSoundEnabled(!soundEnabled)}
            >
              <span className="toggle-switch-thumb" />
            </button>
          </label>
        </div>
        <div className="header-right">
          <button className="save-button" onClick={handleSaveAll} disabled={saving}>
            {saving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>

      <div className="view-content">
        {loading ? (
          <div className="loading-state">
            <div className="loading-spinner"></div>
            <p>Loading settings...</p>
          </div>
        ) : mutableFields && Object.keys(mutableFields).length > 0 ? (
          <div className="checks-grid">
            {Object.entries(mutableFields).map(([key, fields]) =>
              renderCheckCard(key, fields)
            )}
          </div>
        ) : (
          <div className="empty-state">
            <p>No configurable checks found</p>
            <p className="empty-hint">No checks have mutable fields defined</p>
          </div>
        )}
      </div>
    </div>
  )
}
