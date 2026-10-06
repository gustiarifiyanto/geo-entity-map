import { useCallback, useMemo, useRef, useState, type ReactNode } from 'react'
import { AppBar, type View } from './components/AppBar'
import { ConfirmDialog } from './components/ConfirmDialog'
import { useToast } from './components/toast/context'
import { AuthScreen } from './features/auth/AuthScreen'
import { useI18n } from './i18n/context'
import { useLogout, useMe } from './features/auth/hooks'
import { HelpDialog } from './features/help/HelpDialog'
import { Dashboard } from './features/dashboard/Dashboard'
import { DeleteEntityDialog } from './features/entities/DeleteEntityDialog'
import { EntityDetailPanel } from './features/entities/EntityDetailPanel'
import { CreateEntityPanel, EditEntityPanel } from './features/entities/EntityFormPanels'
import { EntityMap } from './features/entities/EntityMap'
import { EntitySearch } from './features/entities/EntitySearch'
import { EMPTY_FILTER, filterEntities, isFilterActive, type EntityFilterState } from './features/entities/search'
import { MapHeader } from './features/entities/MapHeader'
import { useEntities, useMeta, useUpdateEntityLocation } from './features/entities/hooks'
import { distanceMeters, formatDistance } from './features/geofences/distance'
import { useGeofences } from './features/geofences/hooks'
import { ZoneEditorContext, type ZoneEditor, type ZonePreview } from './features/geofences/zoneEditor'
import { ADMIN_ROLE, type User } from './types/auth'
import type { Entity } from './types/entity'

/** What the side panel shows. Only UI state lives here; entities come from React Query. */
type Panel =
  | { kind: 'none' }
  | { kind: 'view'; id: string }
  | { kind: 'create'; latitude: number; longitude: number }
  | { kind: 'edit'; id: string }

const NO_PANEL: Panel = { kind: 'none' }

function App() {
  const me = useMe()
  const { t, errorText } = useI18n()

  // Data first: a failed background heartbeat (e.g. the backend restarting)
  // must not replace the map with an error screen.
  if (me.data) {
    // key: a different account starts with a fresh map state.
    return <MapScreen key={me.data.id} user={me.data} />
  }
  if (me.isPending) {
    return <CenteredMessage>{t.common.loading}</CenteredMessage>
  }
  if (me.isError) {
    return (
      <CenteredMessage>
        <p className="text-red-700">{errorText(me.error)}</p>
        <button type="button" onClick={() => void me.refetch()} className="mt-2 font-medium underline">
          {t.common.retry}
        </button>
      </CenteredMessage>
    )
  }
  return <AuthScreen />
}

function CenteredMessage({ children }: { children: ReactNode }) {
  return <div className="flex h-full flex-col items-center justify-center text-sm text-gray-500">{children}</div>
}

