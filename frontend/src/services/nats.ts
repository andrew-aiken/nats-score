import { ref, readonly } from 'vue'
import { connect, StringCodec, consumerOpts, createInbox } from 'nats.ws'
import type { NatsConnection, JetStreamClient, JetStreamSubscription } from 'nats.ws'
import type { ConnectionStatus, NatsMessage } from '../types'

// Hardcoded credentials
const NATS_CONFIG = {
  servers: 'ws://localhost:8080',
  user: 'admin',
  pass: 'adminpass',
  streamName: 'RESULTS',
  subject: 'results.>'
}

const sc = StringCodec()

class NatsService {
  private connection: NatsConnection | null = null
  private jetstream: JetStreamClient | null = null
  private jsSubscription: JetStreamSubscription | null = null
  
  private _status = ref<ConnectionStatus>('disconnected')
  private _messages = ref<NatsMessage[]>([])
  private _error = ref<string | null>(null)
  private _subscribedSubject = ref<string | null>(null)

  public status = readonly(this._status)
  public messages = readonly(this._messages)
  public error = readonly(this._error)
  public subscribedSubject = readonly(this._subscribedSubject)

  async connect(): Promise<void> {
    if (this.connection) {
      return
    }

    this._status.value = 'connecting'
    this._error.value = null

    try {
      this.connection = await connect({
        servers: NATS_CONFIG.servers,
        user: NATS_CONFIG.user,
        pass: NATS_CONFIG.pass,
      })

      this._status.value = 'connected'
      console.log('Connected to NATS')

      // Get JetStream context
      this.jetstream = this.connection.jetstream()
      console.log('JetStream context created')

      // Monitor connection status
      this.monitorConnection()

      // Auto-subscribe to stream with history
      await this.subscribeToStream(NATS_CONFIG.subject)
    } catch (err) {
      this._status.value = 'error'
      this._error.value = err instanceof Error ? err.message : 'Failed to connect'
      console.error('Failed to connect to NATS:', err)
      throw err
    }
  }

  private async monitorConnection(): Promise<void> {
    if (!this.connection) return

    const done = this.connection.closed()
    done.then(() => {
      this._status.value = 'disconnected'
      this.connection = null
      this.jetstream = null
      console.log('NATS connection closed')
    })
  }

  async disconnect(): Promise<void> {
    if (!this.connection) return

    // Unsubscribe from JetStream
    if (this.jsSubscription) {
      this.jsSubscription.unsubscribe()
      this.jsSubscription = null
      this._subscribedSubject.value = null
    }

    await this.connection.drain()
    this.connection = null
    this.jetstream = null
    this._status.value = 'disconnected'
  }

  async subscribeToStream(subject: string): Promise<void> {
    if (!this.connection || !this.jetstream) {
      throw new Error('Not connected to NATS')
    }

    if (this.jsSubscription) {
      console.log('Already subscribed, unsubscribing first...')
      this.jsSubscription.unsubscribe()
    }

    try {
      // Create consumer options - deliver all messages from the beginning
      const opts = consumerOpts()
      opts.deliverAll() // Start from the first message in the stream
      opts.ackNone() // No acknowledgment needed (view only)
      opts.deliverTo(createInbox()) // Required for push consumer

      console.log(`Subscribing to JetStream: ${subject} (with history)`)
      
      this.jsSubscription = await this.jetstream.subscribe(subject, opts)
      this._subscribedSubject.value = subject
      
      console.log(`Subscribed to ${subject} - loading historical messages...`)

      // Process messages (both historical and new)
      this.processJetStreamMessages()
    } catch (err) {
      console.error('Failed to subscribe to JetStream:', err)
      this._error.value = err instanceof Error ? err.message : 'Failed to subscribe'
      throw err
    }
  }

  private processJetStreamMessages(): void {
    if (!this.jsSubscription) return

    const sub = this.jsSubscription
    ;(async () => {
      for await (const msg of sub) {
        try {
          const payload = sc.decode(msg.data)
          
          // Get message timestamp from JetStream metadata if available
          const info = msg.info
          const timestamp = info?.timestampNanos 
            ? new Date(Number(info.timestampNanos) / 1_000_000)
            : new Date()
          
          console.log(`[JetStream] ${msg.subject} (seq: ${info?.streamSequence}):`, payload)
          
          const natsMessage: NatsMessage = {
            id: `${info?.streamSequence || crypto.randomUUID()}`,
            subject: msg.subject,
            payload,
            timestamp,
            sequence: info?.streamSequence
          }
          
          // Add messages - historical ones at the end, new ones at the beginning
          // Since we're delivering all, we append to maintain order
          this._messages.value = [...this._messages.value, natsMessage].slice(-500)
        } catch (err) {
          console.error('[JetStream] Error processing message:', err)
        }
      }
    })()
  }

  getSubscribedSubjects(): string[] {
    return this._subscribedSubject.value ? [this._subscribedSubject.value] : []
  }

  clearMessages(): void {
    this._messages.value = []
  }
}

// Export singleton instance
export const natsService = new NatsService()
