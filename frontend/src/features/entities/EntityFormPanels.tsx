import { useState, type ReactNode } from 'react'
import { useToast } from '../../components/toast/context'
import { CAP_INSTALLATION, hasCapability } from '../../schemas/capabilities'
import { emptyFormValues, entityToFormValues } from '../../schemas/entity'
import type { Entity, Meta } from '../../types/entity'
import type { Installation } from '../../types/installation'
import { useInstallation } from '../installations/hooks'
import { useSaveInstallation } from '../installations/save'
import { EMPTY_PHOTO_DRAFT, useSavePhotoDraft, type PhotoDraft } from '../photos/draft'
import { usePhotos } from '../photos/hooks'
import { PhotoField } from '../photos/PhotoField'
import { EntityForm } from './EntityForm'
import { useCreateEntity, useUpdateEntity } from './hooks'

/**
 * Reports parts (photos, installation) that failed after the entity itself
 * was saved. The entity stays saved; the user can retry those parts.
 */
function useFollowUpFailureToast() {
  const toast = useToast()
  return (photos: { failed: number; firstError?: string }, installationError: string | null) => {
    const problems: string[] = []
    if (photos.failed > 0) {
      problems.push(`${photos.failed} photo change${photos.failed > 1 ? 's' : ''} (${photos.firstError})`)
    }
    if (installationError) problems.push(`the installation schedule (${installationError})`)
    if (problems.length > 0) toast.error(`Saved, but these failed: ${problems.join('; ')}`)
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
  const saveInstallation = useSaveInstallation()
  const reportFailures = useFollowUpFailureToast()
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
      onSubmit={async (input, installation) => {
        // Photos and the schedule need the new entity's id, so they follow it.
        const entity = await create.mutateAsync(input)
        reportFailures(await savePhotos(entity.id, photos), await saveInstallation(entity.id, installation, false))
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
  const tracksInstallation = hasCapability(meta, entity.type, CAP_INSTALLATION)
  const installation = useInstallation(entity.id, tracksInstallation)

  // The form reads its default values once, so wait for the stored schedule.
  if (tracksInstallation && installation.isPending) {
    return <PanelMessage>Loading…</PanelMessage>
  }
  if (tracksInstallation && installation.isError) {
    return <PanelMessage>Could not load the installation schedule: {installation.error.message}</PanelMessage>
  }
  return (
    <EditEntityForm
      meta={meta}
      entity={entity}
      installation={installation.data ?? null}
      onSaved={onSaved}
      onCancel={onCancel}
    />
  )
}

function EditEntityForm({
  meta,
  entity,
  installation,
  onSaved,
  onCancel,
}: EditEntityPanelProps & { installation: Installation | null }) {
  const update = useUpdateEntity()
  const existing = usePhotos(entity.id)
  const savePhotos = useSavePhotoDraft()
  const saveInstallation = useSaveInstallation()
  const reportFailures = useFollowUpFailureToast()
  const [photos, setPhotos] = useState<PhotoDraft>(EMPTY_PHOTO_DRAFT)

  return (
    <EntityForm
      meta={meta}
      title="Edit entity"
      submitLabel="Save changes"
      defaultValues={entityToFormValues(entity, installation)}
      extraFields={
        existing.isPending ? (
          <p className="text-sm text-gray-400">Loading photos…</p>
        ) : existing.isError ? (
          <p className="text-sm text-red-600">Photos could not be loaded: {existing.error.message}</p>
        ) : (
          <PhotoField existing={existing.data} draft={photos} onChange={setPhotos} />
        )
      }
      onSubmit={async (input, schedule) => {
        const saved = await update.mutateAsync({ id: entity.id, input })
        // A schedule is only touched while the (new) type supports it; after a
        // type change it is kept but hidden, so nothing is lost by mistake.
        const installationError = hasCapability(meta, saved.type, CAP_INSTALLATION)
          ? await saveInstallation(entity.id, schedule, installation !== null)
          : null
        reportFailures(await savePhotos(entity.id, photos), installationError)
        onSaved(saved)
      }}
      onCancel={onCancel}
    />
  )
}

function PanelMessage({ children }: { children: ReactNode }) {
  return (
    <div className="rounded-xl bg-white p-4 text-sm text-gray-500 shadow-xl ring-1 ring-black/5">{children}</div>
  )
}