/** The map. Only admins can add, edit, move or delete; other roles only view. */
function MapScreen({ user }: { user: User }) {
  const canManage = user.role === ADMIN_ROLE
  const logout = useLogout()
  const { t, errorText, locale } = useI18n()
  const meta = useMeta()
  const entities = useEntities()
  const [view, setView] = useState<View>('map')
  const [confirmLogout, setConfirmLogout] = useState(false)
  const [helpOpen, setHelpOpen] = useState(false)
  const [panel, setPanel] = useState<Panel>(NO_PANEL)
  const [deleteTarget, setDeleteTarget] = useState<Entity | null>(null)
  const updateLocation = useUpdateEntityLocation()
  const toast = useToast()
  const zones = useGeofences()

  // Zone editing in the form: the next map click can pick a zone center, and
  // the zone being typed is previewed on the map.
  const pickHandler = useRef<((latitude: number, longitude: number) => void) | null>(null)
  const [picking, setPicking] = useState(false)
  const [zonePreview, setZonePreview] = useState<ZonePreview | null>(null)
  const zoneEditor = useMemo<ZoneEditor>(
    () => ({
      picking,
      startPicking: (onPick) => {
        pickHandler.current = onPick
        setPicking(true)
      },
      cancelPicking: () => {
        pickHandler.current = null
        setPicking(false)
      },
      setPreview: setZonePreview,
    }),
    [picking],
  )

  const selectedId = panel.kind === 'view' || panel.kind === 'edit' ? panel.id : null
  // Derived from the query cache, so the panel closes if the entity disappears.
  const selected = entities.data?.find((e) => e.id === selectedId) ?? null
  const isFormOpen = panel.kind === 'create' || panel.kind === 'edit'

  const closePanel = useCallback(() => setPanel(NO_PANEL), [])
  const showEntity = useCallback((id: string) => setPanel({ kind: 'view', id }), [])

  const handleMapClick = useCallback(
    (latitude: number, longitude: number) => {
      // While a zone center is being picked, the click goes to the form.
      if (pickHandler.current) {
        pickHandler.current(latitude, longitude)
        pickHandler.current = null
        setPicking(false)
        return
      }
      // Viewers cannot add entities, so the click does nothing for them.
      // While editing, ignore map clicks so unsaved changes are not lost.
      // Otherwise start (or move) a new entity at the clicked point.
      if (!canManage || !meta.data) return
      setPanel((current) => (current.kind === 'edit' ? current : { kind: 'create', latitude, longitude }))
    },
    [canManage, meta.data],
  )

  const handleSelect = useCallback(
    (id: string) => {
      if (!isFormOpen) showEntity(id)
    },
    [isFormOpen, showEntity],
  )

  // From the dashboard: switch to the map, fly to the entity and show its
  // details. An open form is kept, so unsaved changes are never lost.
  const [focus, setFocus] = useState<{ latitude: number; longitude: number } | null>(null)

  // Search card: the matches are listed and are the only pins on the map while
  // a search or filter is active. The selected entity always stays visible, so
  // its open panel never points at a hidden pin.
  const [filter, setFilter] = useState<EntityFilterState>(EMPTY_FILTER)
  const searchResults = useMemo(() => filterEntities(entities.data ?? [], filter), [entities.data, filter])
  const mapEntities = useMemo(() => {
    const all = entities.data ?? []
    if (!isFilterActive(filter)) return all
    const shown = new Set(searchResults.map((e) => e.id))
    return all.filter((e) => shown.has(e.id) || e.id === selectedId)
  }, [entities.data, filter, searchResults, selectedId])
  const openOnMap = useCallback(
    (id: string) => {
      const entity = entities.data?.find((e) => e.id === id)
      if (!entity) return
      setView('map')
      setFocus({ latitude: entity.latitude, longitude: entity.longitude })
      if (isFormOpen) toast.error(t.dashboard.formOpen)
      else showEntity(id)
    },
    [entities.data, isFormOpen, showEntity, toast, t],
  )

  const handleMove = useCallback(
    (id: string, latitude: number, longitude: number) => {
      const original = entities.data?.find((e) => e.id === id)
      if (!original) return
      const { name } = original
      const previous = { latitude: original.latitude, longitude: original.longitude }

      const undo = () =>
        updateLocation.mutate(
          { id, location: previous },
          {
            onSuccess: () => toast.success(t.map.movedBack(name)),
            onError: (error) => toast.error(t.map.undoFailed(name, errorText(error))),
          },
        )

      updateLocation.mutate(
        { id, location: { latitude, longitude } },
        {
          onSuccess: () => {
            const action = { label: t.common.undo, onClick: undo }
            // Moving outside the zone is allowed, but the admin is warned.
            const zone = zones.data?.find((z) => z.entity_id === id)
            const distance = zone && distanceMeters(zone.center_latitude, zone.center_longitude, latitude, longitude)
            if (zone && distance !== undefined && distance > zone.radius_m) {
              toast.error(
                t.map.movedOutside(name, formatDistance(distance - zone.radius_m, locale)),
                { action },
              )
            } else {
              toast.success(t.map.moved(name), { action })
            }
          },
          // The hook has already restored the previous position.
          onError: (error) => toast.error(t.map.moveFailed(name, errorText(error))),
        },
      )
    },
    [entities.data, updateLocation, toast, zones.data, t, errorText, locale],
  )

  const error = entities.error ?? meta.error

  return (
    <div className="relative h-full">
      <AppBar
        user={user}
        view={view}
        onViewChange={setView}
        onLogout={() => setConfirmLogout(true)}
        loggingOut={logout.isPending}
      />
      {/* The map fills the screen under the translucent bar and stays mounted
          under the dashboard, so switching tabs keeps the selection, an open
          form and the map position. Overlays start below the bar (top-17). */}
      <ZoneEditorContext.Provider value={zoneEditor}>
        <main className="absolute inset-0">
          <EntityMap
            entities={mapEntities}
            selectedId={selectedId}
            onSelect={handleSelect}
            onMapClick={handleMapClick}
            markersDraggable={canManage && panel.kind === 'view'}
            onMove={handleMove}
            draft={panel.kind === 'create' ? panel : null}
            onDraftMove={handleMapClick}
            zones={zones.data}
            zonePreview={zonePreview}
            picking={picking}
            focus={focus}
          />

          <div className="pointer-events-none absolute inset-x-3 top-17 z-[1000] flex flex-col items-start gap-2">
            <div className="pointer-events-auto animate-fade-in-up">
              <MapHeader entityCount={entities.data?.length} statuses={meta.data?.statuses ?? []} canManage={canManage} />
            </div>
            <div className="pointer-events-auto w-72 max-w-full animate-fade-in-up">
              <EntitySearch
                filter={filter}
                onFilterChange={setFilter}
                results={searchResults}
                total={entities.data?.length ?? 0}
                statuses={meta.data?.statuses ?? []}
                onOpen={openOnMap}
              />
            </div>
            {error && (
              <div
                role="alert"
                className="pointer-events-auto flex animate-fade-in-up items-center gap-3 rounded-lg bg-red-50 px-4 py-2 text-sm text-red-700 shadow ring-1 ring-red-200"
              >
                <span>{errorText(error)}</span>
                <button
                  type="button"
                  onClick={() => {
                    void entities.refetch()
                    void meta.refetch()
                  }}
                  className="font-medium underline"
                >
                  {t.common.retry}
                </button>
              </div>
            )}
          </div>

          {/* Admin-only guide, bottom left (zoom and attribution are on the right). */}
          {canManage && (
            <button
              type="button"
              onClick={() => setHelpOpen(true)}
              aria-label={t.help.button}
              title={t.help.button}
              className="absolute bottom-6 left-3 z-[1000] flex h-10 w-10 animate-fade-in items-center justify-center rounded-full bg-white/95 text-gray-700 shadow-lg ring-1 ring-black/10 backdrop-blur transition-colors hover:bg-white hover:text-gray-900"
            >
              <svg viewBox="0 0 20 20" className="h-5 w-5" fill="currentColor" aria-hidden="true">
                <path
                  fillRule="evenodd"
                  d="M18 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0ZM8.94 6.94a.75.75 0 1 1-1.061-1.061 3 3 0 1 1 2.871 5.026v.345a.75.75 0 0 1-1.5 0v-.5c0-.72.57-1.172 1.081-1.287A1.5 1.5 0 1 0 8.94 6.94ZM10 15a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z"
                  clipRule="evenodd"
                />
              </svg>
            </button>
          )}

          {panel.kind !== 'none' && (
            <div
              // A new key per panel (not per keystroke or map click) replays the entrance animation.
              key={panel.kind === 'create' ? 'create' : `${panel.kind}-${panel.id}`}
              className="absolute inset-x-3 bottom-3 z-[1000] flex max-h-[70%] animate-fade-in-up flex-col sm:inset-x-auto sm:top-17 sm:right-3 sm:bottom-3 sm:max-h-none sm:w-96"
            >
              {panel.kind === 'view' && selected && (
                <EntityDetailPanel
                  entity={selected}
                  onClose={closePanel}
                  onEdit={canManage ? () => setPanel({ kind: 'edit', id: selected.id }) : undefined}
                  onDelete={canManage ? () => setDeleteTarget(selected) : undefined}
                  movable={canManage}
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

          {view === 'dashboard' && (
            <div className="absolute inset-0 z-[1100] animate-fade-in overflow-y-auto bg-gray-50 pt-20">
              <Dashboard
                  entities={entities.data}
                  meta={meta.data}
                  showUserStats={canManage}
                  onOpenEntity={openOnMap}
                />
            </div>
          )}
        </main>
      </ZoneEditorContext.Provider>

      <HelpDialog open={helpOpen} onClose={() => setHelpOpen(false)} meta={meta.data} />

      <ConfirmDialog
        open={confirmLogout}
        title={t.logout.title}
        confirmLabel={t.logout.confirm}
        pendingLabel={t.logout.pending}
        pending={logout.isPending}
        tone="neutral"
        onConfirm={() =>
          logout.mutate(undefined, {
            // On success the login screen replaces this one.
            onError: (err) => {
              setConfirmLogout(false)
              toast.error(t.logout.failed(errorText(err)))
            },
          })
        }
        onCancel={() => setConfirmLogout(false)}
      >
        <p>{t.logout.body(user.email)}</p>
      </ConfirmDialog>

      <DeleteEntityDialog
        entity={deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onDeleted={() => {
          setDeleteTarget(null)
          closePanel()
        }}
      />
    </div>
  )
}

export default App
