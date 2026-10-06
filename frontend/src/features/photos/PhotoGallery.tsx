import { useState } from 'react'
import { PhotoViewer } from './PhotoViewer'
import { usePhotos } from './hooks'

interface PhotoGalleryProps {
  entityId: string
  entityName: string
}

/** Read-only photo grid for the detail panel. Photos are managed in the entity form. */
export function PhotoGallery({ entityId, entityName }: PhotoGalleryProps) {
  const photos = usePhotos(entityId)
  const [viewing, setViewing] = useState<number | null>(null)

  if (photos.isPending) {
    return <p className="text-gray-400">Loading photos…</p>
  }
  if (photos.isError) {
    return (
      <p className="text-red-700">
        {photos.error.message}{' '}
        <button type="button" onClick={() => void photos.refetch()} className="font-medium underline">
          Retry
        </button>
      </p>
    )
  }
  if (photos.data.length === 0) {
    return <p className="text-gray-400">No photos</p>
  }

  return (
    <>
      <ul className="grid grid-cols-3 gap-2">
        {photos.data.map((photo, i) => (
          <li key={photo.id} className="group animate-fade-in">
            <button
              type="button"
              onClick={() => setViewing(i)}
              aria-label={`View photo ${i + 1} of ${entityName}`}
              className="block aspect-square w-full overflow-hidden rounded-lg bg-gray-100 ring-1 ring-black/5"
            >
              <img
                src={photo.url}
                alt=""
                loading="lazy"
                className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105 motion-reduce:transition-none"
              />
            </button>
          </li>
        ))}
      </ul>
      <PhotoViewer
        photos={photos.data}
        index={viewing}
        title={entityName}
        onIndexChange={setViewing}
        onClose={() => setViewing(null)}
      />
    </>
  )
}
