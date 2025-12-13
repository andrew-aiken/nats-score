export interface NatsMessage {
  id: string
  subject: string
  payload: string
  timestamp: Date
  sequence?: number
}

export interface NatsCredentials {
  jwt: string
  seed: string
}

export interface NatsCredentials {
  jwt: string
  seed: string
}

export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'error'

export interface Subscription {
  subject: string
  unsubscribe: () => void
}

