import { useEffect, useRef, useState, type ChangeEvent } from 'react'
import { useI18n } from '../../i18n/context'
import { checkPhotoFile, isFieldFragment, MAX_PHOTOS_PER_ENTITY, PHOTO_ACCEPT } from '../../schemas/photo'
import type { Photo } from '../../types/photo'
import type { PhotoDraft, StagedPhoto } from './draft'

interface PhotoFieldProps {
  /** Photos already saved (edit form); empty for a new entity. */
  existing: Photo[]
  draft: PhotoDraft
  onChange: (draft: PhotoDraft) => void
}

let nextKey = 0

/** A rejected file: kept as the raw (English) fragment and translated when shown. */
interface PhotoProblem {
  file?: string
  message: string
}

/**
 * Photo picker for the entity form. Adding or removing only changes the
 * draft; the panel uploads and deletes after the entity is saved.
 */
export function PhotoField({ existing, draft, onChange }: PhotoFieldProps) {
  const { t, fieldError, sentence } = useI18n()
  const input = useRef<HTMLInputElement>(null)
  const [errors, setErrors] = useState<PhotoProblem[]>([])

  // Revoke preview URLs when the form closes.
  const latest = useRef(draft)
  useEffect(() => {
    latest.current = draft
  }, [draft])
  useEffect(() => () => latest.current.added.forEach((p) => URL.revokeObjectURL(p.previewUrl)), [])

  const kept = existing.filter((p) => !draft.removedIds.includes(p.id))
  const total = kept.length + draft.added.length
  const isFull = total >= MAX_PHOTOS_PER_ENTITY

  const onFilesChosen = async (event: ChangeEvent<HTMLInputElement>) => {
    const files = [...(event.target.files ?? [])]
    // Reset so choosing the same file again still fires a change event.
    event.target.value = ''

    const problems: PhotoProblem[] = []
    const added: StagedPhoto[] = []
    for (const file of files) {
      const problem = await checkPhotoFile(file, total + added.length)
      if (problem) {
        problems.push({ file: file.name, message: problem })
        continue
      }
      added.push({ key: `staged-${nextKey++}`, file, previewUrl: URL.createObjectURL(file) })
    }
    setErrors(problems)
    if (added.length > 0) onChange({ ...draft, added: [...draft.added, ...added] })
  }

  const dropStaged = (photo: StagedPhoto) => {
    URL.revokeObjectURL(photo.previewUrl)
    onChange({ ...draft, added: draft.added.filter((p) => p.key !== photo.key) })
  }

  const toggleRemove = (id: string) => {
    const removing = !draft.removedIds.includes(id)
    // Undoing a removal must still respect the limit.
    if (!removing && isFull) {
      setErrors([{ message: `this entity already has the maximum of ${MAX_PHOTOS_PER_ENTITY} photos` }])
      return
    }
    setErrors([])
    onChange({
      ...draft,
      removedIds: removing ? [...draft.removedIds, id] : draft.removedIds.filter((r) => r !== id),
    })
  }

  const summary = [
    draft.added.length > 0 && t.photos.toAdd(draft.added.length),
    draft.removedIds.length > 0 && t.photos.toRemove(draft.removedIds.length),
  ]
    .filter(Boolean)
    .join(', ')

  return (
    <div>
      <div className="mb-1 flex items-baseline justify-between">
        <span className="text-sm font-medium text-gray-700">
          {t.photos.field} <span className="ml-1 font-normal text-gray-400">{t.common.optional}</span>
        </span>
        <span className="text-xs text-gray-400">
          {total}/{MAX_PHOTOS_PER_ENTITY}
        </span>
      </div>

      <ul className="grid grid-cols-4 gap-2">
        {existing.map((photo) => {
          const removed = draft.removedIds.includes(photo.id)
          return (
            <Thumb
              key={photo.id}
              src={photo.url}
              dimmed={removed}
              badge={removed ? t.photos.removed : undefined}
              actionLabel={removed ? t.photos.keep : t.photos.remove}
              actionIcon={removed ? 'undo' : 'remove'}
              onAction={() => toggleRemove(photo.id)}
            />
          )
        })}
        {draft.added.map((photo) => (
          <Thumb
            key={photo.key}
            src={photo.previewUrl}
            badge={t.photos.newBadge}
            actionLabel={t.photos.removeFile(photo.file.name)}
            actionIcon="remove"
            onAction={() => dropStaged(photo)}
          />
        ))}
        {!isFull && (
          <li>
            <button
              type="button"
              onClick={() => input.current?.click()}
              aria-label={t.photos.add}
              title={t.photos.add}
              className="flex aspect-square w-full items-center justify-center rounded-lg border-2 border-dashed border-gray-300 text-gray-400 transition-colors hover:border-gray-400 hover:text-gray-600"
            >
              <svg viewBox="0 0 20 20" className="h-6 w-6" fill="currentColor" aria-hidden="true">
                <path d="M10.75 4.75a.75.75 0 0 0-1.5 0v4.5h-4.5a.75.75 0 0 0 0 1.5h4.5v4.5a.75.75 0 0 0 1.5 0v-4.5h4.5a.75.75 0 0 0 0-1.5h-4.5v-4.5Z" />
              </svg>
            </button>
          </li>
        )}
      </ul>

      <input
        ref={input}
        type="file"
        accept={PHOTO_ACCEPT}
        multiple
        onChange={(event) => void onFilesChosen(event)}
        className="hidden"
        aria-hidden="true"
        tabIndex={-1}
      />
      <p className="mt-1 text-xs text-gray-500">
        {t.photos.rules} {summary ? t.photos.pending(summary) : t.photos.savedWithForm}
      </p>
      {errors.map(({ file, message }) => {
        const text = isFieldFragment(message) ? fieldError(t.photos.single, message) : sentence(message)
        return (
          <p key={`${file}-${message}`} className="mt-1 text-xs text-red-600" role="alert">
            {file ? `${file}: ${text}` : text}
          </p>
        )
      })}
    </div>
  )
}

