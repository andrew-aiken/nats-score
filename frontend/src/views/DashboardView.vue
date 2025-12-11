<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { natsService } from '../services/nats'
import ConnectionStatus from '../components/ConnectionStatus.vue'
import MessageFeed from '../components/MessageFeed.vue'

const subjectInput = ref('')

const status = natsService.status
const messages = natsService.messages
const error = natsService.error
const subscribedSubject = natsService.subscribedSubject

const connectToNats = async () => {
  try {
    await natsService.connect()
  } catch (err) {
    console.error('Connection failed:', err)
  }
}

const disconnect = async () => {
  await natsService.disconnect()
}

const changeSubscription = async () => {
  const subject = subjectInput.value.trim()
  if (!subject) return
  
  try {
    await natsService.subscribeToStream(subject)
    subjectInput.value = ''
  } catch (err) {
    console.error('Subscribe failed:', err)
  }
}

const clearMessages = () => {
  natsService.clearMessages()
}

const handleKeyPress = (event: KeyboardEvent) => {
  if (event.key === 'Enter') {
    changeSubscription()
  }
}

onMounted(() => {
  connectToNats()
})

onUnmounted(() => {
  // Don't disconnect when leaving - keep connection for other views
})
</script>

<template>
  <div class="dashboard-view">
    <div class="view-header">
      <ConnectionStatus :status="status" :error="error" />
      <div class="header-actions">
        <button 
          v-if="status === 'connected'" 
          class="btn btn--outline" 
          @click="disconnect"
        >
          Disconnect
        </button>
        <button 
          v-else-if="status !== 'connecting'"
          class="btn btn--primary" 
          @click="connectToNats"
        >
          Connect
        </button>
      </div>
    </div>

    <div class="dashboard-content">
      <aside class="sidebar">
        <div class="subscribe-section">
          <h3 class="section-title">Change Subject Filter</h3>
          <div class="subscribe-form">
            <input 
              v-model="subjectInput"
              type="text" 
              class="subject-input"
              placeholder="Enter subject (e.g., results.>)"
              :disabled="status !== 'connected'"
              @keypress="handleKeyPress"
            />
            <button 
              class="btn btn--primary" 
              :disabled="status !== 'connected' || !subjectInput.trim()"
              @click="changeSubscription"
            >
              Change
            </button>
          </div>
        </div>

        <div class="subscriptions-section">
          <h3 class="section-title">Active Subscription</h3>
          <div v-if="!subscribedSubject" class="no-subs">
            No active subscription
          </div>
          <div v-else class="subscription-item">
            <span class="sub-subject">{{ subscribedSubject }}</span>
            <span class="stream-badge">JetStream</span>
          </div>
        </div>
      </aside>

      <main class="main-content">
        <MessageFeed :messages="messages" @clear="clearMessages" />
      </main>
    </div>
  </div>
</template>

<style scoped>
.dashboard-view {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.view-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
}

.header-actions {
  display: flex;
  gap: 0.75rem;
}

.dashboard-content {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.sidebar {
  width: 320px;
  padding: 1.5rem;
  background: var(--color-surface);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  gap: 2rem;
  overflow-y: auto;
}

.section-title {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  margin: 0 0 1rem 0;
}

.subscribe-form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.subject-input {
  width: 100%;
  padding: 0.75rem 1rem;
  font-size: 0.875rem;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  color: var(--color-text);
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.subject-input:focus {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
}

.subject-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.subject-input::placeholder {
  color: var(--color-text-muted);
}

.no-subs {
  font-size: 0.875rem;
  color: var(--color-text-muted);
  text-align: center;
  padding: 1rem;
}

.subscription-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.625rem 0.875rem;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.sub-subject {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.8125rem;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stream-badge {
  font-size: 0.625rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-accent);
  background: rgba(99, 102, 241, 0.15);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.main-content {
  flex: 1;
  padding: 1.5rem;
  overflow: hidden;
}

.btn {
  padding: 0.625rem 1.25rem;
  font-size: 0.875rem;
  font-weight: 500;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn--primary {
  background: var(--color-accent);
  color: white;
}

.btn--primary:hover:not(:disabled) {
  background: var(--color-accent-hover);
}

.btn--outline {
  background: transparent;
  color: var(--color-text);
  border: 1px solid var(--color-border);
}

.btn--outline:hover:not(:disabled) {
  border-color: var(--color-text-muted);
}
</style>

