// Display helpers for installation statuses. The values come from the
// backend; labels come from the i18n dictionaries (i18n.value). Unknown
// statuses fall back to a neutral color and the end of the list.

const STATUS_STYLE: Record<string, { color: string; order: number }> = {
  overdue: { color: '#dc2626', order: 0 },
  in_progress: { color: '#2563eb', order: 1 },
  scheduled: { color: '#6b7280', order: 2 },
  completed_late: { color: '#d97706', order: 3 },
  completed_on_time: { color: '#16a34a', order: 4 },
}

export function installationStatusColor(status: string): string {
  return STATUS_STYLE[status]?.color ?? '#475569'
}

/** Sort key: most urgent first; unknown statuses last. */
export function installationStatusOrder(status: string): number {
  return STATUS_STYLE[status]?.order ?? 99
}

export function formatDay(iso: string, locale: string): string {
  // Parse as a calendar day, not an instant, so the time zone cannot shift it.
  const [y, m, d] = iso.split('-').map(Number)
  const date = new Date(y, m - 1, d)
  return Number.isNaN(date.getTime()) ? iso : date.toLocaleDateString(locale)
}
