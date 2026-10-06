import type { Installation } from '../../types/installation'
import { CAP_INSTALLATION, hasCapability } from '../../schemas/capabilities'
import { useMeta } from '../entities/hooks'
import { useInstallation } from './hooks'
import { useI18n } from '../../i18n/context'
import { formatDay, installationStatusColor } from './status'

interface InstallationSummaryProps {
  entityId: string
  entityType: string
}

/** Installation status for the detail panel; renders nothing for types without the capability. */
export function InstallationSummary({ entityId, entityType }: InstallationSummaryProps) {
  const { t, errorText } = useI18n()
  const meta = useMeta()
  const enabled = hasCapability(meta.data, entityType, CAP_INSTALLATION)
  const installation = useInstallation(entityId, enabled)

  if (!enabled) return null

  return (
    <div>
      <dt className="text-xs font-medium tracking-wide text-gray-400 uppercase">{t.installation.section}</dt>
      <dd className="mt-1 text-gray-900">
        {installation.isPending ? (
          <span className="text-gray-400">{t.common.loading}</span>
        ) : installation.isError ? (
          <span className="text-red-700">{errorText(installation.error)}</span>
        ) : installation.data === null ? (
          <span className="text-gray-400">{t.installation.none}</span>
        ) : (
          <ScheduleDetails installation={installation.data} />
        )}
      </dd>
    </div>
  )
}

export function InstallationStatusBadge({ status }: { status: string }) {
  const { value } = useI18n()
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-700">
      <span className="h-2 w-2 rounded-full" style={{ backgroundColor: installationStatusColor(status) }} />
      {value(status)}
    </span>
  )
}

function ScheduleDetails({ installation: i }: { installation: Installation }) {
  const { t, locale } = useI18n()
  const days = t.common.days
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
            {done ? t.installation.late(days(i.days_late)) : t.installation.lateSoFar(days(i.days_late))}
          </span>
        )}
      </div>

      <div
        className="flex h-2 overflow-hidden rounded-full bg-gray-100"
        role="img"
        aria-label={t.installation.progress(days(i.elapsed_days), days(i.planned_days))}
      >
        <div className="h-full animate-grow-x bg-gray-700" style={{ width: `${(onTime / total) * 100}%` }} />
        {i.elapsed_days > planned && (
          <div className="ml-0.5 h-full animate-grow-x bg-red-500" style={{ width: `${((i.elapsed_days - planned) / total) * 100}%` }} />
        )}
      </div>
      <p className="text-xs text-gray-500">
        {done
          ? t.installation.taken(days(i.elapsed_days), days(i.planned_days))
          : t.installation.soFar(days(i.elapsed_days), days(i.planned_days))}
      </p>

      <dl className="grid grid-cols-3 gap-2 text-xs">
        <DateItem label={t.installation.started} value={formatDay(i.started_on, locale)} />
        <DateItem label={t.installation.target} value={formatDay(i.target_on, locale)} />
        <DateItem label={t.installation.completed} value={i.completed_on ? formatDay(i.completed_on, locale) : '—'} />
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
