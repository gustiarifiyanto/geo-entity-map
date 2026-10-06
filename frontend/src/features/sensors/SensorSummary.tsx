import { CAP_READINGS, hasCapability } from '../../schemas/capabilities'
import { useMeta } from '../entities/hooks'
import { useReadings } from './hooks'
import { ReadingsChart } from './ReadingsChart'

interface SensorSummaryProps {
  entityId: string
  entityType: string
}

/** Latest reading and a 24-hour chart for the detail panel; nothing for types without sensors. */
export function SensorSummary({ entityId, entityType }: SensorSummaryProps) {
  const meta = useMeta()
  const enabled = hasCapability(meta.data, entityType, CAP_READINGS)
  const readings = useReadings(entityId, enabled)

  if (!enabled) return null

  return (
    <div>
      <dt className="text-xs font-medium tracking-wide text-gray-400 uppercase">Sensor</dt>
      <dd className="mt-1 text-gray-900">
        {readings.isPending ? (
          <span className="text-gray-400">Loading…</span>
        ) : readings.isError ? (
          <span className="text-red-700">{readings.error.message}</span>
        ) : readings.data === null ? (
          <span className="text-gray-400">No sensor set up</span>
        ) : (
          <div className="space-y-2">
            <div className="flex items-baseline justify-between gap-3">
              <span className="text-xs text-gray-500">{readings.data.metric.label}</span>
              {readings.data.latest && (
                <span className="text-xs text-gray-400">{timeAgo(readings.data.latest.recorded_at)}</span>
              )}
            </div>
            {readings.data.latest ? (
              <p className="text-2xl font-semibold text-gray-900">
                {readings.data.latest.value.toLocaleString(undefined, { maximumFractionDigits: 1 })}
                <span className="ml-1 text-sm font-normal text-gray-500">{readings.data.metric.unit}</span>
              </p>
            ) : (
              <p className="text-gray-400">No readings yet</p>
            )}
            <ReadingsChart readings={readings.data.readings} metric={readings.data.metric} />
          </div>
        )}
      </dd>
    </div>
  )
}

function timeAgo(iso: string): string {
  const minutes = Math.round((Date.now() - new Date(iso).getTime()) / 60_000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes} min ago`
  const hours = Math.round(minutes / 60)
  if (hours < 48) return `${hours} h ago`
  return new Date(iso).toLocaleString()
}
