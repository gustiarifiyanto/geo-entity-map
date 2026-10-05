import { createContext, useContext } from 'react'

export interface ToastAction {
  label: string
  onClick: () => void
}

export interface ToastOptions {
  /** Optional button shown in the toast (e.g. "Undo"); clicking it dismisses the toast. */
  action?: ToastAction
}

export interface ToastApi {
  success: (message: string, options?: ToastOptions) => void
  error: (message: string, options?: ToastOptions) => void
}

export const ToastContext = createContext<ToastApi | null>(null)

export function useToast(): ToastApi {
  const toast = useContext(ToastContext)
  if (!toast) throw new Error('useToast must be used within <ToastProvider>')
  return toast
}
