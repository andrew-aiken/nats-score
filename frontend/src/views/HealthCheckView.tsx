import { useState, useMemo, useEffect } from 'react'
import { useNatsStore } from '../services/nats'
import ConnectionStatus from '../components/ConnectionStatus'
import TimeRangeSelector from '../components/TimeRangeSelector'
import HealthTimeline from '../components/HealthTimeline'
import type { HealthCheck } from '../components/HealthTimeline'
import './HealthCheckView.css'

export default function HealthCheckView() {
  const status = useNatsStore(state => state.status)
  const messages = useNatsStore(state => state.messages)
  const error = useNatsStore(state => state.error)
  const connect = useNatsStore(state => state.connect)

  const [timeRange, setTimeRange] = useState('24h')

  // Convert time range string to hours
  const getHoursFromRange = (range: string): number => {
    switch (range) {
      case '1h': return 1
      case '6h': return 6
      case '24h': return 24
      case '7d': return 168
      case 'all': return -1
      default: return 24
    }
  }

  // Parse messages into health check format and filter by time range
  const healthChecks = useMemo<HealthCheck[]>(() => {
    const hours = getHoursFromRange(timeRange)
    const cutoff = hours > 0 ? new Date(Date.now() - hours * 60 * 60 * 1000) : new Date(0)
    
    return messages
      .map(msg => {
        try {
          const data = JSON.parse(msg.payload)
          
          // Parse the timestamp from the message data
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
      .filter((check): check is HealthCheck => {
        if (!check) return false
        return check.timestamp >= cutoff
      })
      .sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime())
  }, [messages, timeRange])

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
        <HealthTimeline checks={healthChecks} />
      </div>
    </div>
  )
}
