import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createDeviceKey, getReadings, getSensor } from '../../api/sensors'

/** How often readings refresh while shown (the simulator writes once a minute). */
export const READINGS_REFRESH_MS = 30_000

export const sensorKeys = {
  all: ['sensors'] as const,
  one: (entityId: string) => [...sensorKeys.all, 'one', entityId] as const,
  readings: (entityId: string) => [...sensorKeys.all, 'readings', entityId] as const,
}

/** One entity's sensor (null if none). Pass enabled=false for types without the capability. */
export function useSensor(entityId: string, enabled: boolean) {
  return useQuery({ queryKey: sensorKeys.one(entityId), queryFn: () => getSensor(entityId), enabled })
}

export function useReadings(entityId: string, enabled: boolean) {
  return useQuery({
    queryKey: sensorKeys.readings(entityId),
    queryFn: () => getReadings(entityId),
    enabled,
    refetchInterval: READINGS_REFRESH_MS,
    refetchIntervalInBackground: false,
  })
}

export function useCreateDeviceKey(entityId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => createDeviceKey(entityId),
    // Refresh has_api_key / key_created_at; the key itself is never cached.
    onSettled: () => queryClient.invalidateQueries({ queryKey: sensorKeys.one(entityId) }),
  })
}
