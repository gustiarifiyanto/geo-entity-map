import { emptyFormValues, entityToFormValues } from '../../schemas/entity'
import type { Entity, Meta } from '../../types/entity'
import { EntityForm } from './EntityForm'
import { useCreateEntity, useUpdateEntity } from './hooks'

interface CreateEntityPanelProps {
  meta: Meta
  latitude: number
  longitude: number
  onCreated: (entity: Entity) => void
  onCancel: () => void
}

export function CreateEntityPanel({ meta, latitude, longitude, onCreated, onCancel }: CreateEntityPanelProps) {
  const create = useCreateEntity()

  return (
    <EntityForm
      meta={meta}
      title="New entity"
      submitLabel="Create"
      defaultValues={emptyFormValues(latitude, longitude)}
      pickedLatitude={latitude}
      pickedLongitude={longitude}
      locationHint="Click the map or drag the black pin to change the location."
      onSubmit={async (input) => onCreated(await create.mutateAsync(input))}
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

  return (
    <EntityForm
      meta={meta}
      title="Edit entity"
      submitLabel="Save changes"
      defaultValues={entityToFormValues(entity)}
      onSubmit={async (input) => onSaved(await update.mutateAsync({ id: entity.id, input }))}
      onCancel={onCancel}
    />
  )
}
