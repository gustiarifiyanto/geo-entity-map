import { useEntities, useMeta } from './features/entities/hooks'

// Temporary status screen to verify the data layer; replaced by the map in the next step.
function App() {
  const meta = useMeta()
  const entities = useEntities()
  const error = meta.error ?? entities.error

  return (
    <main className="flex h-full flex-col items-center justify-center gap-2 text-center">
      <h1 className="text-2xl font-semibold">Geo Entity Map</h1>
      {error ? (
        <p className="text-red-600">{error.message}</p>
      ) : meta.data && entities.data ? (
        <>
          <p>{entities.data.length} entities loaded</p>
          <p className="text-sm text-gray-500">types: {meta.data.types.join(', ')}</p>
          <p className="text-sm text-gray-500">statuses: {meta.data.statuses.join(', ')}</p>
        </>
      ) : (
        <p className="text-gray-500">Loading…</p>
      )}
    </main>
  )
}

export default App
