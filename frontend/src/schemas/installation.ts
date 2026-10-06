import { z } from 'zod'
import type { Installation, InstallationInput } from '../types/installation'

// Mirrors the backend rules for installation dates (Validator.Installation).
// Messages match the backend so client and server errors read the same.

/** Form value: the dates as typed, plus whether the schedule is tracked at all. */
export interface InstallationFormValue {
  enabled: boolean
  started_on: string
  target_on: string
  completed_on: string
}

export const emptyInstallationValue = (): InstallationFormValue => ({
  enabled: false,
  started_on: '',
  target_on: '',
  completed_on: '',
})

export function toInstallationValue(installation: Installation | null): InstallationFormValue {
  if (!installation) return emptyInstallationValue()
  return {
    enabled: true,
    started_on: installation.started_on,
    target_on: installation.target_on,
    completed_on: installation.completed_on ?? '',
  }
}

/** True for a real calendar day written exactly as YYYY-MM-DD (2026-02-30 is not). */
export function isDate(s: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(s)) return false
  const d = new Date(`${s}T00:00:00Z`)
  return !Number.isNaN(d.getTime()) && d.toISOString().slice(0, 10) === s
}

/** Today's date in the browser's time zone, as YYYY-MM-DD. */
export function todayISO(now = new Date()): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`
}

/**
 * Validates an enabled schedule and returns the request body; a disabled
 * schedule becomes null. YYYY-MM-DD strings compare correctly as text.
 */
export const installationSchema = z
  .object({
    enabled: z.boolean(),
    started_on: z.string(),
    target_on: z.string(),
    completed_on: z.string(),
  })
  .transform((value, ctx): InstallationInput | null => {
    if (!value.enabled) return null
    const started = value.started_on.trim()
    const target = value.target_on.trim()
    const completed = value.completed_on.trim()
    const issue = (path: keyof InstallationFormValue, message: string) =>
      ctx.addIssue({ code: 'custom', path: [path], message })

    let ok = true
    for (const [path, v] of [['started_on', started], ['target_on', target]] as const) {
      if (v === '') {
        issue(path, 'is required')
        ok = false
      } else if (!isDate(v)) {
        issue(path, 'must be a date (YYYY-MM-DD)')
        ok = false
      }
    }
    if (completed !== '' && !isDate(completed)) {
      issue('completed_on', 'must be a date (YYYY-MM-DD)')
      ok = false
    }
    if (!ok) return z.NEVER

    if (target < started) {
      issue('target_on', 'must be on or after the start date')
      ok = false
    }
    if (completed !== '') {
      if (completed < started) {
        issue('completed_on', 'must be on or after the start date')
        ok = false
      } else if (completed > todayISO()) {
        issue('completed_on', 'cannot be in the future')
        ok = false
      }
    }
    if (!ok) return z.NEVER
    return { started_on: started, target_on: target, completed_on: completed === '' ? null : completed }
  })
