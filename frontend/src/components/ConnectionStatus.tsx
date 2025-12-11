import { useMemo } from 'react'
import type { ConnectionStatus as ConnectionStatusType } from '../types'
import './ConnectionStatus.css'

interface ConnectionStatusProps {
  status: ConnectionStatusType
  error?: string | null
}

export default function ConnectionStatus({ status, error }: ConnectionStatusProps) {
  const statusConfig = useMemo(() => {
    switch (status) {
      case 'connected':
        return { label: 'Connected', className: 'status--connected', icon: '●' }
      case 'connecting':
        return { label: 'Connecting...', className: 'status--connecting', icon: '◐' }
      case 'error':
        return { label: 'Error', className: 'status--error', icon: '✕' }
      default:
        return { label: 'Disconnected', className: 'status--disconnected', icon: '○' }
    }
  }, [status])

  return (
    <div className={`connection-status ${statusConfig.className}`}>
      <span className="status-icon">{statusConfig.icon}</span>
      <span className="status-label">{statusConfig.label}</span>
      {error && <span className="status-error">{error}</span>}
    </div>
  )
}
