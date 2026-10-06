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
  /** Asks to log out; the caller confirms before actually logging out. */
  onLogout: () => void
  loggingOut: boolean
}

/** Translucent bar floating over the map, so the map shows through softly. */
export function AppBar({ user, view, onViewChange, onLogout, loggingOut }: AppBarProps) {
  const activeIndex = VIEWS.findIndex((v) => v.id === view)

  return (
    <header className="absolute inset-x-0 top-0 z-[1200] flex h-14 items-center gap-3 border-b border-white/50 bg-white/70 px-4 shadow-sm backdrop-blur-md">
      <h1 className="hidden text-base font-semibold text-gray-900 sm:block">Geo Entity Map</h1>

      <div role="tablist" aria-label="View" className="relative grid grid-cols-2 rounded-lg bg-gray-900/5 p-1 text-sm">
        {/* Sliding indicator behind the active tab. */}
        <span
          aria-hidden="true"
          className="absolute inset-y-1 left-1 w-[calc(50%-0.25rem)] rounded-md bg-white shadow-sm transition-transform duration-300 ease-out motion-reduce:transition-none"
          style={{ transform: `translateX(${activeIndex * 100}%)` }}
        />
        {VIEWS.map(({ id, label }) => (
          <button
            key={id}
            type="button"
            role="tab"
            aria-selected={view === id}
            onClick={() => onViewChange(id)}
            className={`relative rounded-md px-3 py-1 font-medium transition-colors ${
              view === id ? 'text-gray-900' : 'text-gray-500 hover:text-gray-800'
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
        <span className="rounded-full bg-gray-900/5 px-2 py-0.5 font-medium text-gray-700">{formatLabel(user.role)}</span>
        <button
          type="button"
          onClick={onLogout}
          disabled={loggingOut}
          aria-label="Log out"
          title="Log out"
          className="rounded-lg p-2 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 disabled:opacity-60"
        >
          <svg viewBox="0 0 20 20" className="h-5 w-5" fill="currentColor" aria-hidden="true">
            <path
              fillRule="evenodd"
              d="M3 4.25A2.25 2.25 0 0 1 5.25 2h5.5A2.25 2.25 0 0 1 13 4.25v2a.75.75 0 0 1-1.5 0v-2a.75.75 0 0 0-.75-.75h-5.5a.75.75 0 0 0-.75.75v11.5c0 .414.336.75.75.75h5.5a.75.75 0 0 0 .75-.75v-2a.75.75 0 0 1 1.5 0v2A2.25 2.25 0 0 1 10.75 18h-5.5A2.25 2.25 0 0 1 3 15.75V4.25Z"
              clipRule="evenodd"
            />
            <path
              fillRule="evenodd"
              d="M19 10a.75.75 0 0 0-.75-.75H8.704l1.048-.943a.75.75 0 1 0-1.004-1.114l-2.5 2.25a.75.75 0 0 0 0 1.114l2.5 2.25a.75.75 0 1 0 1.004-1.114l-1.048-.943h9.546A.75.75 0 0 0 19 10Z"
              clipRule="evenodd"
              transform="rotate(180 12.5 10)"
            />
          </svg>
        </button>
      </div>
    </header>
  )
}
