// Mirrors the backend sensor objects.
import type { MetricSpec } from './entity'

export interface SensorConfig {
  entity_id: string
  metric: string
  has_api_key: boolean
  key_created_at: string | null
  updated_at: string
}

/** Returned once, when a device key is created. */
export interface DeviceKey {
  api_key: string
  key_created_at: string
}

export interface Reading {
  value: number
  recorded_at: string
}

export interface Readings {
  metric: MetricSpec
  /** Oldest first. */
  readings: Reading[]
  latest: Reading | null
}
