import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useMemo, type ReactNode } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import {
  charCount,
  createEntityFormSchema,
  DESCRIPTION_MAX,
  type EntityFormOutput,
  type EntityFormValues,
} from '../../schemas/entity'
import { inputClass } from '../../components/formStyles'
import type { EntityInput, Meta } from '../../types/entity'
import { formatLabel } from './labels'
import { applyServerError } from './serverErrors'

interface EntityFormProps {
  meta: Meta
  title: string
  submitLabel: string
  defaultValues: EntityFormValues
  /** Location picked on the map; when it changes the coordinate fields follow. */
  pickedLatitude?: number
  pickedLongitude?: number
  locationHint?: string
  /** Extra controls shown after the regular fields (e.g. the photo picker). */
  extraFields?: ReactNode
  onSubmit: (input: EntityInput) => Promise<void>
  onCancel: () => void
}

export function EntityForm({
  meta,
  title,
  submitLabel,
  defaultValues,
  pickedLatitude,
  pickedLongitude,
  locationHint,
  extraFields,
  onSubmit,
  onCancel,
}: EntityFormProps) {
  const schema = useMemo(() => createEntityFormSchema(meta), [meta])
  const {
    register,
    handleSubmit,
    setValue,
    setError,
    control,
    formState: { errors, isSubmitting },
  } = useForm<EntityFormValues, unknown, EntityFormOutput>({
    resolver: zodResolver(schema),
    // Read once on mount: later prop changes never reset what the user typed.
    defaultValues,
    // Validate on submit only: the name field is autofocused, so validating on
    // blur would show "is required" as soon as the user clicks the map to pick
    // a location. After the first submit, errors update as the user types.
    mode: 'onSubmit',
    reValidateMode: 'onChange',
  })

  useEffect(() => {
    if (pickedLatitude === undefined || pickedLongitude === undefined) return
    const options = { shouldDirty: true, shouldValidate: true }
    setValue('latitude', pickedLatitude, options)
    setValue('longitude', pickedLongitude, options)
  }, [pickedLatitude, pickedLongitude, setValue])

  const description = useWatch({ control, name: 'description' })

  const submit = handleSubmit(async (values) => {
    try {
      await onSubmit(values)
    } catch (error) {
      applyServerError(error, setError)
    }
  })

  return (
    <form
      onSubmit={submit}
      noValidate
      aria-label={title}
      className="flex max-h-full flex-col overflow-hidden rounded-xl bg-white shadow-xl ring-1 ring-black/5"
    >
      <header className="flex items-center justify-between border-b border-gray-100 p-4">
        <h2 className="text-lg font-semibold text-gray-900">{title}</h2>
        <button
          type="button"
          onClick={onCancel}
          aria-label="Cancel"
          className="rounded-md p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-700"
        >
          <svg viewBox="0 0 20 20" className="h-5 w-5" fill="currentColor" aria-hidden="true">
            <path d="M6.28 5.22a.75.75 0 0 0-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 1 0 1.06 1.06L10 11.06l3.72 3.72a.75.75 0 1 0 1.06-1.06L11.06 10l3.72-3.72a.75.75 0 0 0-1.06-1.06L10 8.94 6.28 5.22Z" />
          </svg>
        </button>
      </header>

      <div className="flex-1 space-y-4 overflow-y-auto p-4">
        {errors.root?.server && (
          <div role="alert" className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 ring-1 ring-red-200">
            {errors.root.server.message}
          </div>
        )}

        <FormField id="name" label="Name" error={errors.name?.message}>
          <input
            id="name"
            type="text"
            autoFocus
            autoComplete="off"
            aria-invalid={errors.name ? true : undefined}
            className={inputClass}
            {...register('name')}
          />
        </FormField>

        <div className="grid grid-cols-2 gap-3">
          <FormField id="type" label="Type" error={errors.type?.message}>
            <select
              id="type"
              aria-invalid={errors.type ? true : undefined}
              className={inputClass}
              {...register('type')}
            >
              <option value="" disabled>
                Select type
              </option>
              {meta.types.map((type) => (
                <option key={type} value={type}>
                  {formatLabel(type)}
                </option>
              ))}
            </select>
          </FormField>
          <FormField id="status" label="Status" error={errors.status?.message}>
            <select
              id="status"
              aria-invalid={errors.status ? true : undefined}
              className={inputClass}
              {...register('status')}
            >
              <option value="" disabled>
                Select status
              </option>
              {meta.statuses.map((status) => (
                <option key={status} value={status}>
                  {formatLabel(status)}
                </option>
              ))}
            </select>
          </FormField>
        </div>

        <div>
          <div className="grid grid-cols-2 gap-3">
            <FormField id="latitude" label="Latitude" error={errors.latitude?.message}>
              <input
                id="latitude"
                type="number"
                step="any"
                inputMode="decimal"
                aria-invalid={errors.latitude ? true : undefined}
                className={inputClass}
                {...register('latitude', { valueAsNumber: true })}
              />
            </FormField>
            <FormField id="longitude" label="Longitude" error={errors.longitude?.message}>
              <input
                id="longitude"
                type="number"
                step="any"
                inputMode="decimal"
                aria-invalid={errors.longitude ? true : undefined}
                className={inputClass}
                {...register('longitude', { valueAsNumber: true })}
              />
            </FormField>
          </div>
          {locationHint && <p className="mt-1 text-xs text-gray-500">{locationHint}</p>}
        </div>

        <FormField
          id="description"
          label="Description"
          optional
          error={errors.description?.message}
          aside={`${charCount(description ?? '')}/${DESCRIPTION_MAX}`}
        >
          <textarea
            id="description"
            rows={3}
            aria-invalid={errors.description ? true : undefined}
            className={inputClass}
            {...register('description')}
          />
        </FormField>

        <FormField id="attributes" label="Attributes" optional error={errors.attributes?.message}>
          <textarea
            id="attributes"
            rows={4}
            spellCheck={false}
            placeholder={'{\n  "plate": "B 1234 XYZ"\n}'}
            aria-invalid={errors.attributes ? true : undefined}
            aria-describedby="attributes-hint"
            className={`${inputClass} font-mono text-xs`}
            {...register('attributes')}
          />
          <p id="attributes-hint" className="mt-1 text-xs text-gray-500">
            A JSON object with any extra properties.
          </p>
        </FormField>

        {extraFields}
      </div>

      <footer className="flex gap-2 border-t border-gray-100 p-4">
        <button
          type="button"
          onClick={onCancel}
          className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={isSubmitting}
          className="flex-1 rounded-lg bg-gray-900 px-3 py-2 text-sm font-medium text-white hover:bg-gray-700 disabled:opacity-60"
        >
          {isSubmitting ? 'Saving…' : submitLabel}
        </button>
      </footer>
    </form>
  )
}

interface FormFieldProps {
  id: string
  label: string
  error?: string
  optional?: boolean
  aside?: string
  children: ReactNode
}

function FormField({ id, label, error, optional, aside, children }: FormFieldProps) {
  return (
    <div>
      <div className="mb-1 flex items-baseline justify-between">
        <label htmlFor={id} className="text-sm font-medium text-gray-700">
          {label}
          {optional && <span className="ml-1 font-normal text-gray-400">(optional)</span>}
        </label>
        {aside && <span className="text-xs text-gray-400">{aside}</span>}
      </div>
      {children}
      {error && (
        <p className="mt-1 text-xs text-red-600" role="alert">
          {/* Messages are fragments shared with the backend, e.g. "is required". */}
          {label} {error}
        </p>
      )}
    </div>
  )
}