interface ThumbProps {
  src: string
  dimmed?: boolean
  badge?: string
  actionLabel: string
  actionIcon: 'remove' | 'undo'
  onAction: () => void
}

function Thumb({ src, dimmed = false, badge, actionLabel, actionIcon, onAction }: ThumbProps) {
  return (
    <li className="relative animate-fade-in">
      <img
        src={src}
        alt=""
        className={`aspect-square w-full rounded-lg object-cover ring-1 ring-black/5 transition-opacity ${
          dimmed ? 'opacity-30 grayscale' : ''
        }`}
      />
      {badge && (
        <span
          className={`absolute bottom-1 left-1 rounded px-1 text-[10px] font-medium text-white ${
            dimmed ? 'bg-red-600' : 'bg-gray-900/70'
          }`}
        >
          {badge}
        </span>
      )}
      <button
        type="button"
        onClick={onAction}
        aria-label={actionLabel}
        title={actionLabel}
        className="absolute top-1 right-1 rounded-full bg-black/60 p-1 text-white transition-colors hover:bg-black/80"
      >
        {actionIcon === 'remove' ? (
          <svg viewBox="0 0 20 20" className="h-3.5 w-3.5" fill="currentColor" aria-hidden="true">
            <path d="M6.28 5.22a.75.75 0 0 0-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 1 0 1.06 1.06L10 11.06l3.72 3.72a.75.75 0 1 0 1.06-1.06L11.06 10l3.72-3.72a.75.75 0 0 0-1.06-1.06L10 8.94 6.28 5.22Z" />
          </svg>
        ) : (
          <svg viewBox="0 0 20 20" className="h-3.5 w-3.5" fill="currentColor" aria-hidden="true">
            <path
              fillRule="evenodd"
              d="M7.793 2.232a.75.75 0 0 1-.025 1.06L3.622 7.25h10.003a5.375 5.375 0 0 1 0 10.75H10.75a.75.75 0 0 1 0-1.5h2.875a3.875 3.875 0 0 0 0-7.75H3.622l4.146 3.957a.75.75 0 0 1-1.036 1.085l-5.5-5.25a.75.75 0 0 1 0-1.085l5.5-5.25a.75.75 0 0 1 1.06.025Z"
              clipRule="evenodd"
            />
          </svg>
        )}
      </button>
    </li>
  )
}
