<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import type { NatsMessage } from '../types'

const props = defineProps<{
  messages: readonly NatsMessage[]
}>()

defineEmits<{
  clear: []
}>()

const feedContent = ref<HTMLElement | null>(null)
const autoScroll = ref(true)

// Auto-scroll to bottom when new messages arrive
watch(() => props.messages.length, async () => {
  if (autoScroll.value && feedContent.value) {
    await nextTick()
    feedContent.value.scrollTop = feedContent.value.scrollHeight
  }
})

// Detect if user has scrolled up (disable auto-scroll)
const handleScroll = () => {
  if (!feedContent.value) return
  const { scrollTop, scrollHeight, clientHeight } = feedContent.value
  // Re-enable auto-scroll if user scrolls to bottom
  autoScroll.value = scrollHeight - scrollTop - clientHeight < 100
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

const isEmpty = computed(() => props.messages.length === 0)
</script>

<template>
  <div class="message-feed">
    <div class="feed-header">
      <h2 class="feed-title">Live Messages</h2>
      <span class="message-count">{{ messages.length }} messages</span>
      <button 
        v-if="!isEmpty" 
        class="clear-btn" 
        @click="$emit('clear')"
      >
        Clear
      </button>
    </div>
    
    <div class="feed-content" ref="feedContent" @scroll="handleScroll">
      <div v-if="isEmpty" class="empty-state">
        <div class="empty-icon">📭</div>
        <p>No messages yet</p>
        <p class="empty-hint">Subscribe to a subject to see messages</p>
      </div>
      
      <TransitionGroup v-else name="message" tag="div" class="messages-list">
        <div 
          v-for="msg in messages" 
          :key="msg.id" 
          class="message-item"
        >
          <div class="message-header">
            <span class="message-subject">{{ msg.subject }}</span>
            <div class="message-meta">
              <span v-if="msg.sequence" class="message-seq">#{{ msg.sequence }}</span>
              <span class="message-time">{{ formatTimestamp(msg.timestamp) }}</span>
            </div>
          </div>
          <pre 
            class="message-payload" 
            :class="{ 'is-json': isJson(msg.payload) }"
          >{{ formatPayload(msg.payload) }}</pre>
        </div>
      </TransitionGroup>
    </div>
  </div>
</template>

<style scoped>
.message-feed {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-surface);
  border-radius: 12px;
  overflow: hidden;
}

.feed-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.feed-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.message-count {
  font-size: 0.75rem;
  color: var(--color-text-muted);
  background: var(--color-bg);
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
}

.clear-btn {
  margin-left: auto;
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-muted);
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.clear-btn:hover {
  color: var(--color-text);
  border-color: var(--color-text-muted);
}

.feed-content {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--color-text-muted);
  text-align: center;
}

.empty-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
  opacity: 0.5;
}

.empty-hint {
  font-size: 0.875rem;
  opacity: 0.7;
}

.messages-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.message-item {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  overflow: hidden;
  transition: border-color 0.2s;
}

.message-item:hover {
  border-color: var(--color-accent);
}

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.625rem 0.875rem;
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid var(--color-border);
}

.message-subject {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--color-accent);
}

.message-meta {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.message-seq {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.6875rem;
  color: var(--color-accent);
  opacity: 0.7;
}

.message-time {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.6875rem;
  color: var(--color-text-muted);
}

.message-payload {
  margin: 0;
  padding: 0.875rem;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--color-text);
  white-space: pre-wrap;
  word-break: break-word;
  overflow-x: auto;
}

.message-payload.is-json {
  color: var(--color-text-secondary);
}

/* Transition animations */
.message-enter-active {
  transition: all 0.3s ease-out;
}

.message-leave-active {
  transition: all 0.2s ease-in;
}

.message-enter-from {
  opacity: 0;
  transform: translateY(-10px);
}

.message-leave-to {
  opacity: 0;
  transform: translateX(20px);
}
</style>

