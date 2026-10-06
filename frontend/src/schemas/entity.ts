import { z } from 'zod'
import type { Entity, Meta } from '../types/entity'
import { attributesSchema, toAttributesValue } from './attributes'
import { emptyInstallationValue, installationSchema, toInstallationValue } from './installation'
import type { Installation } from '../types/installation'
import { CAP_INSTALLATION, hasCapability } from './capabilities'

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

/** Attributes are edited as name/value rows (see ./attributes) and sent as an object or null. */
const attributes = attributesSchema

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
    // Raw here; validated below only when the chosen type has the capability,
    // so hidden fields of another type can never block saving.
    installation: z.object({
      enabled: z.boolean(),
      started_on: z.string(),
      target_on: z.string(),
      completed_on: z.string(),
    }),
  })
    .transform(({ installation, ...entity }, ctx) => {
      if (!hasCapability(meta, entity.type, CAP_INSTALLATION)) {
        return { entity, installation: null }
      }
      const result = installationSchema.safeParse(installation)
      if (!result.success) {
        for (const issue of result.error.issues) {
          ctx.addIssue({ code: 'custom', path: ['installation', ...issue.path], message: issue.message })
        }
        return z.NEVER
      }
      return { entity, installation: result.data }
    })
}

export type EntityFormSchema = ReturnType<typeof createEntityFormSchema>
/** Raw form values (attributes as editor rows). */
export type EntityFormValues = z.input<EntityFormSchema>
/**
 * Parsed values: the EntityInput to send, and the installation to PUT (null
 * when not tracked or when the type has no installation capability).
 */
export type EntityFormOutput = z.output<EntityFormSchema>

export function emptyFormValues(latitude: number, longitude: number): EntityFormValues {
  return {
    name: '',
    type: '',
    status: '',
    latitude,
    longitude,
    description: '',
    attributes: toAttributesValue(null),
    installation: emptyInstallationValue(),
  }
}

export function entityToFormValues(entity: Entity, installation: Installation | null = null): EntityFormValues {
  return {
    name: entity.name,
    type: entity.type,
    status: entity.status,
    latitude: entity.latitude,
    longitude: entity.longitude,
    description: entity.description,
    attributes: toAttributesValue(entity.attributes),
    installation: toInstallationValue(installation),
  }
}
