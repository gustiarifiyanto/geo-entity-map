import { createContext, useContext } from 'react'
import { ApiError } from '../api/client'
import type { Messages } from './en'
import { fieldError, sentence, valueLabel, type Lang } from './translate'

export interface I18n {
  lang: Lang
  setLang: (lang: Lang) => void
  /** The dictionary of the chosen language. */
  t: Messages
  /** BCP 47 locale for dates and numbers. */
  locale: string
  /** Label of a value sent by the backend (type, status, role, metric, …). */
  value: (value: string, fallback?: string) => string
  /** A field error: the label plus the (translated) backend/zod fragment. */
  fieldError: (label: string, fragment: string) => string
  /** A standalone message from the backend, translated when known. */
  sentence: (text: string) => string
  /** A readable, translated message for any error thrown by an API call. */
  errorText: (error: unknown) => string
}

export const LOCALES: Record<Lang, string> = { en: 'en-US', id: 'id-ID' }

export function makeI18n(lang: Lang, t: Messages, setLang: (lang: Lang) => void): I18n {
  return {
    lang,
    setLang,
    t,
    locale: LOCALES[lang],
    value: (v, fallback) => valueLabel(lang, v, fallback),
    fieldError: (label, fragment) => fieldError(lang, label, fragment),
    sentence: (text) => sentence(lang, text),
    errorText: (error) => {
      if (error instanceof ApiError) {
        // Client-side fallbacks (no usable body) are written in English in
        // the API client; translate them by their code/status instead.
        if (error.code === 'network_error') return t.errors.network
        if (error.code === 'http_error') {
          if (error.status === 401) return t.errors.unauthorized
          if (error.status === 403) return t.errors.forbidden
          if (error.status === 404) return t.errors.notFound
          if (error.status >= 502 && error.status <= 504) return t.errors.network
          return t.errors.server
        }
        if (error.status === 422 && Object.keys(error.fields).length > 0) return t.errors.fixFields
        return sentence(lang, error.message)
      }
      return error instanceof Error ? sentence(lang, error.message) : t.common.somethingWrong
    },
  }
}

export const I18nContext = createContext<I18n | null>(null)

export function useI18n(): I18n {
  const i18n = useContext(I18nContext)
  if (!i18n) throw new Error('useI18n must be used within <I18nProvider>')
  return i18n
}
