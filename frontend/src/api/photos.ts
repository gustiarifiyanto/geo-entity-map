import type { Photo } from '../types/photo'
import { http } from './client'

export function listPhotos(entityId: string): Promise<Photo[]> {
  return http.get<Photo[]>(`/entities/${encodeURIComponent(entityId)}/photos`)
}

export function uploadPhoto(entityId: string, file: File): Promise<Photo> {
  const form = new FormData()
  form.append('photo', file)
  return http.post<Photo>(`/entities/${encodeURIComponent(entityId)}/photos`, form)
}

export function deletePhoto(id: string): Promise<void> {
  return http.delete(`/photos/${encodeURIComponent(id)}`)
}
