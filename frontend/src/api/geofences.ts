import type { Geofence, GeofenceInput } from '../types/geofence'
import { http } from './client'

const geofencePath = (entityId: string) => `/entities/${encodeURIComponent(entityId)}/geofence`

/** The entity's operating zone, or null when none was set yet. */
export function getGeofence(entityId: string): Promise<Geofence | null> {
  return http.get<Geofence | null>(geofencePath(entityId))
}

export function putGeofence(entityId: string, input: GeofenceInput): Promise<Geofence> {
  return http.put<Geofence>(geofencePath(entityId), input)
}

export function deleteGeofence(entityId: string): Promise<void> {
  return http.delete(geofencePath(entityId))
}

export function listGeofences(): Promise<Geofence[]> {
  return http.get<Geofence[]>('/geofences')
}
