import type { DeviceKey, Readings, SensorConfig } from '../types/sensor'
import { http } from './client'

const base = (entityId: string) => `/entities/${encodeURIComponent(entityId)}`

/** The entity's sensor, or null when none was set up yet. */
export function getSensor(entityId: string): Promise<SensorConfig | null> {
  return http.get<SensorConfig | null>(`${base(entityId)}/sensor`)
}

export function putSensor(entityId: string, metric: string): Promise<SensorConfig> {
  return http.put<SensorConfig>(`${base(entityId)}/sensor`, { metric })
}

export function deleteSensor(entityId: string): Promise<void> {
  return http.delete(`${base(entityId)}/sensor`)
}

/** Creates (or replaces) the device API key. The key is only ever shown in this response. */
export function createDeviceKey(entityId: string): Promise<DeviceKey> {
  return http.post<DeviceKey>(`${base(entityId)}/sensor/key`, undefined)
}

/** Readings of the last `hours` hours, or null when the entity has no sensor. */
export function getReadings(entityId: string, hours = 24): Promise<Readings | null> {
  return http.get<Readings | null>(`${base(entityId)}/readings?hours=${hours}`)
}
