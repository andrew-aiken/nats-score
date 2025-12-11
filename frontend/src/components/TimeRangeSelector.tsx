import './TimeRangeSelector.css'

export interface TimeRange {
  label: string
  value: string
  hours: number
}

interface TimeRangeSelectorProps {
  value: string
  onChange: (value: string) => void
}

const presets: TimeRange[] = [
  { label: '1h', value: '1h', hours: 1 },
  { label: '6h', value: '6h', hours: 6 },
  { label: '24h', value: '24h', hours: 24 },
  { label: '7d', value: '7d', hours: 168 },
  { label: 'All', value: 'all', hours: -1 }
]

export default function TimeRangeSelector({ value, onChange }: TimeRangeSelectorProps) {
  return (
    <div className="time-range-selector">
      <span className="selector-label">Time Range:</span>
      <div className="range-buttons">
        {presets.map(preset => (
          <button
            key={preset.value}
            className={`range-btn ${value === preset.value ? 'active' : ''}`}
            onClick={() => onChange(preset.value)}
          >
            {preset.label}
          </button>
        ))}
      </div>
    </div>
  )
}
