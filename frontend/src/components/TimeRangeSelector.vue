<script setup lang="ts">

export interface TimeRange {
  label: string
  value: string
  hours: number
}

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const presets: TimeRange[] = [
  { label: '1h', value: '1h', hours: 1 },
  { label: '6h', value: '6h', hours: 6 },
  { label: '24h', value: '24h', hours: 24 },
  { label: '7d', value: '7d', hours: 168 },
  { label: 'All', value: 'all', hours: -1 }
]

const selectRange = (value: string) => {
  emit('update:modelValue', value)
}

const isSelected = (value: string) => props.modelValue === value
</script>

<template>
  <div class="time-range-selector">
    <span class="selector-label">Time Range:</span>
    <div class="range-buttons">
      <button
        v-for="preset in presets"
        :key="preset.value"
        class="range-btn"
        :class="{ active: isSelected(preset.value) }"
        @click="selectRange(preset.value)"
      >
        {{ preset.label }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.time-range-selector {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.selector-label {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
}

.range-buttons {
  display: flex;
  gap: 0.25rem;
  background: var(--color-bg);
  padding: 0.25rem;
  border-radius: 8px;
}

.range-btn {
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-muted);
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.range-btn:hover {
  color: var(--color-text);
}

.range-btn.active {
  color: white;
  background: var(--color-accent);
}
</style>

