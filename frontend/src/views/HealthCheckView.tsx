import { useState, useMemo, useEffect, useRef } from 'react'
import { useNatsStore } from '../services/nats'
import { toast } from '../services/toast'
import ConnectionStatus from '../components/ConnectionStatus'
import TimeRangeSelector from '../components/TimeRangeSelector'
import HealthTimeline from '../components/HealthTimeline'
import type { HealthCheck } from '../components/HealthTimeline'
import './HealthCheckView.css'


// Parse subject to extract service key (team_id.service_name)
const parseSubjectToKey = (subject: string): string => {
  const parts = subject.split('.')
  if (parts.length >= 3 && parts[0] === 'results') {
    return `${parts[1]}.${parts.slice(2).join('.')}`
  }
  return subject
}

// Get a friendly service name from subject, using checkNames lookup if available
const getServiceName = (subject: string): string => {
  const parts = subject.split('.')
  if (parts.length >= 3 && parts[0] === 'results') {
    const checkKey = parts.slice(2).join('.')
    return checkKey
  }
  return subject
}

export default function HealthCheckView() {
  const status = useNatsStore(state => state.status)
  const messages = useNatsStore(state => state.messages)
  const error = useNatsStore(state => state.error)
  const connect = useNatsStore(state => state.connect)

  const [timeRange, setTimeRange] = useState(() => {
    return localStorage.getItem('healthcheck-time-range') || '1h'
  })
  
  // Persist time range selection to localStorage
  useEffect(() => {
    localStorage.setItem('healthcheck-time-range', timeRange)
  }, [timeRange])
  
  // Track last known status for each service to detect state transitions
  const serviceStatusRef = useRef<Map<string, boolean>>(new Map())
  // Track which check IDs we've already processed
  const processedCheckIdsRef = useRef<Set<string>>(new Set())
  // Track when page was loaded - only show toasts for checks after this time
  const pageLoadTimeRef = useRef<Date>(new Date())

  // Convert time range string to hours
  const getHoursFromRange = (range: string): number => {
    switch (range) {
      case '10m': return 10 / 60
      case '30m': return 0.5
      case '1h': return 1
      case '3h': return 3
      case 'all': return -1
      default: return 1
    }
  }

  // Parse ALL messages to track service status (not filtered by time range)
  const allHealthChecks = useMemo<HealthCheck[]>(() => {
    return messages
      .map(msg => {
        try {
          const data = JSON.parse(msg.payload)
          const timestamp = data.timestamp ? new Date(data.timestamp) : msg.timestamp
          
          return {
            id: msg.id,
            details: data.details,
            message: data.message || '',
            passed: data.passed === true,
            points: data.points || 0,
            timestamp,
            subject: msg.subject
          } as HealthCheck
        } catch {
          return null
        }
      })
      .filter((check): check is HealthCheck => check !== null)
      .sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime())
  }, [messages])

  // Filter by time range for display
  const healthChecks = useMemo<HealthCheck[]>(() => {
    const hours = getHoursFromRange(timeRange)
    const cutoff = hours > 0 ? new Date(Date.now() - hours * 60 * 60 * 1000) : new Date(0)
    
    return allHealthChecks.filter(check => check.timestamp >= cutoff)
  }, [allHealthChecks, timeRange])

  // Detect state transitions and show toasts
  useEffect(() => {
    if (allHealthChecks.length === 0) return

    for (const check of allHealthChecks) {
      // Skip if we've already processed this check
      if (processedCheckIdsRef.current.has(check.id)) {
        continue
      }
      
      // Mark as processed
      processedCheckIdsRef.current.add(check.id)
      
      const key = parseSubjectToKey(check.subject)
      const previousStatus = serviceStatusRef.current.get(key)
      
      // Only show toast if:
      // 1. The check happened after page load
      // 2. The previous check for this service was passing
      // 3. This check is failing
      const isAfterPageLoad = check.timestamp > pageLoadTimeRef.current
      if (isAfterPageLoad && previousStatus === true && check.passed === false) {
        toast.error(
          'Service Check Failed',
          `${getServiceName(check.subject)} is now failing`
        )
      }
      
      // Always update the tracked status (even for historical checks)
      serviceStatusRef.current.set(key, check.passed)
    }
  }, [allHealthChecks])

  const connectToNats = async () => {
    try {
      await connect()
    } catch (err) {
      console.error('Connection failed:', err)
    }
  }

  useEffect(() => {
    // Connect if not already connected
    if (status === 'disconnected') {
      connectToNats()
    }
  }, [])

  return (
    <div className="health-check-view">
      <div className="view-header">
        <div className="header-left">
          <ConnectionStatus status={status} error={error} />
        </div>
        <div className="header-right">
          <TimeRangeSelector value={timeRange} onChange={setTimeRange} />
        </div>
      </div>

      <div className="view-content">
        <HealthTimeline checks={healthChecks} timeRange={timeRange} />
      </div>
    </div>
  )
}
