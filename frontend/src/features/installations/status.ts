// Display helpers for installation statuses. The values come from the
// backend; unknown ones fall back to a neutral color and a generated label.
import { formatLabel } from '../entities/labels'

const STATUS_STYLE: Record<string, { color: string; label: string; order: number }> = {
  overdue: { color: '#dc2626', label: 'Overdue', order: 0 },
  in_progress: { color: '#2563eb', label: 'In progress', order: 1 },
  scheduled: { color: '#6b7280', label: 'Scheduled', order: 2 },
  completed_late: { color: '#d97706', label: 'Completed late', order: 3 },
  completed_on_time: { color: '#16a34a', label: 'Completed on time', order: 4 },
}

export function installationStatusColor(status: string): string {
  return STATUS_STYLE[status]?.color ?? '#475569'
}

export function installationStatusLabel(status: string): string {
  return STATUS_STYLE[status]?.label ?? formatLabel(status)
}

/** Sort key: most urgent first; unknown statuses last. */
export function installationStatusOrder(status: string): number {
  return STATUS_STYLE[status]?.order ?? 99
}

const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? '' : 's'}`

export const daysText = (n: number) => plural(n, 'day')

export function formatDay(iso: string): string {
  // Parse as a calendar day, not an instant, so the time zone cannot shift it.
  const [y, m, d] = iso.split('-').map(Number)
  const date = new Date(y, m - 1, d)
  return Number.isNaN(date.getTime()) ? iso : date.toLocaleDateString()
}
