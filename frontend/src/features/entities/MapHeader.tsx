import { formatLabel, statusColor } from './labels'

interface MapHeaderProps {
  entityCount: number | undefined
  statuses: string[]
}

export function MapHeader({ entityCount, statuses }: MapHeaderProps) {
  return (
    <div className="rounded-xl bg-white/95 px-4 py-3 shadow-lg ring-1 ring-black/5 backdrop-blur">
      <h1 className="text-base font-semibold text-gray-900">Geo Entity Map</h1>
      <p className="text-xs text-gray-500">
        {entityCount === undefined ? 'Loading…' : `${entityCount} entities`} · click the map to add
      </p>
      {statuses.length > 0 && (
        <ul className="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-gray-600" aria-label="Status legend">
          {statuses.map((status) => (
            <li key={status} className="flex items-center gap-1.5">
              <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: statusColor(status) }} />
              {formatLabel(status)}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
