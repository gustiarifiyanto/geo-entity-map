import { useEffect, type ReactNode } from 'react'
import type { Entity } from '../../types/entity'
import { InstallationSummary } from '../installations/InstallationSummary'
import { PhotoGallery } from '../photos/PhotoGallery'
import { formatLabel, statusColor } from './labels'

interface EntityDetailPanelProps {
  entity: Entity
  onClose: () => void
  onEdit?: () => void
  onDelete?: () => void
  /** Whether the user may drag the pin; only then is the drag hint shown. */
  movable?: boolean
}

export function EntityDetailPanel({
  entity,
  onClose,
  onEdit,
  onDelete,
  movable,
}: EntityDetailPanelProps) {
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      // An open dialog (photo viewer, confirmation) handles Escape itself.
      if (event.key === 'Escape' && !document.querySelector('dialog[open]')) onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  const attributes = entity.attributes ? Object.entries(entity.attributes) : []

  return (
    <aside
      aria-label="Entity details"
      className="flex max-h-full flex-col overflow-hidden rounded-xl bg-white shadow-xl ring-1 ring-black/5"
    >
      <header className="flex items-start justify-between gap-3 border-b border-gray-100 p-4">
        <div className="min-w-0">
          <h2 className="truncate text-lg font-semibold text-gray-900" title={entity.name}>
            {entity.name}
          </h2>
          <div className="mt-1 flex flex-wrap items-center gap-2 text-xs">
            <span className="rounded-full bg-gray-100 px-2 py-0.5 font-medium text-gray-700">
              {formatLabel(entity.type)}
            </span>
            <StatusBadge status={entity.status} />
          </div>
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label="Close details"
          className="rounded-md p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-700"
        >
          <svg viewBox="0 0 20 20" className="h-5 w-5" fill="currentColor" aria-hidden="true">
            <path d="M6.28 5.22a.75.75 0 0 0-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 1 0 1.06 1.06L10 11.06l3.72 3.72a.75.75 0 1 0 1.06-1.06L11.06 10l3.72-3.72a.75.75 0 0 0-1.06-1.06L10 8.94 6.28 5.22Z" />
          </svg>
        </button>
      </header>

      <dl className="flex-1 space-y-4 overflow-y-auto p-4 text-sm">
        <Field label="Coordinates">
          <span className="font-mono">
            {entity.latitude.toFixed(6)}, {entity.longitude.toFixed(6)}
          </span>
          {movable && <p className="mt-1 text-xs text-gray-500">Drag the highlighted pin on the map to move it.</p>}
        </Field>
        <InstallationSummary entityId={entity.id} entityType={entity.type} />
        <Field label="Photos">
          <PhotoGallery entityId={entity.id} entityName={entity.name} />
        </Field>
        <Field label="Description">
          {entity.description || <span className="text-gray-400">No description</span>}
        </Field>
        <Field label="Attributes">
          {attributes.length === 0 ? (
            <span className="text-gray-400">No attributes</span>
          ) : (
            <ul className="divide-y divide-gray-100 rounded-md border border-gray-100">
              {attributes.map(([key, value]) => (
                <li key={key} className="flex justify-between gap-3 px-3 py-1.5">
                  <span className="text-gray-500">{key}</span>
                  <span className="break-all text-right font-mono text-gray-900">
                    {typeof value === 'string' ? value : JSON.stringify(value)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </Field>
        <div className="grid grid-cols-2 gap-4">
          <Field label="Created">{formatDate(entity.created_at)}</Field>
          <Field label="Updated">{formatDate(entity.updated_at)}</Field>
        </div>
      </dl>

      {(onEdit || onDelete) && (
        <footer className="flex gap-2 border-t border-gray-100 p-4">
          {onEdit && (
            <button
              type="button"
              onClick={onEdit}
              className="flex-1 rounded-lg bg-gray-900 px-3 py-2 text-sm font-medium text-white hover:bg-gray-700"
            >
              Edit
            </button>
          )}
          {onDelete && (
            <button
              type="button"
              onClick={onDelete}
              className="flex-1 rounded-lg border border-red-200 px-3 py-2 text-sm font-medium text-red-600 hover:bg-red-50"
            >
              Delete
            </button>
          )}
        </footer>
      )}
    </aside>
  )
}

export function StatusBadge({ status }: { status: string }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full bg-gray-100 px-2 py-0.5 font-medium text-gray-700">
      <span className="h-2 w-2 rounded-full" style={{ backgroundColor: statusColor(status) }} />
      {formatLabel(status)}
    </span>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-gray-400">{label}</dt>
      <dd className="mt-1 text-gray-900">{children}</dd>
    </div>
  )
}

function formatDate(iso: string): string {
  const date = new Date(iso)
  return Number.isNaN(date.getTime()) ? iso : date.toLocaleString()
}
