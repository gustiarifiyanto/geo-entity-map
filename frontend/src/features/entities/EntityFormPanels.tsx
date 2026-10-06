import { useState, type ReactNode } from 'react'
import { useToast } from '../../components/toast/context'
import { useI18n } from '../../i18n/context'
import { CAP_GEOFENCE, CAP_INSTALLATION, CAP_READINGS, hasCapability } from '../../schemas/capabilities'
import { emptyFormValues, entityToFormValues, type EntityExtras } from '../../schemas/entity'
import type { Entity, Meta } from '../../types/entity'
import type { Geofence } from '../../types/geofence'
import type { Installation } from '../../types/installation'
import type { SensorConfig } from '../../types/sensor'
import { useGeofence } from '../geofences/hooks'
import { useSaveGeofence } from '../geofences/save'
import { useInstallation } from '../installations/hooks'
import { useSaveInstallation } from '../installations/save'
import { EMPTY_PHOTO_DRAFT, useSavePhotoDraft, type PhotoDraft } from '../photos/draft'
import { usePhotos } from '../photos/hooks'
import { PhotoField } from '../photos/PhotoField'
import { DeviceKeyPanel } from '../sensors/DeviceKeyPanel'
import { useSensor } from '../sensors/hooks'
import { useSaveSensor } from '../sensors/save'
import { EntityForm } from './EntityForm'
import { useCreateEntity, useUpdateEntity } from './hooks'

/** Extras as stored before the form opened (null = none). */
interface StoredExtrasData {
  installation: Installation | null
  sensor: SensorConfig | null
  geofence: Geofence | null
}

/** What failed after the entity itself was saved; empty when all went well. */
interface FollowUpResult {
  photos: { failed: number; firstError?: string }
  installation: string | null
  sensor: string | null
  geofence: string | null
}

/**
 * Reports parts (photos, installation, sensor) that failed after the entity
 * itself was saved. The entity stays saved; the user can retry those parts.
 */
function useFollowUpFailureToast() {
  const toast = useToast()
  const { t, sentence } = useI18n()
  return ({ photos, installation, sensor, geofence }: FollowUpResult) => {
    const problems: string[] = []
    if (photos.failed > 0) {
      problems.push(t.form.photoChanges(photos.failed, sentence(photos.firstError ?? '')))
    }
    if (installation) problems.push(t.form.installationPart(sentence(installation)))
    if (sensor) problems.push(t.form.sensorPart(sentence(sensor)))
    if (geofence) problems.push(t.form.zonePart(sentence(geofence)))
    if (problems.length > 0) toast.error(t.form.followUpFailed(problems.join('; ')))
  }
}

/** Saves everything that belongs next to an entity, once the entity itself is saved. */
function useSaveFollowUps() {
  const savePhotos = useSavePhotoDraft()
  const saveInstallation = useSaveInstallation()
  const saveSensor = useSaveSensor()
  const saveGeofence = useSaveGeofence()
  const report = useFollowUpFailureToast()

  return async (
    meta: Meta,
    entity: Entity,
    extras: EntityExtras,
    photos: PhotoDraft,
    stored: StoredExtrasData,
  ) => {
    // Extras are only touched while the (possibly new) type supports them; after
    // a type change they are kept but hidden, so nothing is lost by mistake.
    const installation = hasCapability(meta, entity.type, CAP_INSTALLATION)
      ? await saveInstallation(entity.id, extras.installation, stored.installation !== null)
      : null
    const sensor = hasCapability(meta, entity.type, CAP_READINGS)
      ? await saveSensor(entity.id, extras.sensorMetric, stored.sensor)
      : null
    const geofence = hasCapability(meta, entity.type, CAP_GEOFENCE)
      ? await saveGeofence(entity.id, extras.geofence, stored.geofence !== null)
      : null
    report({ photos: await savePhotos(entity.id, photos), installation, sensor, geofence })
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
  const { t } = useI18n()
  const create = useCreateEntity()
  const saveFollowUps = useSaveFollowUps()
  const [photos, setPhotos] = useState<PhotoDraft>(EMPTY_PHOTO_DRAFT)

  return (
    <EntityForm
      meta={meta}
      title={t.form.newTitle}
      submitLabel={t.common.create}
      defaultValues={emptyFormValues(latitude, longitude)}
      pickedLatitude={latitude}
      pickedLongitude={longitude}
      locationHint={t.form.locationHint}
      extraFields={<PhotoField existing={[]} draft={photos} onChange={setPhotos} />}
      onSubmit={async (input, extras) => {
        // Extras need the new entity's id, so they follow it.
        const entity = await create.mutateAsync(input)
        await saveFollowUps(meta, entity, extras, photos, { installation: null, sensor: null, geofence: null })
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
  const { t, errorText } = useI18n()
  const installation = useInstallation(entity.id, hasCapability(meta, entity.type, CAP_INSTALLATION))
  const sensor = useSensor(entity.id, hasCapability(meta, entity.type, CAP_READINGS))
  const geofence = useGeofence(entity.id, hasCapability(meta, entity.type, CAP_GEOFENCE))

  // The form reads its default values once, so wait for the stored extras.
  // (A disabled query stays pending, so check fetchStatus too.)
  const loading = [installation, sensor, geofence].some((q) => q.isPending && q.fetchStatus !== 'idle')
  const failed = [installation, sensor, geofence].find((q) => q.isError)
  if (loading) {
    return <PanelMessage>{t.common.loading}</PanelMessage>
  }
  if (failed?.error) {
    return <PanelMessage>{t.form.loadFailed(errorText(failed.error))}</PanelMessage>
  }
  return (
    <EditEntityForm
      meta={meta}
      entity={entity}
      stored={{
        installation: installation.data ?? null,
        sensor: sensor.data ?? null,
        geofence: geofence.data ?? null,
      }}
      onSaved={onSaved}
      onCancel={onCancel}
    />
  )
}

function EditEntityForm({
  meta,
  entity,
  stored,
  onSaved,
  onCancel,
}: EditEntityPanelProps & { stored: StoredExtrasData }) {
  const { t, errorText } = useI18n()
  const update = useUpdateEntity()
  const existing = usePhotos(entity.id)
  const saveFollowUps = useSaveFollowUps()
  const [photos, setPhotos] = useState<PhotoDraft>(EMPTY_PHOTO_DRAFT)

  return (
    <EntityForm
      meta={meta}
      title={t.form.editTitle}
      entityId={entity.id}
      submitLabel={t.common.save}
      defaultValues={entityToFormValues(entity, stored)}
      extraFields={
        <>
          {/* Keys are made right away (they must be shown once), so only for a saved sensor. */}
          {stored.sensor && <DeviceKeyPanel entityId={entity.id} sensor={stored.sensor} />}
          {existing.isPending ? (
            <p className="text-sm text-gray-400">{t.photos.loading}</p>
          ) : existing.isError ? (
            <p className="text-sm text-red-600">{t.photos.loadFailed(errorText(existing.error))}</p>
          ) : (
            <PhotoField existing={existing.data} draft={photos} onChange={setPhotos} />
          )}
        </>
      }
      onSubmit={async (input, extras) => {
        const saved = await update.mutateAsync({ id: entity.id, input })
        await saveFollowUps(meta, saved, extras, photos, stored)
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
