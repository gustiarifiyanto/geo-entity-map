// Mirrors the photo objects of the backend API.

export interface Photo {
  id: string
  entity_id: string
  /** Where the image is served, e.g. /api/photos/{id}. */
  url: string
  content_type: string
  size_bytes: number
  created_at: string
}
