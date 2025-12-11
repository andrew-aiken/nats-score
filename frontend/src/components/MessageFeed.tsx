import { useEffect, useRef, useState, useMemo } from 'react'
import type { NatsMessage } from '../types'
import './MessageFeed.css'

interface MessageFeedProps {
  messages: readonly NatsMessage[]
  onClear: () => void
}

export default function MessageFeed({ messages, onClear }: MessageFeedProps) {
  const feedContentRef = useRef<HTMLDivElement>(null)
  const [autoScroll, setAutoScroll] = useState(true)

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    if (autoScroll && feedContentRef.current) {
      feedContentRef.current.scrollTop = feedContentRef.current.scrollHeight
    }
  }, [messages.length, autoScroll])

  // Detect if user has scrolled up (disable auto-scroll)
  const handleScroll = () => {
    if (!feedContentRef.current) return
    const { scrollTop, scrollHeight, clientHeight } = feedContentRef.current
    // Re-enable auto-scroll if user scrolls to bottom
    setAutoScroll(scrollHeight - scrollTop - clientHeight < 100)
  }

  const formatTimestamp = (date: Date): string => {
    const timeStr = date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
    const ms = date.getMilliseconds().toString().padStart(3, '0')
    return `${timeStr}.${ms}`
  }

  const formatPayload = (payload: string): string => {
    try {
      const parsed = JSON.parse(payload)
      return JSON.stringify(parsed, null, 2)
    } catch {
      return payload
    }
  }

  const isJson = (payload: string): boolean => {
    try {
      JSON.parse(payload)
      return true
    } catch {
      return false
    }
  }

  const isEmpty = useMemo(() => messages.length === 0, [messages.length])

  return (
    <div className="message-feed">
      <div className="feed-header">
        <h2 className="feed-title">Live Messages</h2>
        <span className="message-count">{messages.length} messages</span>
        {!isEmpty && (
          <button className="clear-btn" onClick={onClear}>
            Clear
          </button>
        )}
      </div>
      
      <div className="feed-content" ref={feedContentRef} onScroll={handleScroll}>
        {isEmpty ? (
          <div className="empty-state">
            <div className="empty-icon">📭</div>
            <p>No messages yet</p>
            <p className="empty-hint">Subscribe to a subject to see messages</p>
          </div>
        ) : (
          <div className="messages-list">
            {messages.map(msg => (
              <div key={msg.id} className="message-item">
                <div className="message-header">
                  <span className="message-subject">{msg.subject}</span>
                  <div className="message-meta">
                    {msg.sequence && (
                      <span className="message-seq">#{msg.sequence}</span>
                    )}
                    <span className="message-time">{formatTimestamp(msg.timestamp)}</span>
                  </div>
                </div>
                <pre className={`message-payload ${isJson(msg.payload) ? 'is-json' : ''}`}>
                  {formatPayload(msg.payload)}
                </pre>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
