<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { natsService } from '../services/nats'
import ConnectionStatus from '../components/ConnectionStatus.vue'
import TimeRangeSelector from '../components/TimeRangeSelector.vue'
import HealthTimeline from '../components/HealthTimeline.vue'
import type { HealthCheck } from '../components/HealthTimeline.vue'

const status = natsService.status
const messages = natsService.messages
const error = natsService.error

const timeRange = ref('24h')

// Convert time range string to hours
const getHoursFromRange = (range: string): number => {
  switch (range) {
    case '1h': return 1
    case '6h': return 6
    case '24h': return 24
    case '7d': return 168
    case 'all': return -1
    default: return 24
  }
}

// Parse messages into health check format and filter by time range
const healthChecks = computed<HealthCheck[]>(() => {
  const hours = getHoursFromRange(timeRange.value)
  const cutoff = hours > 0 ? new Date(Date.now() - hours * 60 * 60 * 1000) : new Date(0)
  
  return messages.value
    .map(msg => {
      try {
        const data = JSON.parse(msg.payload)
        
        // Parse the timestamp from the message data
        const timestamp = data.timestamp ? new Date(data.timestamp) : msg.timestamp
        
        return {
          id: msg.id,
          details: data.details,
          message: data.message || '',
          passed: data.passed === true,
          points: data.points || 0,
          timestamp,
          subject: msg.subject
        } as HealthCheck
      } catch {
        return null
      }
    })
    .filter((check): check is HealthCheck => {
      if (!check) return false
      return check.timestamp >= cutoff
    })
    .sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime())
})

const connectToNats = async () => {
  try {
    await natsService.connect()
  } catch (err) {
    console.error('Connection failed:', err)
  }
}

onMounted(() => {
  // Connect if not already connected
  if (status.value === 'disconnected') {
    connectToNats()
  }
})
</script>

<template>
  <div class="health-check-view">
    <div class="view-header">
      <div class="header-left">
        <ConnectionStatus :status="status" :error="error" />
      </div>
      <div class="header-right">
        <TimeRangeSelector v-model="timeRange" />
      </div>
    </div>

    <div class="view-content">
      <HealthTimeline :checks="healthChecks" />
    </div>
  </div>
</template>

<style scoped>
.health-check-view {
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

.header-left {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.view-content {
  flex: 1;
  padding: 1.5rem;
  overflow: hidden;
}
</style>

