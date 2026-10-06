import { useCallback, useState, type ReactNode } from 'react'
import { AppBar, type View } from './components/AppBar'
import { ConfirmDialog } from './components/ConfirmDialog'
import { useToast } from './components/toast/context'
import { AuthScreen } from './features/auth/AuthScreen'
import { useLogout, useMe } from './features/auth/hooks'
import { Dashboard } from './features/dashboard/Dashboard'
import { DeleteEntityDialog } from './features/entities/DeleteEntityDialog'
import { EntityDetailPanel } from './features/entities/EntityDetailPanel'
import { CreateEntityPanel, EditEntityPanel } from './features/entities/EntityFormPanels'
import { EntityMap } from './features/entities/EntityMap'
import { MapHeader } from './features/entities/MapHeader'
import { useEntities, useMeta, useUpdateEntityLocation } from './features/entities/hooks'
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

  // Data first: a failed background heartbeat (e.g. the backend restarting)
  // must not replace the map with an error screen.
  if (me.data) {
    // key: a different account starts with a fresh map state.
    return <MapScreen key={me.data.id} user={me.data} />
  }
  if (me.isPending) {
    return <CenteredMessage>Loading…</CenteredMessage>
  }
  if (me.isError) {
    return (
      <CenteredMessage>
        <p className="text-red-700">{me.error.message}</p>
        <button type="button" onClick={() => void me.refetch()} className="mt-2 font-medium underline">
          Retry
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
  const meta = useMeta()
  const entities = useEntities()
  const [view, setView] = useState<View>('map')
  const [confirmLogout, setConfirmLogout] = useState(false)
  const [panel, setPanel] = useState<Panel>(NO_PANEL)
  const [deleteTarget, setDeleteTarget] = useState<Entity | null>(null)
  const updateLocation = useUpdateEntityLocation()
  const toast = useToast()

  const selectedId = panel.kind === 'view' || panel.kind === 'edit' ? panel.id : null
  // Derived from the query cache, so the panel closes if the entity disappears.
  const selected = entities.data?.find((e) => e.id === selectedId) ?? null
  const isFormOpen = panel.kind === 'create' || panel.kind === 'edit'

  const closePanel = useCallback(() => setPanel(NO_PANEL), [])
  const showEntity = useCallback((id: string) => setPanel({ kind: 'view', id }), [])

  const handleMapClick = useCallback(
    (latitude: number, longitude: number) => {
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
            onSuccess: () => toast.success(`"${name}" moved back.`),
            onError: (error) => toast.error(`Could not undo the move of "${name}": ${error.message}`),
          },
        )

      updateLocation.mutate(
        { id, location: { latitude, longitude } },
        {
          onSuccess: () => toast.success(`"${name}" moved.`, { action: { label: 'Undo', onClick: undo } }),
          // The hook has already restored the previous position.
          onError: (error) => toast.error(`Could not move "${name}": ${error.message}`),
        },
      )
    },
    [entities.data, updateLocation, toast],
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
      <main className="absolute inset-0">
        <EntityMap
          entities={entities.data ?? []}
          selectedId={selectedId}
          onSelect={handleSelect}
          onMapClick={handleMapClick}
          markersDraggable={canManage && panel.kind === 'view'}
          onMove={handleMove}
          draft={panel.kind === 'create' ? panel : null}
          onDraftMove={handleMapClick}
        />

        <div className="pointer-events-none absolute inset-x-3 top-17 z-[1000] flex flex-col items-start gap-2">
          <div className="pointer-events-auto animate-fade-in-up">
            <MapHeader entityCount={entities.data?.length} statuses={meta.data?.statuses ?? []} canManage={canManage} />
          </div>
          {error && (
            <div
              role="alert"
              className="pointer-events-auto flex animate-fade-in-up items-center gap-3 rounded-lg bg-red-50 px-4 py-2 text-sm text-red-700 shadow ring-1 ring-red-200"
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
            <Dashboard entities={entities.data} meta={meta.data} showUserStats={canManage} />
          </div>
        )}
      </main>

      <ConfirmDialog
        open={confirmLogout}
        title="Log out?"
        confirmLabel="Log out"
        pendingLabel="Logging out…"
        pending={logout.isPending}
        tone="neutral"
        onConfirm={() =>
          logout.mutate(undefined, {
            // On success the login screen replaces this one.
            onError: (err) => {
              setConfirmLogout(false)
              toast.error(`Could not log out: ${err.message}`)
            },
          })
        }
        onCancel={() => setConfirmLogout(false)}
      >
        <p>
          You are logged in as <span className="font-medium text-gray-900">{user.email}</span>. You will need to log in
          again to use the map.
        </p>
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
