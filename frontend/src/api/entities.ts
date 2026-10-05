import type { Entity, EntityFilter, EntityInput, LocationInput, Meta } from '../types/entity'
import { http } from './client'

const entityPath = (id: string) => `/entities/${encodeURIComponent(id)}`

export function getMeta(): Promise<Meta> {
  return http.get<Meta>('/meta')
}

export function listEntities(filter: EntityFilter = {}): Promise<Entity[]> {
  const params = new URLSearchParams()
  if (filter.type) params.set('type', filter.type)
  if (filter.status) params.set('status', filter.status)
  const query = params.toString()
  return http.get<Entity[]>(`/entities${query ? `?${query}` : ''}`)
}

export function getEntity(id: string): Promise<Entity> {
  return http.get<Entity>(entityPath(id))
}

export function createEntity(input: EntityInput): Promise<Entity> {
  return http.post<Entity>('/entities', input)
}

export function updateEntity(id: string, input: EntityInput): Promise<Entity> {
  return http.put<Entity>(entityPath(id), input)
}

export function updateEntityLocation(id: string, input: LocationInput): Promise<Entity> {
  return http.patch<Entity>(`${entityPath(id)}/location`, input)
}

export function deleteEntity(id: string): Promise<void> {
  return http.delete(entityPath(id))
}
