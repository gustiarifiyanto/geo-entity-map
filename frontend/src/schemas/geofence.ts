import { z } from 'zod'
import type { Geofence, GeofenceInput } from '../types/geofence'

// Mirrors the backend rules for operating zones (validation.Geofence).
// Messages match the backend so client and server errors read the same.

export const MIN_RADIUS_M = 100
export const MAX_RADIUS_M = 50_000

/** Form value; numbers come from number inputs, so an empty one is NaN. */
export interface GeofenceFormValue {
  enabled: boolean
  center_latitude: number
  center_longitude: number
  radius_m: number
}

/** A new zone starts around the entity's position with a 5 km radius. */
export function toGeofenceValue(geofence: Geofence | null, latitude: number, longitude: number): GeofenceFormValue {
  if (!geofence) return { enabled: false, center_latitude: latitude, center_longitude: longitude, radius_m: 5000 }
  return {
    enabled: true,
    center_latitude: geofence.center_latitude,
    center_longitude: geofence.center_longitude,
    radius_m: geofence.radius_m,
  }
}

/** An emptied number input is NaN; accept it here so the checks below can say "is required". */
export const numberOrNaN = z.union([z.number(), z.nan()])

const inRange = (v: number, limit: number) => Number.isFinite(v) && v >= -limit && v <= limit

/** Validates an enabled zone and returns the request body; a disabled zone becomes null. */
export const geofenceSchema = z
  .object({
    enabled: z.boolean(),
    center_latitude: numberOrNaN,
    center_longitude: numberOrNaN,
    radius_m: numberOrNaN,
  })
  .transform((v, ctx): GeofenceInput | null => {
    if (!v.enabled) return null
    const issue = (path: keyof GeofenceFormValue, message: string) =>
      ctx.addIssue({ code: 'custom', path: [path], message })

    if (Number.isNaN(v.center_latitude)) issue('center_latitude', 'is required')
    else if (!inRange(v.center_latitude, 90)) issue('center_latitude', 'must be between -90 and 90')
    if (Number.isNaN(v.center_longitude)) issue('center_longitude', 'is required')
    else if (!inRange(v.center_longitude, 180)) issue('center_longitude', 'must be between -180 and 180')
    if (Number.isNaN(v.radius_m)) issue('radius_m', 'is required')
    else if (!(v.radius_m >= MIN_RADIUS_M && v.radius_m <= MAX_RADIUS_M)) {
      issue('radius_m', `must be between ${MIN_RADIUS_M} and ${MAX_RADIUS_M}`)
    }

    if (ctx.issues.length > 0) return z.NEVER
    return { center_latitude: v.center_latitude, center_longitude: v.center_longitude, radius_m: v.radius_m }
  })
