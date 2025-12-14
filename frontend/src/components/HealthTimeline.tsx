import { useMemo, useState } from 'react'
import './HealthTimeline.css'

export interface HealthCheck {
  id: string
  details: unknown
  message: string
  passed: boolean
  points: number
  timestamp: Date
  subject: string
}

export interface ServiceGroup {
  key: string
  teamId: string
  serviceName: string
  checks: HealthCheck[]
  passRate: number
  lastStatus: boolean
}

interface HealthTimelineProps {
  checks: HealthCheck[]
  timeRange: string
  checkNames?: Record<string, string>
}

// Parse subject to extract team_id and service_name
// Format: results.<team_id>.<service_name>
const parseSubject = (subject: string): { teamId: string; serviceName: string } => {
  const parts = subject.split('.')
  if (parts.length >= 3 && parts[0] === 'results') {
    return {
      teamId: parts[1] ?? 'unknown',
      serviceName: parts.slice(2).join('.')
    }
  }
  return { teamId: 'unknown', serviceName: subject }
}

// Get time range configuration
const getTimeRangeConfig = (range: string) => {
  switch (range) {
    case '10m':
      return { minutes: 10, intervalMinutes: 1, labelFormat: 'minute' as const }
    case '30m':
      return { minutes: 30, intervalMinutes: 5, labelFormat: 'minute' as const }
    case '1h':
      return { minutes: 60, intervalMinutes: 10, labelFormat: 'minute' as const }
    case '3h':
      return { minutes: 180, intervalMinutes: 30, labelFormat: 'hour' as const }
    case 'all':
      return { minutes: -1, intervalMinutes: 60, labelFormat: 'hour' as const }
    default:
      return { minutes: 60, intervalMinutes: 10, labelFormat: 'minute' as const }
  }
}

