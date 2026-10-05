import { useCallback, useMemo, useRef, useState, type ReactNode } from 'react'
import { ToastContext, type ToastAction, type ToastApi, type ToastOptions } from './context'

type Variant = 'success' | 'error'

interface Toast {
  id: number
  message: string
  variant: Variant
  action?: ToastAction
}

const DURATION_MS: Record<Variant, number> = { success: 3000, error: 6000 }
/** Toasts with an action (e.g. Undo) stay longer so there is time to use it. */
const ACTION_DURATION_MS = 8000
const MAX_VISIBLE = 3

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])
  const nextId = useRef(0)

  const dismiss = useCallback((id: number) => {
    setToasts((current) => current.filter((t) => t.id !== id))
  }, [])

  const show = useCallback(
    (message: string, variant: Variant, options: ToastOptions = {}) => {
      const id = ++nextId.current
      const toast: Toast = { id, message, variant, action: options.action }
      setToasts((current) => [...current.slice(-(MAX_VISIBLE - 1)), toast])
      window.setTimeout(() => dismiss(id), options.action ? ACTION_DURATION_MS : DURATION_MS[variant])
    },
    [dismiss],
  )

  const api = useMemo<ToastApi>(
    () => ({
      success: (message, options) => show(message, 'success', options),
      error: (message, options) => show(message, 'error', options),
    }),
    [show],
  )

  return (
    <ToastContext.Provider value={api}>
      {children}
      <div className="pointer-events-none fixed inset-x-0 top-3 z-[2000] flex flex-col items-center gap-2 px-3">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            role={toast.variant === 'error' ? 'alert' : 'status'}
            className={`pointer-events-auto flex max-w-md items-start gap-3 rounded-lg px-4 py-2.5 text-sm shadow-lg ring-1 ${
              toast.variant === 'error'
                ? 'bg-red-50 text-red-800 ring-red-200'
                : 'bg-gray-900 text-white ring-black/10'
            }`}
          >
            <span className="flex-1">{toast.message}</span>
            {toast.action && (
              <button
                type="button"
                onClick={() => {
                  dismiss(toast.id)
                  toast.action?.onClick()
                }}
                className="font-semibold underline underline-offset-2 hover:no-underline"
              >
                {toast.action.label}
              </button>
            )}
            <button
              type="button"
              onClick={() => dismiss(toast.id)}
              aria-label="Dismiss notification"
              className="opacity-60 hover:opacity-100"
            >
              ×
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}
