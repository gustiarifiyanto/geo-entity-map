import { inputClass } from '../../components/formStyles'
import { useI18n } from '../../i18n/context'
import type { Entity } from '../../types/entity'
import { statusColor } from './labels'
import { EMPTY_FILTER, isFilterActive, type EntityFilterState } from './search'

interface EntitySearchProps {
  filter: EntityFilterState
  onFilterChange: (filter: EntityFilterState) => void
  /** Entities matching the filter (already sorted); the map shows the same set. */
  results: Entity[]
  total: number
  statuses: string[]
  /** Flies to the entity and opens its details. */
  onOpen: (id: string) => void
}

/**
 * Search by name and filter by status. The matches are listed here and are
 * the only pins on the map while a search or filter is active.
 */
export function EntitySearch({ filter, onFilterChange, results, total, statuses, onOpen }: EntitySearchProps) {
  const { t, value } = useI18n()
  const active = isFilterActive(filter)

  return (
    <div
      role="search"
      aria-label={t.search.label}
      className="w-72 max-w-full rounded-xl bg-white/95 p-3 shadow-lg ring-1 ring-black/5 backdrop-blur"
    >
      <div className="grid grid-cols-[1fr_7rem] gap-2">
        <input
          type="search"
          value={filter.query}
          onChange={(event) => onFilterChange({ ...filter, query: event.target.value })}
          onKeyDown={(event) => {
            // Enter opens the first match; Escape clears everything.
            if (event.key === 'Enter' && results[0]) onOpen(results[0].id)
            if (event.key === 'Escape') onFilterChange(EMPTY_FILTER)
          }}
          placeholder={t.search.placeholder}
          aria-label={t.search.placeholder}
          className={`${inputClass} min-w-0 py-1.5 text-xs`}
        />
        <select
          value={filter.status}
          onChange={(event) => onFilterChange({ ...filter, status: event.target.value })}
          aria-label={t.search.statusFilter}
          className={`${inputClass} py-1.5 text-xs`}
        >
          <option value="">{t.search.allStatuses}</option>
          {statuses.map((s) => (
            <option key={s} value={s}>
              {value(s)}
            </option>
          ))}
        </select>
      </div>

      {active && (
        <div className="mt-2 animate-fade-in">
          <div className="mb-1 flex items-center justify-between text-xs text-gray-500">
            <span aria-live="polite">{t.search.results(results.length, total)}</span>
            <button
              type="button"
              onClick={() => onFilterChange(EMPTY_FILTER)}
              aria-label={t.search.clear}
              title={t.search.clear}
              className="rounded-md p-0.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700"
            >
              <svg viewBox="0 0 20 20" className="h-4 w-4" fill="currentColor" aria-hidden="true">
                <path d="M6.28 5.22a.75.75 0 0 0-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 1 0 1.06 1.06L10 11.06l3.72 3.72a.75.75 0 1 0 1.06-1.06L11.06 10l3.72-3.72a.75.75 0 0 0-1.06-1.06L10 8.94 6.28 5.22Z" />
              </svg>
            </button>
          </div>
          {results.length === 0 ? (
            <p className="py-1 text-xs text-gray-500">{t.search.none}</p>
          ) : (
            <ul className="-mx-1 max-h-60 space-y-0.5 overflow-y-auto text-xs">
              {results.map((e) => (
                <li key={e.id}>
                  <button
                    type="button"
                    onClick={() => onOpen(e.id)}
                    className="group flex w-full items-center gap-2 rounded-md px-1 py-1.5 text-left transition-colors hover:bg-gray-100"
                  >
                    <span className="h-2 w-2 shrink-0 rounded-full" style={{ backgroundColor: statusColor(e.status) }} />
                    <span className="min-w-0 flex-1 truncate text-gray-800 group-hover:underline">{e.name}</span>
                    <span className="shrink-0 text-gray-500">{value(e.type)}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}
