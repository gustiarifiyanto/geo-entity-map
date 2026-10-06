import { useState } from 'react'
import { useToast } from '../../components/toast/context'
import { emptyFormValues, entityToFormValues } from '../../schemas/entity'
import type { Entity, Meta } from '../../types/entity'
import { EMPTY_PHOTO_DRAFT, useSavePhotoDraft, type PhotoDraft } from '../photos/draft'
import { usePhotos } from '../photos/hooks'
import { PhotoField } from '../photos/PhotoField'
import { EntityForm } from './EntityForm'
import { useCreateEntity, useUpdateEntity } from './hooks'

/** Reports photo changes that failed after the entity itself was saved. */
function usePhotoFailureToast() {
  const toast = useToast()
  return ({ failed, firstError }: { failed: number; firstError?: string }) => {
    if (failed > 0) {
      toast.error(`Saved, but ${failed} photo change${failed > 1 ? 's' : ''} failed: ${firstError}`)
    }
  }
}

interface CreateEntityPanelProps {
  meta: Meta
  latitude: number
  longitude: number
  onCreated: (entity: Entity) => void
  onCancel: () => void
}

export function CreateEntityPanel({ meta, latitude, longitude, onCreated, onCancel }: CreateEntityPanelProps) {
  const create = useCreateEntity()
  const savePhotos = useSavePhotoDraft()
  const reportPhotoFailures = usePhotoFailureToast()
  const [photos, setPhotos] = useState<PhotoDraft>(EMPTY_PHOTO_DRAFT)

  return (
    <EntityForm
      meta={meta}
      title="New entity"
      submitLabel="Create"
      defaultValues={emptyFormValues(latitude, longitude)}
      pickedLatitude={latitude}
      pickedLongitude={longitude}
      locationHint="Click the map or drag the black pin to change the location."
      extraFields={<PhotoField existing={[]} draft={photos} onChange={setPhotos} />}
      onSubmit={async (input) => {
        // Photos need the new entity's id, so they are uploaded after it exists.
        const entity = await create.mutateAsync(input)
        reportPhotoFailures(await savePhotos(entity.id, photos))
        onCreated(entity)
      }}
      onCancel={onCancel}
    />
  )
}

interface EditEntityPanelProps {
  meta: Meta
  entity: Entity
  onSaved: (entity: Entity) => void
  onCancel: () => void
}

export function EditEntityPanel({ meta, entity, onSaved, onCancel }: EditEntityPanelProps) {
  const update = useUpdateEntity()
  const existing = usePhotos(entity.id)
  const savePhotos = useSavePhotoDraft()
  const reportPhotoFailures = usePhotoFailureToast()
  const [photos, setPhotos] = useState<PhotoDraft>(EMPTY_PHOTO_DRAFT)

  return (
    <EntityForm
      meta={meta}
      title="Edit entity"
      submitLabel="Save changes"
      defaultValues={entityToFormValues(entity)}
      extraFields={
        existing.isPending ? (
          <p className="text-sm text-gray-400">Loading photos…</p>
        ) : existing.isError ? (
          <p className="text-sm text-red-600">Photos could not be loaded: {existing.error.message}</p>
        ) : (
          <PhotoField existing={existing.data} draft={photos} onChange={setPhotos} />
        )
      }
      onSubmit={async (input) => {
        const saved = await update.mutateAsync({ id: entity.id, input })
        reportPhotoFailures(await savePhotos(entity.id, photos))
        onSaved(saved)
      }}
      onCancel={onCancel}
    />
  )
}
