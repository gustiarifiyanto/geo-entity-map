// Marker colors per status. The allowed values come from GET /api/meta; this
// map only adds colors and falls back gracefully for values added later.
// Labels live in the i18n dictionaries (useI18n().value).

const STATUS_COLORS: Record<string, string> = {
  active: '#16a34a',
  inactive: '#6b7280',
  maintenance: '#d97706',
}
const FALLBACK_STATUS_COLOR = '#2563eb'

export function statusColor(status: string): string {
  return STATUS_COLORS[status] ?? FALLBACK_STATUS_COLOR
}
