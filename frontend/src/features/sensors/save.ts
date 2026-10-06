import { useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { deleteSensor, putSensor } from '../../api/sensors'
import type { SensorConfig } from '../../types/sensor'
import { sensorKeys } from './hooks'

/**
 * Saves an entity's sensor metric after the entity itself was saved: PUT when
 * a metric is chosen (and changed), DELETE when "No sensor" was chosen for a
 * stored sensor. It returns an error message instead of throwing, because the
 * entity is already saved.
 */
export function useSaveSensor() {
  const queryClient = useQueryClient()
  return useCallback(
    async (entityId: string, metric: string | null, stored: SensorConfig | null): Promise<string | null> => {
      if (metric === (stored?.metric ?? null)) return null
      try {
        if (metric) await putSensor(entityId, metric)
        else await deleteSensor(entityId)
        return null
      } catch (error) {
        return error instanceof Error ? error.message : 'unknown error'
      } finally {
        await queryClient.invalidateQueries({ queryKey: sensorKeys.all })
      }
    },
    [queryClient],
  )
}
