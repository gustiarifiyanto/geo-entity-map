import { useI18n } from '../i18n/context'
import type { Lang } from '../i18n/translate'

const LANGS: { id: Lang; label: string; name: string }[] = [
  { id: 'id', label: 'ID', name: 'Bahasa Indonesia' },
  { id: 'en', label: 'EN', name: 'English' },
]

/** ID | EN toggle; the language names stay in their own language. */
export function LanguageSwitch() {
  const { lang, setLang, t } = useI18n()
  return (
    <div role="radiogroup" aria-label={t.language.label} className="flex rounded-lg bg-gray-900/5 p-0.5 text-xs">
      {LANGS.map((l) => (
        <button
          key={l.id}
          type="button"
          role="radio"
          aria-checked={lang === l.id}
          title={l.name}
          lang={l.id}
          onClick={() => setLang(l.id)}
          className={`rounded-md px-2 py-0.5 font-medium transition-colors ${
            lang === l.id ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-800'
          }`}
        >
          {l.label}
        </button>
      ))}
    </div>
  )
}
