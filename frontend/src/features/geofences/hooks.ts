import { useQuery } from '@tanstack/react-query'
import { getGeofence, listGeofences } from '../../api/geofences'

export const geofenceKeys = {
  all: ['geofences'] as const,
  one: (entityId: string) => [...geofenceKeys.all, 'one', entityId] as const,
  list: () => [...geofenceKeys.all, 'list'] as const,
}

/** One entity's zone (null if none). Pass enabled=false for types without the capability. */
export function useGeofence(entityId: string, enabled: boolean) {
  return useQuery({ queryKey: geofenceKeys.one(entityId), queryFn: () => getGeofence(entityId), enabled })
}

export function useGeofences() {
  return useQuery({ queryKey: geofenceKeys.list(), queryFn: listGeofences })
}
