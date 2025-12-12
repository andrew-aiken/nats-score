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
  { label: '10m', value: '10m', hours: 10 / 60 },
  { label: '30m', value: '30m', hours: 0.5 },
  { label: '1h', value: '1h', hours: 1 },
  { label: '3h', value: '3h', hours: 3 },
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
