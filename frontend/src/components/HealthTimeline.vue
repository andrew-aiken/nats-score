<script setup lang="ts">
import { computed, ref } from 'vue'

export interface HealthCheck {
  id: string
  details: unknown
  message: string
  passed: boolean
  points: number
  timestamp: Date
  subject: string
}

export interface ServiceGroup {
  key: string
  teamId: string
  serviceName: string
  checks: HealthCheck[]
  passRate: number
  lastStatus: boolean
}

const props = defineProps<{
  checks: HealthCheck[]
}>()

const selectedCheck = ref<HealthCheck | null>(null)

// Parse subject to extract team_id and service_name
// Format: results.<team_id>.<service_name>
const parseSubject = (subject: string): { teamId: string; serviceName: string } => {
  const parts = subject.split('.')
  if (parts.length >= 3 && parts[0] === 'results') {
    return {
      teamId: parts[1] ?? 'unknown',
      serviceName: parts.slice(2).join('.')
    }
  }
  return { teamId: 'unknown', serviceName: subject }
}

// Group checks by service
const serviceGroups = computed<ServiceGroup[]>(() => {
  const groups = new Map<string, HealthCheck[]>()
  
  for (const check of props.checks) {
    const { teamId, serviceName } = parseSubject(check.subject)
    const key = `${teamId}.${serviceName}`
    
    if (!groups.has(key)) {
      groups.set(key, [])
    }
    groups.get(key)!.push(check)
  }
  
  // Convert to array and sort by service name
  return Array.from(groups.entries())
    .map(([key, checks]) => {
      const firstCheck = checks[0]
      if (!firstCheck) return null
      const { teamId, serviceName } = parseSubject(firstCheck.subject)
      const passed = checks.filter(c => c.passed).length
      const sortedChecks = [...checks].sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime())
      
      return {
        key,
        teamId,
        serviceName,
        checks: sortedChecks,
        passRate: Math.round((passed / checks.length) * 100),
        lastStatus: sortedChecks[sortedChecks.length - 1]?.passed ?? false
      }
    })
    .filter((g): g is ServiceGroup => g !== null)
    .sort((a, b) => {
      // Sort by team, then by service name
      if (a.teamId !== b.teamId) return a.teamId.localeCompare(b.teamId)
      return a.serviceName.localeCompare(b.serviceName)
    })
})

