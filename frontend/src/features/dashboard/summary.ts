import type { Entity } from '../../types/entity'

export interface Count {
  value: string
  count: number
}

/**
 * Counts entities per value of `key`. Every allowed value (from /api/meta) is
 * listed, even with 0; values the data has but meta does not (e.g. right after
 * a backend change) are appended so nothing is hidden.
 */
export function countBy(entities: Entity[], key: 'type' | 'status', allowed: string[]): Count[] {
  const counts = new Map<string, number>(allowed.map((value) => [value, 0]))
  for (const entity of entities) {
    counts.set(entity[key], (counts.get(entity[key]) ?? 0) + 1)
  }
  return [...counts].map(([value, count]) => ({ value, count }))
}