export default function HealthTimeline({ checks, timeRange, checkNames = {} }: HealthTimelineProps) {
  const [selectedCheck, setSelectedCheck] = useState<HealthCheck | null>(null)

  // Get display name for a service (use name from settings if available, otherwise check key)
  const getDisplayName = (serviceName: string): string => {
    return checkNames[serviceName] || serviceName
  }

  const rangeConfig = getTimeRangeConfig(timeRange)

  // Calculate the time bounds for the timeline
  const timeBounds = useMemo(() => {
    const now = new Date()
    
    if (timeRange === 'all' && checks.length > 0) {
      // For "all", use the actual data range
      const timestamps = checks.map(c => c.timestamp.getTime())
      const minTime = Math.min(...timestamps)
      const maxTime = Math.max(...timestamps)
      // Add some padding
      const padding = (maxTime - minTime) * 0.05 || 60000
      return {
        start: new Date(minTime - padding),
        end: new Date(maxTime + padding)
      }
    }
    
    // For fixed ranges, show from (now - range) to now
    const rangeMs = rangeConfig.minutes * 60 * 1000
    return {
      start: new Date(now.getTime() - rangeMs),
      end: now
    }
  }, [timeRange, checks, rangeConfig.minutes])

  // Generate time markers for the timeline
  const timeMarkers = useMemo(() => {
    const markers: { position: number; label: string }[] = []
    const totalMs = timeBounds.end.getTime() - timeBounds.start.getTime()
    const intervalMs = rangeConfig.intervalMinutes * 60 * 1000
    
    // Round start time to the nearest interval
    const startMs = timeBounds.start.getTime()
    const firstMarkerMs = Math.ceil(startMs / intervalMs) * intervalMs
    
    for (let ms = firstMarkerMs; ms <= timeBounds.end.getTime(); ms += intervalMs) {
      const position = ((ms - startMs) / totalMs) * 100
      const date = new Date(ms)
      
      let label: string
      if (rangeConfig.labelFormat === 'minute') {
        label = date.toLocaleTimeString('en-US', { 
          hour: '2-digit', 
          minute: '2-digit',
          hour12: false 
        })
      } else {
        label = date.toLocaleTimeString('en-US', { 
          hour: '2-digit', 
          minute: '2-digit',
          hour12: false 
        })
      }
      
      markers.push({ position, label })
    }
    
    return markers
  }, [timeBounds, rangeConfig])

  // Group checks by service
  const serviceGroups = useMemo<ServiceGroup[]>(() => {
    const groups = new Map<string, HealthCheck[]>()
    
    for (const check of checks) {
      const { teamId, serviceName } = parseSubject(check.subject)
      const key = `${teamId}.${serviceName}`
      
      if (!groups.has(key)) {
        groups.set(key, [])
      }
      groups.get(key)!.push(check)
    }
    
    // Convert to array and sort by service name
    return Array.from(groups.entries())
      .map(([key, checks]) => {
        const firstCheck = checks[0]
        if (!firstCheck) return null
        const { teamId, serviceName } = parseSubject(firstCheck.subject)
        const passed = checks.filter(c => c.passed).length
        const sortedChecks = [...checks].sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime())
        
        return {
          key,
          teamId,
          serviceName,
          checks: sortedChecks,
          passRate: Math.round((passed / checks.length) * 100),
          lastStatus: sortedChecks[sortedChecks.length - 1]?.passed ?? false
        }
      })
      .filter((g): g is ServiceGroup => g !== null)
      .sort((a, b) => {
        // Sort by team, then by service name
        if (a.teamId !== b.teamId) return a.teamId.localeCompare(b.teamId)
        return a.serviceName.localeCompare(b.serviceName)
      })
  }, [checks])

  // Calculate position for a check on the timeline (0-100%)
  const getCheckPosition = (check: HealthCheck): number => {
    const totalMs = timeBounds.end.getTime() - timeBounds.start.getTime()
    if (totalMs === 0) return 50
    const checkMs = check.timestamp.getTime() - timeBounds.start.getTime()
    return Math.max(0, Math.min(100, (checkMs / totalMs) * 100))
  }

  const formatTime = (date: Date): string => {
    return date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  }

  const formatDate = (date: Date): string => {
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  }

  const stats = useMemo(() => {
    const total = checks.length
    const passed = checks.filter(c => c.passed).length
    const failed = total - passed
    const passRate = total > 0 ? Math.round((passed / total) * 100) : 0
    const totalPoints = checks.reduce((sum, c) => sum + c.points, 0)
    const services = serviceGroups.length
    
    return { total, passed, failed, passRate, totalPoints, services }
  }, [checks, serviceGroups.length])

  const handleSelectCheck = (check: HealthCheck) => {
    setSelectedCheck(selectedCheck?.id === check.id ? null : check)
  }

  return (
    <div className="health-timeline">
      {/* Stats Summary */}
      <div className="stats-bar">
        <div className="stat">
          <span className="stat-value">{stats.services}</span>
          <span className="stat-label">Services</span>
        </div>
        <div className="stat">
          <span className="stat-value">{stats.total}</span>
          <span className="stat-label">Total Checks</span>
        </div>
        <div className="stat stat--success">
          <span className="stat-value">{stats.passed}</span>
          <span className="stat-label">Passed</span>
        </div>
        <div className="stat stat--error">
          <span className="stat-value">{stats.failed}</span>
          <span className="stat-label">Failed</span>
        </div>
        <div className="stat">
          <span className="stat-value">{stats.passRate}%</span>
          <span className="stat-label">Pass Rate</span>
        </div>
        <div className="stat stat--accent">
          <span className="stat-value">{stats.totalPoints}</span>
          <span className="stat-label">Total Points</span>
        </div>
      </div>

      {/* Services Grid */}
      <div className="services-container">
        {checks.length === 0 ? (
          <div className="empty-state">
            <div className="empty-icon">🔍</div>
            <p>No health checks in selected time range</p>
          </div>
        ) : (
          <div className="services-list">
            {/* Time axis header */}
            <div className="timeline-header">
              <div className="service-info-spacer"></div>
              <div className="time-axis">
                {timeMarkers.map((marker, idx) => (
                  <div 
                    key={idx} 
                    className="time-marker"
                    style={{ left: `${marker.position}%` }}
                  >
                    <div className="marker-line"></div>
                    <span className="marker-label">{marker.label}</span>
                  </div>
                ))}
              </div>
            </div>

            {serviceGroups.map(service => (
              <div key={service.key} className="service-row">
                <div className="service-info">
                  <div className={`service-status ${service.lastStatus ? 'passed' : 'failed'}`}>
                    ●
                  </div>
                  <div className="service-meta">
                    <span className="service-name">{getDisplayName(service.serviceName)}</span>
                  </div>
                  <div className="service-stats">
                    <span className={`service-pass-rate ${
                      service.passRate >= 80 ? 'good' : 
                      service.passRate >= 50 ? 'warning' : 'bad'
                    }`}>
                      {service.passRate}%
                    </span>
                  </div>
                </div>
                
                <div className="service-timeline">
                  <div className="timeline-track">
                    {/* Grid lines aligned with time markers */}
                    {timeMarkers.map((marker, idx) => (
                      <div 
                        key={idx}
                        className="timeline-grid-line"
                        style={{ left: `${marker.position}%` }}
                      />
                    ))}
                    
                    {/* Health check items */}
                    {service.checks.map(check => (
                      <div
                        key={check.id}
                        className={`timeline-item ${check.passed ? 'passed' : 'failed'} ${
                          selectedCheck?.id === check.id ? 'selected' : ''
                        }`}
                        style={{ left: `${getCheckPosition(check)}%` }}
                        title={`${check.passed ? 'Passed' : 'Failed'} - ${formatTime(check.timestamp)}`}
                        onClick={() => handleSelectCheck(check)}
                      >
                        <div className="item-indicator"></div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Selected Check Details */}
      {selectedCheck && (
        <div className="check-details">
          <div className="details-header">
            <div className={`details-status ${selectedCheck.passed ? 'passed' : ''}`}>
              {selectedCheck.passed ? '✓ Passed' : '✗ Failed'}
            </div>
            <button className="close-btn" onClick={() => setSelectedCheck(null)}>✕</button>
          </div>
          
          <div className="details-content">
            <div className="detail-row">
              <span className="detail-label">Subject</span>
              <span className="detail-value mono">{selectedCheck.subject}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Timestamp</span>
              <span className="detail-value">{formatDate(selectedCheck.timestamp)}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Points</span>
              <span className="detail-value">{selectedCheck.points}</span>
            </div>
            {selectedCheck.message && (
              <div className="detail-row">
                <span className="detail-label">Message</span>
                <span className="detail-value">{selectedCheck.message}</span>
              </div>
            )}
            {selectedCheck.details != null && (
              <div className="detail-row detail-row--full">
                <span className="detail-label">Details</span>
                <pre className="detail-value mono">{JSON.stringify(selectedCheck.details, null, 2)}</pre>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
