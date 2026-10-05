import { useCallback, useState } from 'react'
import { EntityDetailPanel } from './features/entities/EntityDetailPanel'
import { EntityMap } from './features/entities/EntityMap'
import { MapHeader } from './features/entities/MapHeader'
import { useEntities, useMeta } from './features/entities/hooks'

function App() {
  const meta = useMeta()
  const entities = useEntities()
  const [selectedId, setSelectedId] = useState<string | null>(null)

  // Derived from the query cache, so the panel closes if the entity disappears.
  const selected = entities.data?.find((e) => e.id === selectedId) ?? null
  const closeDetail = useCallback(() => setSelectedId(null), [])
  const error = entities.error ?? meta.error

  return (
    <div className="relative h-full">
      <EntityMap
        entities={entities.data ?? []}
        selectedId={selectedId}
        onSelect={setSelectedId}
        onMapClick={closeDetail}
      />

      <div className="pointer-events-none absolute inset-x-3 top-3 z-[1000] flex flex-col items-start gap-2">
        <div className="pointer-events-auto">
          <MapHeader entityCount={entities.data?.length} statuses={meta.data?.statuses ?? []} />
        </div>
        {error && (
          <div
            role="alert"
            className="pointer-events-auto flex items-center gap-3 rounded-lg bg-red-50 px-4 py-2 text-sm text-red-700 shadow ring-1 ring-red-200"
          >
            <span>{error.message}</span>
            <button
              type="button"
              onClick={() => {
                void entities.refetch()
                void meta.refetch()
              }}
              className="font-medium underline"
            >
              Retry
            </button>
          </div>
        )}
      </div>

      {selected && (
        <div className="absolute inset-x-3 bottom-3 z-[1000] max-h-[60%] sm:inset-x-auto sm:top-3 sm:right-3 sm:bottom-3 sm:max-h-none sm:w-96">
          <EntityDetailPanel entity={selected} onClose={closeDetail} />
        </div>
      )}
    </div>
  )
}

export default App
