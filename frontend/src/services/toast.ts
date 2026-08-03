import { create } from 'zustand'

const DEFAULT_DURATION = 5000
const SOUND_PREFERENCE_KEY = 'notification_sound_enabled'

export interface Toast {
  id: string
  type: 'success' | 'error' | 'warning' | 'info'
  title: string
  message?: string
  duration: number
}

interface ToastState {
  toasts: Toast[]
  addToast: (toast: Omit<Toast, 'id' | 'duration'> & { duration?: number }) => void
  removeToast: (id: string) => void
}

// Play alert sound using Web Audio API
const playAlertSound = (type: Toast['type']) => {
  try {
    const audioContext = new (window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext)()
    
    // Different frequencies for different toast types
    const frequencies: Record<Toast['type'], number[]> = {
      error: [440, 330],      // A4 -> E4 (descending, urgent)
      warning: [440, 440],    // A4 repeated
      success: [330, 440],    // E4 -> A4 (ascending, positive)
      info: [440],            // Single tone
    }
    
    const freqs = frequencies[type]
    const noteDuration = 0.15
    
    freqs.forEach((freq, index) => {
      const oscillator = audioContext.createOscillator()
      const gainNode = audioContext.createGain()
      
      oscillator.connect(gainNode)
      gainNode.connect(audioContext.destination)
      
      oscillator.frequency.value = freq
      oscillator.type = 'sine'
      
      const startTime = audioContext.currentTime + (index * noteDuration)
      const endTime = startTime + noteDuration
      
      // Envelope for smoother sound
      gainNode.gain.setValueAtTime(0, startTime)
      gainNode.gain.linearRampToValueAtTime(0.3, startTime + 0.01)
      gainNode.gain.linearRampToValueAtTime(0, endTime)
      
      oscillator.start(startTime)
      oscillator.stop(endTime)
    })
  } catch {
    // Audio not supported or blocked, fail silently
  }
}

interface NotificationPreferenceState {
  soundEnabled: boolean
  setSoundEnabled: (enabled: boolean) => void
}

/**
 * Whether notification toasts should play an alert sound.
 * Persisted to localStorage, defaults to enabled.
 */
export const useNotificationPreferenceStore = create<NotificationPreferenceState>((set) => ({
  soundEnabled: localStorage.getItem(SOUND_PREFERENCE_KEY) !== 'false',
  setSoundEnabled: (enabled: boolean) => {
    localStorage.setItem(SOUND_PREFERENCE_KEY, String(enabled))
    set({ soundEnabled: enabled })
  },
}))

export const useToastStore = create<ToastState>((set) => ({
  toasts: [],

  addToast: (toast) => {
    const id = crypto.randomUUID()
    const duration = toast.duration ?? DEFAULT_DURATION
    const newToast: Toast = { ...toast, id, duration }

    if (useNotificationPreferenceStore.getState().soundEnabled) {
      playAlertSound(toast.type)
    }

    set((state) => ({
      toasts: [...state.toasts, newToast]
    }))

    // Auto-remove after duration
    if (duration > 0) {
      setTimeout(() => {
        set((state) => ({
          toasts: state.toasts.filter((t) => t.id !== id)
        }))
      }, duration)
    }
  },

  removeToast: (id) => {
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id)
    }))
  }
}))

// Helper functions for common toast types
export const toast = {
  success: (title: string, message?: string) => 
    useToastStore.getState().addToast({ type: 'success', title, message }),
  error: (title: string, message?: string) => 
    useToastStore.getState().addToast({ type: 'error', title, message }),
  warning: (title: string, message?: string) => 
    useToastStore.getState().addToast({ type: 'warning', title, message }),
  info: (title: string, message?: string) => 
    useToastStore.getState().addToast({ type: 'info', title, message }),
}


