import { useState, useEffect, useMemo } from 'react'
import { useNatsStore } from '../services/nats'
import { getChecks } from '../services/api'
import ConnectionStatus from '../components/ConnectionStatus'
import MessageFeed from '../components/MessageFeed'
import './DashboardView.css'

export default function DashboardView() {
  const [apiChecks, setApiChecks] = useState<string[]>([])
  const [selectedCheck, setSelectedCheck] = useState('all')
  const [selectedResult, setSelectedResult] = useState<'all' | 'passed' | 'failed'>('all')
  
  const status = useNatsStore(state => state.status)
  const messages = useNatsStore(state => state.messages)
  const error = useNatsStore(state => state.error)
  const subscribedSubject = useNatsStore(state => state.subscribedSubject)
  const connect = useNatsStore(state => state.connect)
  const disconnect = useNatsStore(state => state.disconnect)
  const clearMessages = useNatsStore(state => state.clearMessages)

  const connectToNats = async () => {
    try {
      await connect()
    } catch (err) {
      console.error('Connection failed:', err)
    }
  }

  const handleDisconnect = async () => {
    await disconnect()
  }

  const getCheckFromSubject = (subject: string): string => {
    const parts = subject.split('.')
    if (parts.length >= 3 && parts[0] === 'results') {
      return parts.slice(2).join('.').replace(/\.(0|1)$/, '')
    }
    return subject
  }

  const getPassedFromMessage = (payload: string, subject: string): boolean | null => {
    try {
      const parsed = JSON.parse(payload)
      if (typeof parsed.passed === 'boolean') {
        return parsed.passed
      }
    } catch {
      // Fall back to subject suffix if payload is not JSON or passed is missing.
    }

    if (subject.endsWith('.1')) return true
    if (subject.endsWith('.0')) return false
    return null
  }

  const filteredMessages = useMemo(() => {
    return messages.filter(msg => {
      if (selectedCheck !== 'all' && getCheckFromSubject(msg.subject) !== selectedCheck) {
        return false
      }

      if (selectedResult !== 'all') {
        const passed = getPassedFromMessage(msg.payload, msg.subject)
        if (selectedResult === 'passed' && passed !== true) {
          return false
        }
        if (selectedResult === 'failed' && passed !== false) {
          return false
        }
      }

      return true
    })
  }, [messages, selectedCheck, selectedResult])

  const availableChecks = useMemo(() => {
    const checksFromMessages = messages.map(msg => getCheckFromSubject(msg.subject))
    return Array.from(new Set([...apiChecks, ...checksFromMessages])).sort((a, b) =>
      a.localeCompare(b)
    )
  }, [messages, apiChecks])

  useEffect(() => {
    connectToNats()
  }, [])

  useEffect(() => {
    const loadChecks = async () => {
      try {
        const names = await getChecks()
        setApiChecks(names)
      } catch (err) {
        console.error('Failed to load checks for dashboard filters:', err)
      }
    }

    loadChecks()
  }, [])

  return (
    <div className="dashboard-view">
      <div className="view-header">
        <ConnectionStatus status={status} error={error} />
        <div className="header-actions">
          {status === 'connected' ? (
            <button className="btn btn--outline" onClick={handleDisconnect}>
              Disconnect
            </button>
          ) : status !== 'connecting' ? (
            <button className="btn btn--primary" onClick={connectToNats}>
              Connect
            </button>
          ) : null}
        </div>
      </div>

      <div className="dashboard-content">
        <aside className="sidebar">
          <div className="subscribe-section">
            <h3 className="section-title">Message Filters</h3>
            <div className="subscribe-form">
              <select
                className="filter-select"
                value={selectedCheck}
                onChange={e => setSelectedCheck(e.target.value)}
                disabled={availableChecks.length === 0}
              >
                <option value="all">All Checks</option>
                {availableChecks.map(check => (
                  <option key={check} value={check}>
                    {check}
                  </option>
                ))}
              </select>
              <select
                className="filter-select"
                value={selectedResult}
                onChange={e => setSelectedResult(e.target.value as 'all' | 'passed' | 'failed')}
              >
                <option value="all">All Results</option>
                <option value="passed">Passed</option>
                <option value="failed">Failed</option>
              </select>
            </div>
          </div>

          <div className="subscriptions-section">
            <h3 className="section-title">Active Subscription</h3>
            {!subscribedSubject ? (
              <div className="no-subs">
                No active subscription
              </div>
            ) : (
              <div className="subscription-item">
                <span className="sub-subject">{subscribedSubject}</span>
                <span className="stream-badge">JetStream</span>
              </div>
            )}
          </div>
        </aside>

        <main className="main-content">
          <MessageFeed messages={filteredMessages} onClear={clearMessages} />
        </main>
      </div>
    </div>
  )
}
