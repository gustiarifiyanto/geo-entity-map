// Display helpers. The allowed values come from GET /api/meta; these maps only
// add nicer labels/colors and fall back gracefully for values added later.

const STATUS_COLORS: Record<string, string> = {
  active: '#16a34a',
  inactive: '#6b7280',
  maintenance: '#d97706',
}
const FALLBACK_STATUS_COLOR = '#2563eb'

const LABELS: Record<string, string> = {
  iot_device: 'IoT Device',
}

export function statusColor(status: string): string {
  return STATUS_COLORS[status] ?? FALLBACK_STATUS_COLOR
}

/** "iot_device" -> "IoT Device", "some_new_type" -> "Some New Type". */
export function formatLabel(value: string): string {
  return (
    LABELS[value] ??
    value
      .split('_')
      .filter(Boolean)
      .map((word) => word[0].toUpperCase() + word.slice(1))
      .join(' ')
  )
}
