import type { FieldErrors, UseFormRegister } from 'react-hook-form'
import { inputClass } from '../../components/formStyles'
import type { EntityFormValues } from '../../schemas/entity'
import type { MetricSpec } from '../../types/entity'

interface SensorFieldsProps {
  register: UseFormRegister<EntityFormValues>
  errors: FieldErrors<EntityFormValues>['sensor']
  metrics: MetricSpec[]
}

/** Metric choice inside the entity form; saved together with the form. */
export function SensorFields({ register, errors, metrics }: SensorFieldsProps) {
  const error = errors?.metric?.message
  return (
    <fieldset className="rounded-lg border border-gray-200 p-3">
      <legend className="px-1 text-sm font-medium text-gray-700">Sensor</legend>
      <label htmlFor="sensor-metric" className="mb-1 block text-xs font-medium text-gray-600">
        Measures
      </label>
      <select
        id="sensor-metric"
        aria-invalid={error ? true : undefined}
        className={`${inputClass} py-1.5 text-xs`}
        {...register('sensor.metric')}
      >
        <option value="">No sensor</option>
        {metrics.map((m) => (
          <option key={m.id} value={m.id}>
            {m.label} ({m.unit}, {m.min} to {m.max})
          </option>
        ))}
      </select>
      {error ? (
        <p className="mt-1 text-xs text-red-600" role="alert">
          Metric {error}
        </p>
      ) : (
        <p className="mt-1 text-xs text-gray-500">Values outside the range are rejected as a faulty reading.</p>
      )}
    </fieldset>
  )
}
