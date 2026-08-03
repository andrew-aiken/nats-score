import { useState, useEffect, useMemo } from 'react'
import { startScoring, stopScoring, getGlobalSettings } from '../services/api'
import { useNatsStore } from '../services/nats'
import { toast } from '../services/toast'
import './AdminView.css'

interface TeamStats {
  team: string
  checks: {
    [checkName: string]: {
      total: number
      passed: number
      percentage: number
    }
  }
  overall: {
    total: number
    passed: number
    percentage: number
  }
}

interface GlobalCheck {
  name: string
  type: string
  description: string
  mutableFields: string[] | null
  scoreWeight: number
  definition: Record<string, unknown>
}

// Parse subject to extract team and check name
// Format: results.<team>.<check>
const parseSubject = (subject: string): { team: string; check: string } | null => {
  const parts = subject.split('.')
  if (parts.length >= 3 && parts[0] === 'results') {
    return {
      team: parts[1],
      check: parts.slice(2).join('.')
    }
  }
  return null
}

export default function AdminView() {
  const [globalSettings, setGlobalSettings] = useState<Record<string, GlobalCheck> | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState<'start' | 'stop' | null>(null)
  const [expandedTeams, setExpandedTeams] = useState<Set<string>>(new Set())
  const [expandedChecks, setExpandedChecks] = useState<Set<string>>(new Set())

  const status = useNatsStore(state => state.status)
  const messages = useNatsStore(state => state.messages)
  const connect = useNatsStore(state => state.connect)
  const subscribeToStream = useNatsStore(state => state.subscribeToStream)

  // Connect to NATS and subscribe to all teams' results
  useEffect(() => {
    const connectAndSubscribe = async () => {
      if (status === 'disconnected') {
        await connect()
      }
    }
    connectAndSubscribe()
  }, [])

  // Subscribe to all teams' results once connected
  useEffect(() => {
    if (status === 'connected') {
      subscribeToStream('results.>')
    }
  }, [status])

  // Fetch global settings on mount
  useEffect(() => {
    const fetchSettings = async () => {
      setLoading(true)
      try {
        const settings = await getGlobalSettings()
        setGlobalSettings(settings)
      } catch (err) {
        console.error('Failed to fetch global settings:', err)
        toast.error('Failed to load global settings', err instanceof Error ? err.message : 'Unknown error')
      } finally {
        setLoading(false)
      }
    }

    fetchSettings()
  }, [])

  // Calculate team statistics from NATS messages
  const teamStats = useMemo<TeamStats[]>(() => {
    const teamMap = new Map<string, TeamStats>()

    for (const msg of messages) {
      const parsed = parseSubject(msg.subject)
      if (!parsed) continue

      try {
        const data = JSON.parse(msg.payload)
        const passed = data.passed === true

        let stats = teamMap.get(parsed.team)
        if (!stats) {
          stats = {
            team: parsed.team,
            checks: {},
            overall: { total: 0, passed: 0, percentage: 0 }
          }
          teamMap.set(parsed.team, stats)
        }

        // Update check-specific stats
        if (!stats.checks[parsed.check]) {
          stats.checks[parsed.check] = { total: 0, passed: 0, percentage: 0 }
        }
        stats.checks[parsed.check].total++
        if (passed) {
          stats.checks[parsed.check].passed++
        }
        stats.checks[parsed.check].percentage =
          (stats.checks[parsed.check].passed / stats.checks[parsed.check].total) * 100

        // Update overall stats
        stats.overall.total++
        if (passed) {
          stats.overall.passed++
        }
        stats.overall.percentage = (stats.overall.passed / stats.overall.total) * 100
      } catch {
        // Skip invalid payloads
      }
    }

    // Sort teams naturally (team1, team2, etc.)
    return Array.from(teamMap.values()).sort((a, b) =>
      a.team.localeCompare(b.team, undefined, { numeric: true })
    )
  }, [messages])

  const toggleTeamExpanded = (team: string) => {
    setExpandedTeams(prev => {
      const next = new Set(prev)
      if (next.has(team)) {
        next.delete(team)
      } else {
        next.add(team)
      }
      return next
    })
  }

  const toggleCheckExpanded = (checkName: string) => {
    setExpandedChecks(prev => {
      const next = new Set(prev)
      if (next.has(checkName)) {
        next.delete(checkName)
      } else {
        next.add(checkName)
      }
      return next
    })
  }

  const sortedChecks = useMemo(
    () => Object.entries(globalSettings ?? {}).sort(([a], [b]) => a.localeCompare(b)),
    [globalSettings]
  )

  const getPercentageColor = (percentage: number): string => {
    if (percentage >= 80) return 'var(--success-color, #10b981)'
    if (percentage >= 50) return 'var(--warning-color, #f59e0b)'
    return 'var(--error-color, #ef4444)'
  }

  const handleStart = async () => {
    setActionLoading('start')
    try {
      await startScoring()
      toast.success('Scoring started', 'Successfully started the scoring system')
    } catch (err) {
      console.error('Failed to start scoring:', err)
      toast.error('Failed to start scoring', err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setActionLoading(null)
    }
  }

  const handleStop = async () => {
    setActionLoading('stop')
    try {
      await stopScoring()
      toast.success('Scoring stopped', 'Successfully stopped the scoring system')
    } catch (err) {
      console.error('Failed to stop scoring:', err)
      toast.error('Failed to stop scoring', err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setActionLoading(null)
    }
  }

  return (
    <div className="admin-view">
      <div className="admin-header">
        <h2>Admin Panel</h2>
      </div>

      <div className="admin-controls">
        <h3>Scoring Controls</h3>
        <div className="control-buttons">
          <button 
            className="btn btn-start" 
            onClick={handleStart}
            disabled={actionLoading !== null}
          >
            {actionLoading === 'start' ? 'Starting...' : 'Start Scoring'}
          </button>
          <button 
            className="btn btn-stop" 
            onClick={handleStop}
            disabled={actionLoading !== null}
          >
            {actionLoading === 'stop' ? 'Stopping...' : 'Stop Scoring'}
          </button>
        </div>
      </div>

      <div className="admin-stats">
        <h3>Team Check Statistics</h3>
        <p className="stats-info">
          {status === 'connected' 
            ? `Live data from ${messages.length} check results`
            : status === 'connecting'
            ? 'Connecting to NATS...'
            : 'Disconnected from NATS'}
        </p>
        {teamStats.length > 0 ? (
          <div className="team-stats-grid">
            {teamStats.map(stat => (
              <div key={stat.team} className="team-stat-card">
                <div 
                  className="team-stat-header"
                  onClick={() => toggleTeamExpanded(stat.team)}
                >
                  <div className="team-info">
                    <span className="team-name">{stat.team}</span>
                    <span className="expand-icon">
                      {expandedTeams.has(stat.team) ? '▼' : '▶'}
                    </span>
                  </div>
                  <div className="team-overall">
                    <div 
                      className="percentage-badge"
                      style={{ backgroundColor: getPercentageColor(stat.overall.percentage) }}
                    >
                      {stat.overall.percentage.toFixed(1)}%
                    </div>
                    <span className="check-count">
                      {stat.overall.passed}/{stat.overall.total} checks
                    </span>
                  </div>
                </div>
                
                {expandedTeams.has(stat.team) && (
                  <div className="team-checks">
                    {Object.entries(stat.checks)
                      .sort(([a], [b]) => a.localeCompare(b))
                      .map(([checkName, checkStat]) => (
                        <div key={checkName} className="check-row">
                          <span className="check-name">{checkName}</span>
                          <div className="check-stats">
                            <div className="progress-bar">
                              <div 
                                className="progress-fill"
                                style={{ 
                                  width: `${checkStat.percentage}%`,
                                  backgroundColor: getPercentageColor(checkStat.percentage)
                                }}
                              />
                            </div>
                            <span 
                              className="check-percentage"
                              style={{ color: getPercentageColor(checkStat.percentage) }}
                            >
                              {checkStat.percentage.toFixed(1)}%
                            </span>
                            <span className="check-count-small">
                              ({checkStat.passed}/{checkStat.total})
                            </span>
                          </div>
                        </div>
                      ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        ) : (
          <div className="no-stats">No check results available</div>
        )}
      </div>

      <div className="admin-settings">
        <h3>Global Settings</h3>
        {loading ? (
          <div className="loading">Loading settings...</div>
        ) : sortedChecks.length > 0 ? (
          <div className="check-settings-list">
            {sortedChecks.map(([checkName, check]) => (
              <div key={checkName} className="check-setting-card">
                <div
                  className="check-setting-header"
                  onClick={() => toggleCheckExpanded(checkName)}
                >
                  <div className="check-setting-info">
                    <span className="expand-icon">
                      {expandedChecks.has(checkName) ? '▼' : '▶'}
                    </span>
                    <span className="check-setting-name">{checkName}</span>
                  </div>
                  <div className="check-setting-badges">
                    <span className="check-setting-badge">{check.type}</span>
                    <span className="check-setting-badge check-setting-badge--weight">
                      {check.scoreWeight} pts
                    </span>
                  </div>
                </div>

                {expandedChecks.has(checkName) && (
                  <div className="check-setting-body">
                    {check.description && (
                      <p className="check-setting-description">{check.description}</p>
                    )}
                    <div className="check-setting-field">
                      <span className="check-setting-field-label">Mutable fields</span>
                      <span className="check-setting-field-value">
                        {check.mutableFields && check.mutableFields.length > 0
                          ? check.mutableFields.join(', ')
                          : 'None'}
                      </span>
                    </div>
                    <div className="check-setting-field">
                      <span className="check-setting-field-label">Definition</span>
                      <pre className="check-setting-definition">
                        {JSON.stringify(check.definition, null, 2)}
                      </pre>
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        ) : (
          <div className="no-settings">No global settings available</div>
        )}
      </div>
    </div>
  )
}
