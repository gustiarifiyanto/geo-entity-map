import { z } from 'zod'
import type { Entity, JsonObject, Meta } from '../types/entity'

// Mirrors the backend rules in backend/internal/validation. Messages match the
// backend so client and server errors read the same.

export const NAME_MAX = 100
export const DESCRIPTION_MAX = 500

/** Counts Unicode code points, like Go's rune count (not UTF-16 units). */
export const charCount = (s: string) => [...s].length

const maxChars = (max: number) =>
  [(s: string) => charCount(s) <= max, `must be at most ${max} characters`] as const

const coordinate = (limit: number) => {
  const range = `must be between -${limit} and ${limit}`
  return z
    .number({
      // An empty number input yields NaN (valueAsNumber); ±Infinity is a range
      // error, matching the backend message.
      error: (issue) =>
        issue.input === undefined || Number.isNaN(issue.input)
          ? 'is required'
          : typeof issue.input === 'number'
            ? range
            : 'must be a number',
    })
    .min(-limit, range)
    .max(limit, range)
}

const oneOf = (allowed: readonly string[]) =>
  z
    .string()
    .min(1, { error: 'is required', abort: true })
    .refine((v) => allowed.includes(v), `must be one of: ${allowed.join(', ')}`)

/** Attributes are edited as JSON text and sent as an object (or null when empty). */
const attributes = z.string().transform((text, ctx): JsonObject | null => {
  const trimmed = text.trim()
  if (trimmed === '') return null

  let parsed: unknown
  try {
    parsed = JSON.parse(trimmed)
  } catch {
    ctx.addIssue({ code: 'custom', message: 'must be valid JSON' })
    return z.NEVER
  }
  if (parsed === null) return null
  if (typeof parsed !== 'object' || Array.isArray(parsed)) {
    ctx.addIssue({ code: 'custom', message: 'must be a JSON object' })
    return z.NEVER
  }
  return parsed as JsonObject
})

/** Builds the entity form schema; allowed types/statuses come from GET /api/meta. */
export function createEntityFormSchema(meta: Meta) {
  return z.object({
    name: z
      .string()
      .trim()
      .min(1, { error: 'is required', abort: true })
      .refine(...maxChars(NAME_MAX)),
    type: oneOf(meta.types),
    status: oneOf(meta.statuses),
    latitude: coordinate(90),
    longitude: coordinate(180),
    description: z
      .string()
      .trim()
      .refine(...maxChars(DESCRIPTION_MAX)),
    attributes,
  })
}

export type EntityFormSchema = ReturnType<typeof createEntityFormSchema>
/** Raw form values (attributes as JSON text). */
export type EntityFormValues = z.input<EntityFormSchema>
/** Parsed values, ready to send as EntityInput. */
export type EntityFormOutput = z.output<EntityFormSchema>

export function emptyFormValues(latitude: number, longitude: number): EntityFormValues {
  return {
    name: '',
    type: '',
    status: '',
    latitude,
    longitude,
    description: '',
    attributes: '',
  }
}

export function entityToFormValues(entity: Entity): EntityFormValues {
  return {
    name: entity.name,
    type: entity.type,
    status: entity.status,
    latitude: entity.latitude,
    longitude: entity.longitude,
    description: entity.description,
    attributes: entity.attributes ? JSON.stringify(entity.attributes, null, 2) : '',
  }
}
