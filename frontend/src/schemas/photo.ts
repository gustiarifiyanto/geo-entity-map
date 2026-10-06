// Mirrors the backend photo rules (backend/internal/model/photo.go). Messages
// match the backend so client and server errors read the same.

export const MAX_PHOTO_BYTES = 5 * 1024 * 1024
export const MAX_PHOTOS_PER_ENTITY = 5
/** For the file picker only; the real check reads the file's bytes. */
export const PHOTO_ACCEPT = 'image/jpeg,image/png,image/webp'

const startsWith = (bytes: Uint8Array, signature: number[], offset = 0) =>
  signature.every((b, i) => bytes[offset + i] === b)

const ascii = (s: string) => [...s].map((c) => c.charCodeAt(0))

/**
 * Detects JPEG, PNG or WebP from the first bytes, like Go's
 * http.DetectContentType does on the backend. The file name and the
 * browser-reported type are not trusted.
 */
export function detectImageType(bytes: Uint8Array): string | null {
  if (startsWith(bytes, [0xff, 0xd8, 0xff])) return 'image/jpeg'
  if (startsWith(bytes, [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a])) return 'image/png'
  if (startsWith(bytes, ascii('RIFF')) && startsWith(bytes, ascii('WEBPVP'), 8)) return 'image/webp'
  return null
}

/** Returns the backend's error message for an invalid photo, or null when it is valid. */
export async function checkPhotoFile(file: File, existingCount: number): Promise<string | null> {
  if (existingCount >= MAX_PHOTOS_PER_ENTITY) {
    return `this entity already has the maximum of ${MAX_PHOTOS_PER_ENTITY} photos`
  }
  if (file.size === 0) return 'is required'
  if (file.size > MAX_PHOTO_BYTES) return `must be at most ${MAX_PHOTO_BYTES / 1024 / 1024} MB`
  const head = new Uint8Array(await file.slice(0, 16).arrayBuffer())
  if (!detectImageType(head)) return 'must be a JPEG, PNG or WebP image'
  return null
}

/** Turns a photo error fragment into a sentence, e.g. "is required" -> "Photo is required". */
export function photoErrorText(message: string): string {
  return /^(is|must) /.test(message) ? `Photo ${message}` : message[0].toUpperCase() + message.slice(1)
}
