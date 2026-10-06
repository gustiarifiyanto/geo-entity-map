import type { Entity } from '../../types/entity'

/** What the map search card filters by. Empty values match everything. */
export interface EntityFilterState {
  query: string
  status: string
}

export const EMPTY_FILTER: EntityFilterState = { query: '', status: '' }

/** Lowercase without accents, so "cafe" finds "Café" and "TRUCK" finds "Truck". */
export function normalizeText(s: string): string {
  return s.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase().trim()
}

export function isFilterActive(f: EntityFilterState): boolean {
  return f.query.trim() !== '' || f.status !== ''
}

/** Entities whose name contains the query and whose status matches, sorted by name. */
export function filterEntities(entities: Entity[], f: EntityFilterState): Entity[] {
  const q = normalizeText(f.query)
  return entities
    .filter((e) => (f.status === '' || e.status === f.status) && (q === '' || normalizeText(e.name).includes(q)))
    .sort((a, b) => a.name.localeCompare(b.name))
}
