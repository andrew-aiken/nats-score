import { create } from 'zustand'
import { connect, StringCodec, consumerOpts, createInbox, jwtAuthenticator } from 'nats.ws'
import type { NatsConnection, JetStreamClient, JetStreamSubscription } from 'nats.ws'
import type { ConnectionStatus, NatsMessage } from '../types'
import { getCredentials, login, clearCredentials, isTokenExpired } from './auth'
import { v4 as uuid } from 'uuid'

const NATS_CONFIG = {
  servers: 'ws://localhost:8080',
  subject: 'results.1.>'
}

const sc = StringCodec()

interface NatsState {
  status: ConnectionStatus
  messages: NatsMessage[]
  error: string | null
  subscribedSubject: string | null
  connect: () => Promise<void>
  disconnect: () => Promise<void>
  subscribeToStream: (subject: string) => Promise<void>
  clearMessages: () => void
  getSubscribedSubjects: () => string[]
  getKvValue: (bucket: string, key: string) => Promise<string | null>
  putKvValue: (bucket: string, key: string, value: string) => Promise<void>
}

// Store connection references outside of Zustand state (non-serializable)
let connection: NatsConnection | null = null
let jetstream: JetStreamClient | null = null
let jsSubscription: JetStreamSubscription | null = null

export const useNatsStore = create<NatsState>((set, get) => ({
  status: 'disconnected',
  messages: [],
  error: null,
  subscribedSubject: null,

  connect: async () => {
    if (connection) {
      return
    }

    set({ status: 'connecting', error: null })

    // Get credentials from auth service
    const creds = getCredentials()
    if (!creds) {
      console.log('No credentials found, redirecting to login')
      login()
      return
    }

    // Check if JWT is expired
    if (isTokenExpired(creds.jwt)) {
      console.log('JWT expired, redirecting to login')
      clearCredentials()
      login()
      return
    }

    try {
      const encoder = new TextEncoder()
      connection = await connect({
        servers: NATS_CONFIG.servers,
        authenticator: jwtAuthenticator(creds.jwt, encoder.encode(creds.seed)),
      })

      set({ status: 'connected' })
      console.log('Connected to NATS')

      // Get JetStream context
      jetstream = connection.jetstream()
      console.log('JetStream context created')

      // Monitor connection status
      const done = connection.closed()
      done.then((err) => {
        set({ status: 'disconnected' })
        connection = null
        jetstream = null
        console.log('NATS connection closed')
        
        // If closed due to auth error, clear credentials and redirect to login
        if (err && (err.message?.includes('authorization') || err.message?.includes('auth'))) {
          console.log('Connection closed due to auth error, redirecting to login')
          clearCredentials()
          login()
        }
      })

      // Auto-subscribe to stream with history
      await get().subscribeToStream(NATS_CONFIG.subject)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to connect'
      
      // Check if this is an auth error
      if (errorMessage.includes('authorization') || errorMessage.includes('auth')) {
        console.log('Auth error during connect, clearing credentials and redirecting to login')
        clearCredentials()
        login()
        return
      }
      
      set({ 
        status: 'error', 
        error: errorMessage 
      })
      console.error('Failed to connect to NATS:', err)
      throw err
    }
  },

  disconnect: async () => {
    if (!connection) return

    // Unsubscribe from JetStream
    if (jsSubscription) {
      jsSubscription.unsubscribe()
      jsSubscription = null
      set({ subscribedSubject: null })
    }

    await connection.drain()
    connection = null
    jetstream = null
    set({ status: 'disconnected' })
  },

  subscribeToStream: async (subject: string) => {
    if (!connection || !jetstream) {
      throw new Error('Not connected to NATS')
    }

    if (jsSubscription) {
      console.log('Already subscribed, unsubscribing first...')
      jsSubscription.unsubscribe()
    }

    try {
      // Create consumer options - deliver all messages from the beginning
      const opts = consumerOpts()
      opts.deliverAll() // Start from the first message in the stream
      opts.ackNone() // No acknowledgment needed (view only)
      opts.consumerName(uuid())
      opts.description("Dashboard consumer of " + subject)
      opts.deliverTo(createInbox()) // Required for push consumer

      console.log(`Subscribing to JetStream: ${subject} (with history)`)
      
      jsSubscription = await jetstream.subscribe(subject, opts)
      set({ subscribedSubject: subject })
      
      console.log(`Subscribed to ${subject} - loading historical messages...`)

      // Process messages (both historical and new)
      const sub = jsSubscription
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
            
            // Add messages - deduplicate by ID to prevent duplicates
            set(state => {
              // Check if message already exists
              if (state.messages.some(m => m.id === natsMessage.id)) {
                return state
              }
              return {
                messages: [...state.messages, natsMessage].slice(-500)
              }
            })
          } catch (err) {
            console.error('[JetStream] Error processing message:', err)
          }
        }
      })()
    } catch (err) {
      console.error('Failed to subscribe to JetStream:', err)
      set({ error: err instanceof Error ? err.message : 'Failed to subscribe' })
      throw err
    }
  },

  clearMessages: () => {
    set({ messages: [] })
  },

  getSubscribedSubjects: () => {
    const { subscribedSubject } = get()
    return subscribedSubject ? [subscribedSubject] : []
  },

  getKvValue: async (bucket: string, key: string): Promise<string | null> => {
    if (!jetstream) {
      throw new Error('Not connected to NATS')
    }

    try {
      const kv = await jetstream.views.kv(bucket)
      const entry = await kv.get(key)
      
      if (entry && entry.value) {
        return sc.decode(entry.value)
      }
      return null
    } catch (err) {
      console.error(`Failed to get KV value ${bucket}/${key}:`, err)
      throw err
    }
  },

  putKvValue: async (bucket: string, key: string, value: string): Promise<void> => {
    if (!jetstream) {
      throw new Error('Not connected to NATS')
    }

    try {
      const kv = await jetstream.views.kv(bucket)
      await kv.put(key, sc.encode(value))
      console.log(`Updated KV ${bucket}/${key}`)
    } catch (err) {
      console.error(`Failed to put KV value ${bucket}/${key}:`, err)
      throw err
    }
  }
}))
