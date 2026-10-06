import { useEffect, useId, useRef, type ReactNode } from 'react'

interface ConfirmDialogProps {
  open: boolean
  title: string
  children: ReactNode
  confirmLabel: string
  pendingLabel?: string
  pending?: boolean
  onConfirm: () => void
  onCancel: () => void
  /** "danger" (red) for destructive actions such as delete; "neutral" otherwise. */
  tone?: 'danger' | 'neutral'
}

/**
 * Modal confirmation built on the native <dialog> element, which provides the
 * focus trap, Escape handling and top-layer stacking (above the Leaflet map).
 */
export function ConfirmDialog({
  open,
  title,
  children,
  confirmLabel,
  pendingLabel,
  pending = false,
  onConfirm,
  onCancel,
  tone = 'danger',
}: ConfirmDialogProps) {
  const ref = useRef<HTMLDialogElement>(null)
  // Unique per dialog: several dialogs can be mounted at once.
  const titleId = useId()

  useEffect(() => {
    const dialog = ref.current
    if (!dialog) return
    if (open && !dialog.open) dialog.showModal()
    if (!open && dialog.open) dialog.close()
  }, [open])

  const cancel = () => {
    if (!pending) onCancel()
  }

  return (
    <dialog
      ref={ref}
      aria-labelledby={titleId}
      onCancel={(event) => {
        // Escape key: let React state drive closing.
        event.preventDefault()
        cancel()
      }}
      onClick={(event) => {
        // Clicking the backdrop targets the <dialog> element itself.
        if (event.target === ref.current) cancel()
      }}
      className="m-auto w-[calc(100%-2rem)] max-w-sm rounded-xl p-0 shadow-xl backdrop:bg-black/40"
    >
      <div className="p-5">
        <h2 id={titleId} className="text-lg font-semibold text-gray-900">
          {title}
        </h2>
        <div className="mt-2 text-sm text-gray-600">{children}</div>
        <div className="mt-5 flex justify-end gap-2">
          <button
            type="button"
            onClick={cancel}
            disabled={pending}
            autoFocus
            className="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 disabled:opacity-60"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={pending}
            className={`rounded-lg px-3 py-2 text-sm font-medium text-white transition-colors disabled:opacity-60 ${
              tone === 'danger' ? 'bg-red-600 hover:bg-red-500' : 'bg-gray-900 hover:bg-gray-700'
            }`}
          >
            {pending && pendingLabel ? pendingLabel : confirmLabel}
          </button>
        </div>
      </div>
    </dialog>
  )
}
