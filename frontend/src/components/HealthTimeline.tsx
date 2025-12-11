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

export default function HealthTimeline({ checks }: HealthTimelineProps) {
  const [selectedCheck, setSelectedCheck] = useState<HealthCheck | null>(null)

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
      minute: '2-digit'
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
            {serviceGroups.map(service => (
              <div key={service.key} className="service-row">
                <div className="service-info">
                  <div className={`service-status ${service.lastStatus ? 'passed' : 'failed'}`}>
                    ●
                  </div>
                  <div className="service-meta">
                    <span className="service-name">{service.serviceName}</span>
                    <span className="service-team">Team {service.teamId}</span>
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
                  {service.checks.map(check => (
                    <div
                      key={check.id}
                      className={`timeline-item ${check.passed ? 'passed' : 'failed'} ${
                        selectedCheck?.id === check.id ? 'selected' : ''
                      }`}
                      title={`${check.passed ? 'Passed' : 'Failed'} - ${formatTime(check.timestamp)}`}
                      onClick={() => handleSelectCheck(check)}
                    >
                      <div className="item-indicator"></div>
                    </div>
                  ))}
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
