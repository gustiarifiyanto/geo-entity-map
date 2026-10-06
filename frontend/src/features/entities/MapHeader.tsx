import type { User } from '../../types/auth'
import { formatLabel, statusColor } from './labels'

interface MapHeaderProps {
  entityCount: number | undefined
  statuses: string[]
  user: User
  /** Admins see the "click the map to add" hint; viewers see that the map is read-only. */
  canManage: boolean
  onLogout: () => void
  loggingOut: boolean
}

export function MapHeader({ entityCount, statuses, user, canManage, onLogout, loggingOut }: MapHeaderProps) {
  return (
    <div className="rounded-xl bg-white/95 px-4 py-3 shadow-lg ring-1 ring-black/5 backdrop-blur">
      <h1 className="text-base font-semibold text-gray-900">Geo Entity Map</h1>
      <p className="text-xs text-gray-500">
        {entityCount === undefined ? 'Loading…' : `${entityCount} entities`} ·{' '}
        {canManage ? 'click the map to add' : 'view only'}
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
      <div className="mt-2 flex items-center gap-2 border-t border-gray-100 pt-2 text-xs">
        <span className="min-w-0 truncate text-gray-700" title={user.email}>
          {user.email}
        </span>
        <span className="rounded-full bg-gray-100 px-2 py-0.5 font-medium text-gray-600">{formatLabel(user.role)}</span>
        <button
          type="button"
          onClick={onLogout}
          disabled={loggingOut}
          className="ml-auto font-medium text-gray-500 underline hover:text-gray-900 disabled:opacity-60"
        >
          {loggingOut ? 'Logging out…' : 'Log out'}
        </button>
      </div>
    </div>
  )
}
