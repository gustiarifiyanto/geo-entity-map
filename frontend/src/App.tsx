import { useCallback, useState } from 'react'
import { EntityDetailPanel } from './features/entities/EntityDetailPanel'
import { CreateEntityPanel, EditEntityPanel } from './features/entities/EntityFormPanels'
import { EntityMap } from './features/entities/EntityMap'
import { MapHeader } from './features/entities/MapHeader'
import { useEntities, useMeta } from './features/entities/hooks'

/** What the side panel shows. Only UI state lives here; entities come from React Query. */
type Panel =
  | { kind: 'none' }
  | { kind: 'view'; id: string }
  | { kind: 'create'; latitude: number; longitude: number }
  | { kind: 'edit'; id: string }

const NO_PANEL: Panel = { kind: 'none' }

function App() {
  const meta = useMeta()
  const entities = useEntities()
  const [panel, setPanel] = useState<Panel>(NO_PANEL)

  const selectedId = panel.kind === 'view' || panel.kind === 'edit' ? panel.id : null
  // Derived from the query cache, so the panel closes if the entity disappears.
  const selected = entities.data?.find((e) => e.id === selectedId) ?? null
  const isFormOpen = panel.kind === 'create' || panel.kind === 'edit'

  const closePanel = useCallback(() => setPanel(NO_PANEL), [])
  const showEntity = useCallback((id: string) => setPanel({ kind: 'view', id }), [])

  const handleMapClick = useCallback(
    (latitude: number, longitude: number) => {
      // While editing, ignore map clicks so unsaved changes are not lost.
      // Otherwise start (or move) a new entity at the clicked point.
      if (!meta.data) return
      setPanel((current) => (current.kind === 'edit' ? current : { kind: 'create', latitude, longitude }))
    },
    [meta.data],
  )

  const handleSelect = useCallback(
    (id: string) => {
      if (!isFormOpen) showEntity(id)
    },
    [isFormOpen, showEntity],
  )

  const error = entities.error ?? meta.error

  return (
    <div className="relative h-full">
      <EntityMap
        entities={entities.data ?? []}
        selectedId={selectedId}
        onSelect={handleSelect}
        onMapClick={handleMapClick}
        draft={panel.kind === 'create' ? panel : null}
        onDraftMove={handleMapClick}
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

      {panel.kind !== 'none' && (
        <div className="absolute inset-x-3 bottom-3 z-[1000] flex max-h-[70%] flex-col sm:inset-x-auto sm:top-3 sm:right-3 sm:bottom-3 sm:max-h-none sm:w-96">
          {panel.kind === 'view' && selected && (
            <EntityDetailPanel
              entity={selected}
              onClose={closePanel}
              onEdit={() => setPanel({ kind: 'edit', id: selected.id })}
            />
          )}
          {panel.kind === 'create' && meta.data && (
            <CreateEntityPanel
              meta={meta.data}
              latitude={panel.latitude}
              longitude={panel.longitude}
              onCreated={(entity) => showEntity(entity.id)}
              onCancel={closePanel}
            />
          )}
          {panel.kind === 'edit' && selected && meta.data && (
            <EditEntityPanel
              key={selected.id}
              meta={meta.data}
              entity={selected}
              onSaved={(entity) => showEntity(entity.id)}
              onCancel={() => showEntity(selected.id)}
            />
          )}
        </div>
      )}
    </div>
  )
}

export default App
