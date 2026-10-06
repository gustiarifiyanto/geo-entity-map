import { useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { deleteGeofence, putGeofence } from '../../api/geofences'
import type { GeofenceInput } from '../../types/geofence'
import { geofenceKeys } from './hooks'

/**
 * Saves an entity's zone after the entity itself was saved: PUT when a zone
 * is set, DELETE when the zone was switched off. It returns an error message
 * instead of throwing, because the entity is already saved.
 */
export function useSaveGeofence() {
  const queryClient = useQueryClient()
  return useCallback(
    async (entityId: string, zone: GeofenceInput | null, hadZone: boolean): Promise<string | null> => {
      if (!zone && !hadZone) return null
      try {
        if (zone) await putGeofence(entityId, zone)
        else await deleteGeofence(entityId)
        return null
      } catch (error) {
        return error instanceof Error ? error.message : 'unknown error'
      } finally {
        await queryClient.invalidateQueries({ queryKey: geofenceKeys.all })
      }
    },
    [queryClient],
  )
}
