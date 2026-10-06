import { useI18n } from '../../i18n/context'
import { CAP_GEOFENCE, hasCapability } from '../../schemas/capabilities'
import { useMeta } from '../entities/hooks'
import { formatDistance } from './distance'
import { useGeofence } from './hooks'

interface GeofenceSummaryProps {
  entityId: string
  entityType: string
}

/** Operating zone status for the detail panel; nothing for types without zones. */
export function GeofenceSummary({ entityId, entityType }: GeofenceSummaryProps) {
  const { t, errorText, locale } = useI18n()
  const meta = useMeta()
  const enabled = hasCapability(meta.data, entityType, CAP_GEOFENCE)
  const zone = useGeofence(entityId, enabled)

  if (!enabled) return null

  return (
    <div>
      <dt className="text-xs font-medium tracking-wide text-gray-400 uppercase">{t.zone.section}</dt>
      <dd className="mt-1 text-gray-900">
        {zone.isPending ? (
          <span className="text-gray-400">{t.common.loading}</span>
        ) : zone.isError ? (
          <span className="text-red-700">{errorText(zone.error)}</span>
        ) : zone.data === null ? (
          <span className="text-gray-400">{t.zone.none}</span>
        ) : (
          <div className="space-y-1">
            {zone.data.inside ? (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-700">
                <span className="h-2 w-2 rounded-full bg-gray-500" />
                {t.zone.inside(formatDistance(zone.data.distance_m, locale))}
              </span>
            ) : (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-red-50 px-2 py-0.5 text-xs font-medium text-red-700 ring-1 ring-red-200">
                <span className="h-2 w-2 rounded-full bg-red-600" />
                {t.zone.outside(formatDistance(zone.data.distance_m - zone.data.radius_m, locale))}
              </span>
            )}
            <p className="text-xs text-gray-500">
              {t.zone.radiusAround(formatDistance(zone.data.radius_m, locale))}{' '}
              <span className="font-mono">
                {zone.data.center_latitude.toFixed(5)}, {zone.data.center_longitude.toFixed(5)}
              </span>
            </p>
          </div>
        )}
      </dd>
    </div>
  )
}
