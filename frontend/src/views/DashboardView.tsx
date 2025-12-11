import { useState, useEffect } from 'react'
import { useNatsStore } from '../services/nats'
import ConnectionStatus from '../components/ConnectionStatus'
import MessageFeed from '../components/MessageFeed'
import './DashboardView.css'

export default function DashboardView() {
  const [subjectInput, setSubjectInput] = useState('')
  
  const status = useNatsStore(state => state.status)
  const messages = useNatsStore(state => state.messages)
  const error = useNatsStore(state => state.error)
  const subscribedSubject = useNatsStore(state => state.subscribedSubject)
  const connect = useNatsStore(state => state.connect)
  const disconnect = useNatsStore(state => state.disconnect)
  const subscribeToStream = useNatsStore(state => state.subscribeToStream)
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

  const changeSubscription = async () => {
    const subject = subjectInput.trim()
    if (!subject) return
    
    try {
      await subscribeToStream(subject)
      setSubjectInput('')
    } catch (err) {
      console.error('Subscribe failed:', err)
    }
  }

  const handleKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter') {
      changeSubscription()
    }
  }

  useEffect(() => {
    connectToNats()
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
            <h3 className="section-title">Change Subject Filter</h3>
            <div className="subscribe-form">
              <input 
                value={subjectInput}
                onChange={(e) => setSubjectInput(e.target.value)}
                type="text" 
                className="subject-input"
                placeholder="Enter subject (e.g., results.>)"
                disabled={status !== 'connected'}
                onKeyPress={handleKeyPress}
              />
              <button 
                className="btn btn--primary" 
                disabled={status !== 'connected' || !subjectInput.trim()}
                onClick={changeSubscription}
              >
                Change
              </button>
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
          <MessageFeed messages={messages} onClear={clearMessages} />
        </main>
      </div>
    </div>
  )
}
