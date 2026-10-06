// Mirrors the backend installation objects. Status values come from the
// backend; the frontend only adds labels and colors (with a fallback).

export interface Installation {
  entity_id: string
  started_on: string
  target_on: string
  completed_on: string | null
  status: string
  planned_days: number
  elapsed_days: number
  days_late: number
  updated_at: string
}

/** Body for PUT /api/entities/{id}/installation. Dates are YYYY-MM-DD. */
export interface InstallationInput {
  started_on: string
  target_on: string
  completed_on: string | null
}
