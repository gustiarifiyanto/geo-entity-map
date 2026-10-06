// The same great-circle formula and Earth radius as the backend
// (model.DistanceMeters), so the map and the API agree on "inside".
const EARTH_RADIUS_M = 6_371_008.8

export function distanceMeters(lat1: number, lng1: number, lat2: number, lng2: number): number {
  const rad = Math.PI / 180
  const dLat = (lat2 - lat1) * rad
  const dLng = (lng2 - lng1) * rad
  const a = Math.sin(dLat / 2) ** 2 + Math.cos(lat1 * rad) * Math.cos(lat2 * rad) * Math.sin(dLng / 2) ** 2
  return 2 * EARTH_RADIUS_M * Math.asin(Math.min(1, Math.sqrt(a)))
}

/** 850 m, 1.2 km, 12 km. */
export function formatDistance(meters: number, locale?: string): string {
  if (meters < 1000) return `${Math.round(meters).toLocaleString(locale)} m`
  const km = meters / 1000
  return `${km.toLocaleString(locale, { maximumFractionDigits: km < 10 ? 1 : 0 })} km`
}
