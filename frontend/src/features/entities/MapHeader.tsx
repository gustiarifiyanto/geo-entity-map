import { useI18n } from '../../i18n/context'
import { statusColor } from './labels'

interface MapHeaderProps {
  entityCount: number | undefined
  statuses: string[]
  /** Admins see the "click the map to add" hint; viewers see that the map is read-only. */
  canManage: boolean
}

export function MapHeader({ entityCount, statuses, canManage }: MapHeaderProps) {
  const { t, value } = useI18n()
  return (
    <div className="rounded-xl bg-white/95 px-4 py-3 shadow-lg ring-1 ring-black/5 backdrop-blur">
      <p className="text-xs text-gray-500">
        {entityCount === undefined ? t.common.loading : t.map.entityCount(entityCount)} ·{' '}
        {canManage ? t.map.clickToAdd : t.map.viewOnly}
      </p>
      {statuses.length > 0 && (
        <ul className="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-gray-600" aria-label={t.map.statusLegend}>
          {statuses.map((status) => (
            <li key={status} className="flex items-center gap-1.5">
              <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: statusColor(status) }} />
              {value(status)}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
