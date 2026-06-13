// Toast store — module-level reactive list shared across the app. useToast()
// returns a push() function; <ToastRegion> (mounted once) renders the queue.
import { reactive } from 'vue'

export type ToastVariant = 'success' | 'error' | 'info'

export interface Toast {
  id: string
  title: string
  desc?: string
  variant?: ToastVariant
  duration?: number
}

export const toasts = reactive<Toast[]>([])

let seq = 0

export function useToast() {
  return (t: Omit<Toast, 'id'>) => {
    const id = `t${seq++}`
    toasts.push({ id, ...t })
    setTimeout(() => {
      const i = toasts.findIndex((x) => x.id === id)
      if (i !== -1) toasts.splice(i, 1)
    }, t.duration ?? 3200)
  }
}
