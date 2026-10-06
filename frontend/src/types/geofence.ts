// Mirrors the backend geofence (operating zone) objects.

export interface Geofence {
  entity_id: string
  center_latitude: number
  center_longitude: number
  radius_m: number
  /** Distance from the center to the entity's current position. */
  distance_m: number
  inside: boolean
  updated_at: string
}

/** Body for PUT /api/entities/{id}/geofence. */
export interface GeofenceInput {
  center_latitude: number
  center_longitude: number
  radius_m: number
}