const formatTime = (date: Date): string => {
  return date.toLocaleTimeString('en-US', {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

const formatDate = (date: Date): string => {
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const stats = computed(() => {
  const total = props.checks.length
  const passed = props.checks.filter(c => c.passed).length
  const failed = total - passed
  const passRate = total > 0 ? Math.round((passed / total) * 100) : 0
  const totalPoints = props.checks.reduce((sum, c) => sum + c.points, 0)
  const services = serviceGroups.value.length
  
  return { total, passed, failed, passRate, totalPoints, services }
})

const selectCheck = (check: HealthCheck) => {
  selectedCheck.value = selectedCheck.value?.id === check.id ? null : check
}
</script>

<template>
  <div class="health-timeline">
    <!-- Stats Summary -->
    <div class="stats-bar">
      <div class="stat">
        <span class="stat-value">{{ stats.services }}</span>
        <span class="stat-label">Services</span>
      </div>
      <div class="stat">
        <span class="stat-value">{{ stats.total }}</span>
        <span class="stat-label">Total Checks</span>
      </div>
      <div class="stat stat--success">
        <span class="stat-value">{{ stats.passed }}</span>
        <span class="stat-label">Passed</span>
      </div>
      <div class="stat stat--error">
        <span class="stat-value">{{ stats.failed }}</span>
        <span class="stat-label">Failed</span>
      </div>
      <div class="stat">
        <span class="stat-value">{{ stats.passRate }}%</span>
        <span class="stat-label">Pass Rate</span>
      </div>
      <div class="stat stat--accent">
        <span class="stat-value">{{ stats.totalPoints }}</span>
        <span class="stat-label">Total Points</span>
      </div>
    </div>

    <!-- Services Grid -->
    <div class="services-container">
      <div v-if="checks.length === 0" class="empty-state">
        <div class="empty-icon">🔍</div>
        <p>No health checks in selected time range</p>
      </div>
      
      <div v-else class="services-list">
        <div 
          v-for="service in serviceGroups" 
          :key="service.key"
          class="service-row"
        >
          <div class="service-info">
            <div class="service-status" :class="{ passed: service.lastStatus, failed: !service.lastStatus }">
              {{ service.lastStatus ? '●' : '●' }}
            </div>
            <div class="service-meta">
              <span class="service-name">{{ service.serviceName }}</span>
              <span class="service-team">Team {{ service.teamId }}</span>
            </div>
            <div class="service-stats">
              <span class="service-pass-rate" :class="{ good: service.passRate >= 80, warning: service.passRate >= 50 && service.passRate < 80, bad: service.passRate < 50 }">
                {{ service.passRate }}%
              </span>
            </div>
          </div>
          
          <div class="service-timeline">
            <div
              v-for="check in service.checks"
              :key="check.id"
              class="timeline-item"
              :class="{ 
                passed: check.passed, 
                failed: !check.passed,
                selected: selectedCheck?.id === check.id 
              }"
              :title="`${check.passed ? 'Passed' : 'Failed'} - ${formatTime(check.timestamp)}`"
              @click="selectCheck(check)"
            >
              <div class="item-indicator"></div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Selected Check Details -->
    <Transition name="slide">
      <div v-if="selectedCheck" class="check-details">
        <div class="details-header">
          <div class="details-status" :class="{ passed: selectedCheck.passed }">
            {{ selectedCheck.passed ? '✓ Passed' : '✗ Failed' }}
          </div>
          <button class="close-btn" @click="selectedCheck = null">✕</button>
        </div>
        
        <div class="details-content">
          <div class="detail-row">
            <span class="detail-label">Subject</span>
            <span class="detail-value mono">{{ selectedCheck.subject }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Timestamp</span>
            <span class="detail-value">{{ formatDate(selectedCheck.timestamp) }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Points</span>
            <span class="detail-value">{{ selectedCheck.points }}</span>
          </div>
          <div v-if="selectedCheck.message" class="detail-row">
            <span class="detail-label">Message</span>
            <span class="detail-value">{{ selectedCheck.message }}</span>
          </div>
          <div v-if="selectedCheck.details" class="detail-row detail-row--full">
            <span class="detail-label">Details</span>
            <pre class="detail-value mono">{{ JSON.stringify(selectedCheck.details, null, 2) }}</pre>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.health-timeline {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  height: 100%;
}

.stats-bar {
  display: flex;
  gap: 1rem;
  padding: 1rem;
  background: var(--color-surface);
  border-radius: 12px;
  flex-shrink: 0;
}

.stat {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0.75rem;
  background: var(--color-bg);
  border-radius: 8px;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--color-text);
}

.stat-label {
  font-size: 0.6875rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  margin-top: 0.25rem;
}

.stat--success .stat-value {
  color: var(--color-success);
}

.stat--error .stat-value {
  color: var(--color-error);
}

.stat--accent .stat-value {
  color: var(--color-accent);
}

.services-container {
  flex: 1;
  background: var(--color-surface);
  border-radius: 12px;
  padding: 1rem;
  overflow-y: auto;
  min-height: 0;
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

.services-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.service-row {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 1rem;
  background: var(--color-bg);
  border-radius: 8px;
  border: 1px solid var(--color-border);
}

.service-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-width: 220px;
  flex-shrink: 0;
}

.service-status {
  font-size: 0.75rem;
  line-height: 1;
}

.service-status.passed {
  color: var(--color-success);
}

.service-status.failed {
  color: var(--color-error);
}

.service-meta {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  min-width: 0;
  flex: 1;
}

.service-name {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-team {
  font-size: 0.6875rem;
  color: var(--color-text-muted);
}

.service-stats {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.service-pass-rate {
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  background: var(--color-surface);
}

.service-pass-rate.good {
  color: var(--color-success);
}

.service-pass-rate.warning {
  color: var(--color-warning);
}

.service-pass-rate.bad {
  color: var(--color-error);
}

.service-timeline {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
  flex: 1;
  min-width: 0;
}

.timeline-item {
  width: 20px;
  height: 20px;
  border-radius: 3px;
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.timeline-item.passed {
  background: rgba(34, 197, 94, 0.2);
}

.timeline-item.passed .item-indicator {
  width: 10px;
  height: 10px;
  border-radius: 2px;
  background: var(--color-success);
}

.timeline-item.failed {
  background: rgba(239, 68, 68, 0.2);
}

.timeline-item.failed .item-indicator {
  width: 10px;
  height: 10px;
  border-radius: 2px;
  background: var(--color-error);
}

.timeline-item:hover {
  transform: scale(1.3);
  z-index: 1;
}

.timeline-item.selected {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

.check-details {
  background: var(--color-surface);
  border-radius: 12px;
  overflow: hidden;
  flex-shrink: 0;
}

.details-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.details-status {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-error);
}

.details-status.passed {
  color: var(--color-success);
}

.close-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.875rem;
  color: var(--color-text-muted);
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.close-btn:hover {
  color: var(--color-text);
  background: var(--color-bg);
}

.details-content {
  padding: 1rem 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.detail-row {
  display: flex;
  gap: 1rem;
}

.detail-row--full {
  flex-direction: column;
  gap: 0.5rem;
}

.detail-label {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  min-width: 100px;
}

.detail-value {
  font-size: 0.875rem;
  color: var(--color-text);
}

.detail-value.mono {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.8125rem;
}

pre.detail-value {
  margin: 0;
  padding: 0.75rem;
  background: var(--color-bg);
  border-radius: 6px;
  overflow-x: auto;
  white-space: pre-wrap;
}

/* Transitions */
.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  opacity: 0;
  transform: translateY(10px);
}
</style>
