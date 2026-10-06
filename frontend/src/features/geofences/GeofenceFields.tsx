import { useEffect } from 'react'
import { useWatch, type Control, type FieldErrors, type UseFormRegister, type UseFormSetValue } from 'react-hook-form'
import { inputClass } from '../../components/formStyles'
import type { EntityFormOutput, EntityFormValues } from '../../schemas/entity'
import { MAX_RADIUS_M, MIN_RADIUS_M } from '../../schemas/geofence'
import { formatDistance } from './distance'
import { useZoneEditor } from './zoneEditor'

interface GeofenceFieldsProps {
  /** The entity being edited, or null for a new one (used to replace its circle with the preview). */
  entityId: string | null
  register: UseFormRegister<EntityFormValues>
  control: Control<EntityFormValues, unknown, EntityFormOutput>
  setValue: UseFormSetValue<EntityFormValues>
  errors: FieldErrors<EntityFormValues>['geofence']
}

const NUMBER_FIELDS = [
  { name: 'radius_m', label: 'Radius (m)', step: '100' },
  { name: 'center_latitude', label: 'Center latitude', step: 'any' },
  { name: 'center_longitude', label: 'Center longitude', step: 'any' },
] as const

/**
 * Operating zone inside the entity form. The circle is previewed on the map
 * while editing; it is saved together with the form.
 */
export function GeofenceFields({ entityId, register, control, setValue, errors }: GeofenceFieldsProps) {
  const zoneEditor = useZoneEditor()
  const { startPicking, cancelPicking, setPreview } = zoneEditor
  const zone = useWatch({ control, name: 'geofence' })
  const [pinLatitude, pinLongitude] = useWatch({ control, name: ['latitude', 'longitude'] })

  // Draw the zone as it is typed; remove the preview when the form closes.
  useEffect(() => {
    const drawable =
      zone.enabled &&
      Number.isFinite(zone.center_latitude) &&
      Number.isFinite(zone.center_longitude) &&
      zone.radius_m >= MIN_RADIUS_M &&
      zone.radius_m <= MAX_RADIUS_M
    setPreview(
      drawable
        ? {
            entityId,
            center_latitude: zone.center_latitude,
            center_longitude: zone.center_longitude,
            radius_m: zone.radius_m,
          }
        : null,
    )
  }, [entityId, zone.enabled, zone.center_latitude, zone.center_longitude, zone.radius_m, setPreview])
  useEffect(
    () => () => {
      setPreview(null)
      cancelPicking()
    },
    [setPreview, cancelPicking],
  )

  const setCenter = (latitude: number, longitude: number) => {
    const options = { shouldDirty: true, shouldValidate: true }
    setValue('geofence.center_latitude', latitude, options)
    setValue('geofence.center_longitude', longitude, options)
  }

  return (
    <fieldset className="rounded-lg border border-gray-200 p-3">
      <legend className="px-1 text-sm font-medium text-gray-700">Operating zone</legend>
      <label className="flex items-center gap-2 text-sm text-gray-700">
        <input type="checkbox" className="h-4 w-4 rounded border-gray-300" {...register('geofence.enabled')} />
        Limit to an operating zone
      </label>

      {zone.enabled && (
        <div className="mt-3 animate-fade-in space-y-2">
          <div className="grid grid-cols-3 gap-2">
            {NUMBER_FIELDS.map(({ name, label, step }) => {
              const error = errors?.[name]?.message
              const id = `geofence-${name}`
              return (
                <div key={name}>
                  <label htmlFor={id} className="mb-1 block text-xs font-medium text-gray-600">
                    {label}
                  </label>
                  <input
                    id={id}
                    type="number"
                    step={step}
                    inputMode="decimal"
                    aria-invalid={error ? true : undefined}
                    className={`${inputClass} px-2 py-1.5 text-xs`}
                    {...register(`geofence.${name}`, { valueAsNumber: true })}
                  />
                  {error && (
                    <p className="mt-1 text-xs text-red-600" role="alert">
                      {label.replace(' (m)', '')} {error}
                    </p>
                  )}
                </div>
              )
            })}
          </div>

          <p className="text-xs text-gray-500">
            {Number.isFinite(zone.radius_m) && zone.radius_m > 0 ? `${formatDistance(zone.radius_m)} around the center. ` : ''}
            The vehicle may still be moved outside; it is then flagged.
          </p>

          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => setCenter(pinLatitude, pinLongitude)}
              disabled={!Number.isFinite(pinLatitude) || !Number.isFinite(pinLongitude)}
              className="rounded-lg border border-gray-300 px-2.5 py-1 text-xs font-medium text-gray-700 transition-colors hover:bg-gray-50 disabled:opacity-50"
            >
              Use pin position
            </button>
            <button
              type="button"
              onClick={() => (zoneEditor.picking ? cancelPicking() : startPicking(setCenter))}
              aria-pressed={zoneEditor.picking}
              className={`rounded-lg px-2.5 py-1 text-xs font-medium transition-colors ${
                zoneEditor.picking
                  ? 'bg-blue-600 text-white hover:bg-blue-500'
                  : 'border border-gray-300 text-gray-700 hover:bg-gray-50'
              }`}
            >
              {zoneEditor.picking ? 'Click the map… (cancel)' : 'Pick on map'}
            </button>
          </div>
        </div>
      )}
    </fieldset>
  )
}
