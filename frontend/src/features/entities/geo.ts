import type { LatLngBoundsExpression } from 'leaflet'

export const WORLD_BOUNDS: LatLngBoundsExpression = [
  [-90, -180],
  [90, 180],
]

/** Default view (Jakarta) used before any entity is loaded. */
export const DEFAULT_CENTER: [number, number] = [-6.2, 106.82]
export const DEFAULT_ZOOM = 11

/**
 * Wraps a longitude into [-180, 180]. Leaflet returns values outside that range
 * when the map is panned across the antimeridian, which the backend rejects.
 */
export function normalizeLongitude(lng: number): number {
  if (lng >= -180 && lng <= 180) return lng
  const wrapped = ((((lng + 180) % 360) + 360) % 360) - 180
  // Keep +180 as +180 instead of folding it to -180.
  return wrapped === -180 && lng > 0 ? 180 : wrapped
}

export function clampLatitude(lat: number): number {
  return Math.min(90, Math.max(-90, lat))
}

/** ~0.1 m precision; avoids sending long floating-point tails. */
const round6 = (n: number) => Math.round(n * 1e6) / 1e6

/** Converts a Leaflet position into coordinates the backend accepts. */
export function toCoordinates(latlng: { lat: number; lng: number }): {
  latitude: number
  longitude: number
} {
  return {
    latitude: round6(clampLatitude(latlng.lat)),
    longitude: round6(normalizeLongitude(latlng.lng)),
  }
}
