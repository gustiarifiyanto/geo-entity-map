import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { I18nContext, makeI18n } from './context'
import { en } from './en'
import { id } from './id'
import type { Lang } from './translate'

const STORAGE_KEY = 'geo-entity-map.lang'
const DICTIONARIES = { en, id }

/** The saved choice, else the browser language (Indonesian → id), else English. */
function initialLang(): Lang {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved === 'en' || saved === 'id') return saved
  } catch {
    // Storage can be unavailable (private mode, blocked site data).
  }
  return navigator.language.toLowerCase().startsWith('id') ? 'id' : 'en'
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(initialLang)

  const setLang = useCallback((next: Lang) => {
    setLangState(next)
    try {
      localStorage.setItem(STORAGE_KEY, next)
    } catch {
      // Not remembered, but the switch still works for this visit.
    }
  }, [])

  useEffect(() => {
    document.documentElement.lang = lang
  }, [lang])

  const i18n = useMemo(() => makeI18n(lang, DICTIONARIES[lang], setLang), [lang, setLang])
  return <I18nContext.Provider value={i18n}>{children}</I18nContext.Provider>
}
