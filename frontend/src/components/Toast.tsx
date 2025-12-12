import { useToastStore } from '../services/toast'
import './Toast.css'

export default function ToastContainer() {
  const toasts = useToastStore((state) => state.toasts)
  const removeToast = useToastStore((state) => state.removeToast)

  if (toasts.length === 0) return null

  return (
    <div className="toast-container">
      {toasts.map((toast) => (
        <div key={toast.id} className={`toast toast--${toast.type}`}>
          <div className="toast-body">
            <div className="toast-icon">
              {toast.type === 'success' && '✓'}
              {toast.type === 'error' && '✕'}
              {toast.type === 'warning' && '⚠'}
              {toast.type === 'info' && 'ℹ'}
            </div>
            <div className="toast-content">
              <div className="toast-title">{toast.title}</div>
              {toast.message && <div className="toast-message">{toast.message}</div>}
            </div>
            <button 
              className="toast-close" 
              onClick={() => removeToast(toast.id)}
              aria-label="Close"
            >
              ✕
            </button>
          </div>
          {toast.duration > 0 && (
            <div className="toast-progress">
              <div 
                className="toast-progress-bar"
                style={{ animationDuration: `${toast.duration}ms` }}
              />
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
