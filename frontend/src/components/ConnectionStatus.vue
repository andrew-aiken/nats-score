<script setup lang="ts">
import { computed } from 'vue'
import type { ConnectionStatus } from '../types'

const props = defineProps<{
  status: ConnectionStatus
  error?: string | null
}>()

const statusConfig = computed(() => {
  switch (props.status) {
    case 'connected':
      return { label: 'Connected', class: 'status--connected', icon: '●' }
    case 'connecting':
      return { label: 'Connecting...', class: 'status--connecting', icon: '◐' }
    case 'error':
      return { label: 'Error', class: 'status--error', icon: '✕' }
    default:
      return { label: 'Disconnected', class: 'status--disconnected', icon: '○' }
  }
})
</script>

<template>
  <div class="connection-status" :class="statusConfig.class">
    <span class="status-icon">{{ statusConfig.icon }}</span>
    <span class="status-label">{{ statusConfig.label }}</span>
    <span v-if="error" class="status-error">{{ error }}</span>
  </div>
</template>

<style scoped>
.connection-status {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border-radius: 9999px;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all 0.3s ease;
}

.status-icon {
  font-size: 0.75rem;
}

.status--connected {
  background: rgba(34, 197, 94, 0.15);
  color: #22c55e;
}

.status--connecting {
  background: rgba(234, 179, 8, 0.15);
  color: #eab308;
  animation: pulse 1.5s ease-in-out infinite;
}

.status--disconnected {
  background: rgba(107, 114, 128, 0.15);
  color: #6b7280;
}

.status--error {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.status-error {
  font-size: 0.75rem;
  opacity: 0.8;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}
</style>

