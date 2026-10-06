import type { UseFormSetError } from 'react-hook-form'
import { ApiError } from '../../api/client'
import type { EntityFormValues } from '../../schemas/entity'

const FORM_FIELDS = [
  'name',
  'type',
  'status',
  'latitude',
  'longitude',
  'description',
  'attributes',
] as const satisfies readonly (keyof EntityFormValues)[]

type FormField = (typeof FORM_FIELDS)[number]

const isFormField = (field: string): field is FormField =>
  (FORM_FIELDS as readonly string[]).includes(field)

/**
 * Shows a failed submit on the form: 422 field errors go under their inputs
 * (backend keys are the JSON field names, which match the form fields);
 * anything else becomes a form-level error, translated with errorText.
 */
export function applyServerError(
  error: unknown,
  setError: UseFormSetError<EntityFormValues>,
  errorText: (error: unknown) => string,
) {
  if (error instanceof ApiError && error.isValidation) {
    const unmapped: string[] = []
    let first = true
    for (const [field, message] of Object.entries(error.fields)) {
      if (isFormField(field)) {
        setError(field, { type: 'server', message }, { shouldFocus: first })
        first = false
      } else {
        unmapped.push(`${field} ${message}`)
      }
    }
    if (unmapped.length > 0 || first) {
      setError('root.server', {
        type: 'server',
        message: unmapped.length > 0 ? unmapped.join('; ') : errorText(error),
      })
    }
    return
  }

  setError('root.server', {
    type: 'server',
    message: errorText(error),
  })
}
