import type { FieldErrors, UseFormRegister } from 'react-hook-form'
import { inputClass } from '../../components/formStyles'
import { useI18n } from '../../i18n/context'
import type { EntityFormValues } from '../../schemas/entity'

interface InstallationFieldsProps {
  register: UseFormRegister<EntityFormValues>
  errors: FieldErrors<EntityFormValues>['installation']
  enabled: boolean
}

const DATE_FIELDS = [
  { name: 'started_on', labelKey: 'started', optional: false },
  { name: 'target_on', labelKey: 'target', optional: false },
  { name: 'completed_on', labelKey: 'completed', optional: true },
] as const

/** Installation schedule inside the entity form; saved together with the form. */
export function InstallationFields({ register, errors, enabled }: InstallationFieldsProps) {
  const { t, fieldError } = useI18n()
  return (
    <fieldset className="rounded-lg border border-gray-200 p-3">
      <legend className="px-1 text-sm font-medium text-gray-700">{t.installation.section}</legend>
      <label className="flex items-center gap-2 text-sm text-gray-700">
        <input type="checkbox" className="h-4 w-4 rounded border-gray-300" {...register('installation.enabled')} />
        {t.installation.track}
      </label>

      {enabled && (
        <div className="mt-3 grid animate-fade-in grid-cols-3 gap-2">
          {DATE_FIELDS.map(({ name, labelKey, optional }) => {
            const label = t.installation[labelKey]
            const error = errors?.[name]?.message
            const id = `installation-${name}`
            return (
              <div key={name}>
                <label htmlFor={id} className="mb-1 block text-xs font-medium text-gray-600">
                  {label}
                  {optional && <span className="ml-1 font-normal text-gray-400">{t.common.optional}</span>}
                </label>
                <input
                  id={id}
                  type="date"
                  aria-invalid={error ? true : undefined}
                  className={`${inputClass} px-2 py-1.5 text-xs`}
                  {...register(`installation.${name}`)}
                />
                {error && (
                  <p className="mt-1 text-xs text-red-600" role="alert">
                    {fieldError(label, error)}
                  </p>
                )}
              </div>
            )
          })}
          <p className="col-span-3 text-xs text-gray-500">
            {t.installation.completedHint}
          </p>
        </div>
      )}
    </fieldset>
  )
}
