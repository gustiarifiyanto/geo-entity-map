import type { I18n } from './context'

/** "just now", "12 min ago", "3 h ago", or the full date after two days. */
export function timeAgo(iso: string, { t, locale }: Pick<I18n, 't' | 'locale'>): string {
  const minutes = Math.round((Date.now() - new Date(iso).getTime()) / 60_000)
  if (minutes < 1) return t.common.justNow
  if (minutes < 60) return t.common.minutesAgo(minutes)
  const hours = Math.round(minutes / 60)
  if (hours < 48) return t.common.hoursAgo(hours)
  return new Date(iso).toLocaleString(locale)
}
