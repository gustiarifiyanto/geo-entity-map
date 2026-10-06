import type { Installation } from '../../types/installation'
import { CAP_INSTALLATION, hasCapability } from '../../schemas/capabilities'
import { useMeta } from '../entities/hooks'
import { useInstallation } from './hooks'
import { daysText, formatDay, installationStatusColor, installationStatusLabel } from './status'

interface InstallationSummaryProps {
  entityId: string
  entityType: string
}

/** Installation status for the detail panel; renders nothing for types without the capability. */
export function InstallationSummary({ entityId, entityType }: InstallationSummaryProps) {
  const meta = useMeta()
  const enabled = hasCapability(meta.data, entityType, CAP_INSTALLATION)
  const installation = useInstallation(entityId, enabled)

  if (!enabled) return null

  return (
    <div>
      <dt className="text-xs font-medium tracking-wide text-gray-400 uppercase">Installation</dt>
      <dd className="mt-1 text-gray-900">
        {installation.isPending ? (
          <span className="text-gray-400">Loading…</span>
        ) : installation.isError ? (
          <span className="text-red-700">{installation.error.message}</span>
        ) : installation.data === null ? (
          <span className="text-gray-400">No installation schedule</span>
        ) : (
          <ScheduleDetails installation={installation.data} />
        )}
      </dd>
    </div>
  )
}

export function InstallationStatusBadge({ status }: { status: string }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-700">
      <span className="h-2 w-2 rounded-full" style={{ backgroundColor: installationStatusColor(status) }} />
      {installationStatusLabel(status)}
    </span>
  )
}

function ScheduleDetails({ installation: i }: { installation: Installation }) {
  const planned = Math.max(1, i.planned_days)
  // The bar shows the planned span; time past the target spills over in red.
  const onTime = Math.min(i.elapsed_days, planned)
  const total = Math.max(planned, i.elapsed_days)
  const done = i.completed_on !== null

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-2">
        <InstallationStatusBadge status={i.status} />
        {i.days_late > 0 && (
          <span className="text-xs font-medium text-red-600">
            {daysText(i.days_late)} late{done ? '' : ' so far'}
          </span>
        )}
      </div>

      <div
        className="flex h-2 overflow-hidden rounded-full bg-gray-100"
        role="img"
        aria-label={`${daysText(i.elapsed_days)} of ${daysText(i.planned_days)} planned`}
      >
        <div className="h-full animate-grow-x bg-gray-700" style={{ width: `${(onTime / total) * 100}%` }} />
        {i.elapsed_days > planned && (
          <div className="ml-0.5 h-full animate-grow-x bg-red-500" style={{ width: `${((i.elapsed_days - planned) / total) * 100}%` }} />
        )}
      </div>
      <p className="text-xs text-gray-500">
        {daysText(i.elapsed_days)} {done ? 'taken' : 'so far'} · {daysText(i.planned_days)} planned
      </p>

      <dl className="grid grid-cols-3 gap-2 text-xs">
        <DateItem label="Started" value={formatDay(i.started_on)} />
        <DateItem label="Target" value={formatDay(i.target_on)} />
        <DateItem label="Completed" value={i.completed_on ? formatDay(i.completed_on) : '—'} />
      </dl>
    </div>
  )
}

function DateItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-gray-400">{label}</dt>
      <dd className="font-medium text-gray-900">{value}</dd>
    </div>
  )
}
