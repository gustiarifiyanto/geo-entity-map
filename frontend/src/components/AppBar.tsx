import { formatLabel } from '../features/entities/labels'
import type { User } from '../types/auth'

export type View = 'map' | 'dashboard'

const VIEWS: { id: View; label: string }[] = [
  { id: 'map', label: 'Map' },
  { id: 'dashboard', label: 'Dashboard' },
]

interface AppBarProps {
  user: User
  view: View
  onViewChange: (view: View) => void
  onLogout: () => void
  loggingOut: boolean
}

export function AppBar({ user, view, onViewChange, onLogout, loggingOut }: AppBarProps) {
  return (
    <header className="flex h-14 shrink-0 items-center gap-3 border-b border-gray-200 bg-white px-4">
      <h1 className="hidden text-base font-semibold text-gray-900 sm:block">Geo Entity Map</h1>

      <div role="tablist" aria-label="View" className="flex rounded-lg bg-gray-100 p-1 text-sm">
        {VIEWS.map(({ id, label }) => (
          <button
            key={id}
            type="button"
            role="tab"
            aria-selected={view === id}
            onClick={() => onViewChange(id)}
            className={`rounded-md px-3 py-1 font-medium ${
              view === id ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      <div className="ml-auto flex min-w-0 items-center gap-2 text-xs">
        <span className="hidden min-w-0 truncate text-gray-700 md:block" title={user.email}>
          {user.email}
        </span>
        <span className="rounded-full bg-gray-100 px-2 py-0.5 font-medium text-gray-600">{formatLabel(user.role)}</span>
        <button
          type="button"
          onClick={onLogout}
          disabled={loggingOut}
          className="whitespace-nowrap font-medium text-gray-500 underline hover:text-gray-900 disabled:opacity-60"
        >
          {loggingOut ? 'Logging out…' : 'Log out'}
        </button>
      </div>
    </header>
  )
}
