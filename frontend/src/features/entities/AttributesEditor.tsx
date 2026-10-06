import { inputClass } from '../../components/formStyles'
import type { AttributeRow, AttributesValue } from '../../schemas/attributes'

/** Field errors for the attributes value, shaped like the value itself. */
export interface AttributesErrors {
  message?: string
  rows?: ({ key?: { message?: string } } | undefined)[]
  text?: { message?: string }
}

interface AttributesEditorProps {
  value: AttributesValue
  onChange: (value: AttributesValue) => void
  errors?: AttributesErrors
}

const smallInput = `${inputClass} py-1.5 text-xs`

/**
 * Name/value rows for the free-form attributes object. Values like 5000 or
 * true are stored as numbers/booleans, the rest as text. Nested data falls
 * back to a JSON textarea so it is never lost.
 */
export function AttributesEditor({ value, onChange, errors }: AttributesEditorProps) {
  if (value.mode === 'json') {
    const error = errors?.text?.message ?? errors?.message
    return (
      <div>
        <textarea
          id="attributes"
          rows={5}
          spellCheck={false}
          value={value.text}
          onChange={(event) => onChange({ mode: 'json', text: event.target.value })}
          aria-invalid={error ? true : undefined}
          aria-describedby="attributes-hint"
          className={`${inputClass} font-mono text-xs`}
        />
        <p id="attributes-hint" className="mt-1 text-xs text-gray-500">
          These attributes contain nested data, so they are edited as JSON.
        </p>
        {error && (
          <p className="mt-1 text-xs text-red-600" role="alert">
            Attributes {error}
          </p>
        )}
      </div>
    )
  }

  const rows = value.rows
  const update = (next: AttributeRow[]) => onChange({ mode: 'rows', rows: next })
  const setRow = (index: number, patch: Partial<AttributeRow>) =>
    update(rows.map((row, i) => (i === index ? { ...row, ...patch } : row)))

  return (
    <div>
      {rows.length > 0 && (
        <ul className="space-y-2">
          {rows.map((row, i) => {
            const keyError = errors?.rows?.[i]?.key?.message
            return (
              <li key={i} className="animate-fade-in">
                <div className="grid grid-cols-[2fr_3fr_auto] items-start gap-2">
                  <input
                    type="text"
                    value={row.key}
                    onChange={(event) => setRow(i, { key: event.target.value })}
                    placeholder="Name, e.g. plate"
                    aria-label={`Attribute ${i + 1} name`}
                    aria-invalid={keyError ? true : undefined}
                    autoComplete="off"
                    className={smallInput}
                  />
                  <input
                    type="text"
                    value={row.value}
                    onChange={(event) => setRow(i, { value: event.target.value })}
                    placeholder="Value, e.g. B 1234 XYZ"
                    aria-label={`Attribute ${i + 1} value`}
                    autoComplete="off"
                    className={smallInput}
                  />
                  <button
                    type="button"
                    onClick={() => update(rows.filter((_, j) => j !== i))}
                    aria-label={`Remove attribute ${i + 1}`}
                    title="Remove"
                    className="mt-1 rounded-md p-1 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600"
                  >
                    <svg viewBox="0 0 20 20" className="h-4 w-4" fill="currentColor" aria-hidden="true">
                      <path d="M6.28 5.22a.75.75 0 0 0-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 1 0 1.06 1.06L10 11.06l3.72 3.72a.75.75 0 1 0 1.06-1.06L11.06 10l3.72-3.72a.75.75 0 0 0-1.06-1.06L10 8.94 6.28 5.22Z" />
                    </svg>
                  </button>
                </div>
                {keyError && (
                  <p className="mt-1 text-xs text-red-600" role="alert">
                    {keyError}
                  </p>
                )}
              </li>
            )
          })}
        </ul>
      )}
      <button
        type="button"
        onClick={() => update([...rows, { key: '', value: '' }])}
        className={`${rows.length > 0 ? 'mt-2' : ''} text-xs font-medium text-gray-600 transition-colors hover:text-gray-900`}
      >
        + Add attribute
      </button>
      <p className="mt-1 text-xs text-gray-500">
        Extra properties. Numbers and true/false are saved as such; anything else as text.
      </p>
      {errors?.message && (
        <p className="mt-1 text-xs text-red-600" role="alert">
          Attributes {errors.message}
        </p>
      )}
    </div>
  )
}
