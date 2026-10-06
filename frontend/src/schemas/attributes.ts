import { z } from 'zod'
import type { JsonObject } from '../types/entity'

// The attributes editor. The backend contract is unchanged: attributes is a
// JSON object (or null). In the form it is edited as name/value rows; only
// data the rows cannot represent (nested objects or arrays) falls back to
// editing raw JSON, so nothing is lost.

export interface AttributeRow {
  key: string
  value: string
}

export type AttributesValue = { mode: 'rows'; rows: AttributeRow[] } | { mode: 'json'; text: string }

type Scalar = string | number | boolean | null

const isScalar = (v: unknown): v is Scalar =>
  v === null || typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean'

/**
 * Types a row value: numbers and true/false become JSON numbers/booleans,
 * everything else stays text. A number is only used when it prints back
 * exactly as typed, so "007" or "1e3" stay text.
 */
export function parseAttributeValue(text: string): string | number | boolean {
  const trimmed = text.trim()
  if (trimmed === 'true') return true
  if (trimmed === 'false') return false
  if (trimmed !== '' && String(Number(trimmed)) === trimmed) return Number(trimmed)
  return trimmed
}

/** Converts stored attributes to the editor's form value. */
export function toAttributesValue(attributes: JsonObject | null): AttributesValue {
  if (!attributes) return { mode: 'rows', rows: [] }
  const entries = Object.entries(attributes)
  if (!entries.every(([, v]) => isScalar(v))) {
    return { mode: 'json', text: JSON.stringify(attributes, null, 2) }
  }
  return {
    mode: 'rows',
    rows: entries.map(([key, value]) => ({ key, value: value === null ? '' : String(value) })),
  }
}

const rowsSchema = z.object({
  mode: z.literal('rows'),
  rows: z.array(z.object({ key: z.string(), value: z.string() })),
})

const jsonSchema = z.object({ mode: z.literal('json'), text: z.string() })

export const attributesSchema = z
  .discriminatedUnion('mode', [rowsSchema, jsonSchema])
  .transform((value, ctx): JsonObject | null => {
    if (value.mode === 'json') {
      const trimmed = value.text.trim()
      if (trimmed === '') return null
      let parsed: unknown
      try {
        parsed = JSON.parse(trimmed)
      } catch {
        ctx.addIssue({ code: 'custom', path: ['text'], message: 'must be valid JSON' })
        return z.NEVER
      }
      if (parsed === null) return null
      if (typeof parsed !== 'object' || Array.isArray(parsed)) {
        ctx.addIssue({ code: 'custom', path: ['text'], message: 'must be a JSON object' })
        return z.NEVER
      }
      return parsed as JsonObject
    }

    const result: JsonObject = {}
    const seen = new Set<string>()
    let valid = true
    value.rows.forEach((row, i) => {
      const key = row.key.trim()
      // A completely empty row is ignored, like an empty input.
      if (key === '' && row.value.trim() === '') return
      if (key === '') {
        ctx.addIssue({ code: 'custom', path: ['rows', i, 'key'], message: 'Name is required' })
        valid = false
        return
      }
      if (seen.has(key)) {
        ctx.addIssue({ code: 'custom', path: ['rows', i, 'key'], message: 'Name is used more than once' })
        valid = false
        return
      }
      seen.add(key)
      result[key] = parseAttributeValue(row.value)
    })
    if (!valid) return z.NEVER
    return Object.keys(result).length > 0 ? result : null
  })
