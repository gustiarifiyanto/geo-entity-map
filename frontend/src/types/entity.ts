// Shared types mirroring the backend API contract.
// Allowed types/statuses are NOT hardcoded here: they come from GET /api/meta.

export type JsonObject = { [key: string]: unknown }

export interface Entity {
  id: string
  name: string
  type: string
  status: string
  latitude: number
  longitude: number
  description: string
  attributes: JsonObject | null
  created_at: string
  updated_at: string
}

export interface Meta {
  types: string[]
  statuses: string[]
  /** Features only some types have, e.g. { facility: ["installation"] }. */
  capabilities: Record<string, string[]>
  /** What sensors can measure; units and valid ranges come from the backend. */
  metrics: MetricSpec[]
}

export interface MetricSpec {
  id: string
  label: string
  unit: string
  min: number
  max: number
}

/** Body for POST /api/entities and PUT /api/entities/{id}. */
export interface EntityInput {
  name: string
  type: string
  status: string
  latitude: number
  longitude: number
  description: string
  attributes: JsonObject | null
}

/** Body for PATCH /api/entities/{id}/location. */
export interface LocationInput {
  latitude: number
  longitude: number
}

export interface EntityFilter {
  type?: string
  status?: string
}
