import { useEffect, useId, useRef, type ReactNode } from 'react'
import { useI18n } from '../../i18n/context'
import { CAP_GEOFENCE, CAP_INSTALLATION, CAP_READINGS } from '../../schemas/capabilities'
import type { Meta } from '../../types/entity'

interface HelpDialogProps {
  open: boolean
  onClose: () => void
  meta: Meta | undefined
}

/**
 * Admin guide: how to add an entity on the map and what each form section
 * does. Type-specific sections name the types from GET /api/meta, so the
 * guide stays right when capabilities change.
 */
export function HelpDialog({ open, onClose, meta }: HelpDialogProps) {
  const { t, value } = useI18n()
  const ref = useRef<HTMLDialogElement>(null)
  const titleId = useId()

  useEffect(() => {
    const dialog = ref.current
    if (!dialog) return
    if (open && !dialog.open) dialog.showModal()
    if (!open && dialog.open) dialog.close()
  }, [open])

  /** Readable list of the types that have a capability, e.g. "IoT Device, Facility". */
  const typesWith = (capability: string) =>
    (meta?.types ?? []).filter((type) => meta?.capabilities[type]?.includes(capability)).map((type) => value(type))

  const extras = [
    { capability: CAP_INSTALLATION, text: t.help.installation },
    { capability: CAP_READINGS, text: t.help.sensor },
    { capability: CAP_GEOFENCE, text: t.help.zone },
  ]
    .map(({ capability, text }) => ({ capability, types: typesWith(capability), text }))
    .filter(({ types }) => types.length > 0)

  return (
    <dialog
      ref={ref}
      aria-labelledby={titleId}
      onCancel={(event) => {
        // Escape key: let React state drive closing.
        event.preventDefault()
        onClose()
      }}
      onClick={(event) => {
        // Clicking the backdrop targets the <dialog> element itself.
        if (event.target === ref.current) onClose()
      }}
      className="m-auto max-h-[85vh] w-[calc(100%-2rem)] max-w-lg overflow-hidden rounded-xl p-0 shadow-xl backdrop:bg-black/40"
    >
      <div className="flex max-h-[85vh] flex-col">
        <header className="flex items-center justify-between gap-3 border-b border-gray-100 px-5 py-3">
          <h2 id={titleId} className="text-lg font-semibold text-gray-900">
            {t.help.title}
          </h2>
          <button
            type="button"
            onClick={onClose}
            aria-label={t.common.close}
            autoFocus
            className="rounded-md p-1 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700"
          >
            <svg viewBox="0 0 20 20" className="h-5 w-5" fill="currentColor" aria-hidden="true">
              <path d="M6.28 5.22a.75.75 0 0 0-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 1 0 1.06 1.06L10 11.06l3.72 3.72a.75.75 0 1 0 1.06-1.06L11.06 10l3.72-3.72a.75.75 0 0 0-1.06-1.06L10 8.94 6.28 5.22Z" />
            </svg>
          </button>
        </header>

        <div className="space-y-5 overflow-y-auto px-5 py-4 text-sm text-gray-700">
          <HelpSection title={t.help.createTitle}>
            <ol className="list-decimal space-y-1 pl-5">
              {t.help.createSteps.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
          </HelpSection>

          <HelpSection title={t.help.fieldsTitle}>
            <ul className="list-disc space-y-1 pl-5">
              {Object.entries(t.help.fields).map(([key, text]) => (
                <li key={key}>{text}</li>
              ))}
              {extras.map(({ capability, types, text }) => (
                <li key={capability}>{text(types.join(', '))}</li>
              ))}
            </ul>
          </HelpSection>

          <HelpSection title={t.help.tipsTitle}>
            <ul className="list-disc space-y-1 pl-5">
              {t.help.tips.map((tip) => (
                <li key={tip}>{tip}</li>
              ))}
            </ul>
          </HelpSection>
        </div>
      </div>
    </dialog>
  )
}

function HelpSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <h3 className="mb-1.5 text-sm font-semibold text-gray-900">{title}</h3>
      {children}
    </section>
  )
}
