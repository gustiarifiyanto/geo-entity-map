import { useEffect, useRef } from 'react'
import { useI18n } from '../../i18n/context'
import type { Photo } from '../../types/photo'

interface PhotoViewerProps {
  photos: Photo[]
  /** Index of the photo shown; the viewer is open while this is not null. */
  index: number | null
  title: string
  onIndexChange: (index: number) => void
  onClose: () => void
}

/**
 * Full-size photo on a native <dialog> (focus trap, Escape, above the map).
 * Arrow keys and the side buttons move between photos.
 */
export function PhotoViewer({ photos, index, title, onIndexChange, onClose }: PhotoViewerProps) {
  const { t } = useI18n()
  const ref = useRef<HTMLDialogElement>(null)
  const open = index !== null && photos[index] !== undefined

  useEffect(() => {
    const dialog = ref.current
    if (!dialog) return
    if (open && !dialog.open) dialog.showModal()
    if (!open && dialog.open) dialog.close()
  }, [open])

  const photo = open ? photos[index] : null
  const hasMany = photos.length > 1
  const go = (step: number) => {
    if (index !== null) onIndexChange((index + step + photos.length) % photos.length)
  }

  return (
    <dialog
      ref={ref}
      aria-label={index !== null ? t.photos.viewerLabel(title, index + 1, photos.length) : title}
      onCancel={(event) => {
        // Escape key: let React state drive closing.
        event.preventDefault()
        onClose()
      }}
      onClick={(event) => {
        // Clicking the backdrop targets the <dialog> element itself.
        if (event.target === ref.current) onClose()
      }}
      onKeyDown={(event) => {
        if (!hasMany) return
        if (event.key === 'ArrowRight') go(1)
        if (event.key === 'ArrowLeft') go(-1)
      }}
      className="m-auto max-h-[90vh] w-[calc(100%-2rem)] max-w-4xl overflow-hidden rounded-xl bg-gray-950 p-0 shadow-2xl backdrop:bg-black/70"
    >
      {photo && index !== null && (
        <>
          <div className="flex items-center justify-between gap-3 px-4 py-2 text-sm text-gray-200">
            <span className="truncate">
              {title}
              {hasMany && <span className="ml-2 text-gray-400">{`${index + 1} / ${photos.length}`}</span>}
            </span>
            <button
              type="button"
              onClick={onClose}
              aria-label={t.photos.close}
              autoFocus
              className="rounded-md p-1 text-gray-400 transition-colors hover:bg-white/10 hover:text-white"
            >
              <svg viewBox="0 0 20 20" className="h-5 w-5" fill="currentColor" aria-hidden="true">
                <path d="M6.28 5.22a.75.75 0 0 0-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 1 0 1.06 1.06L10 11.06l3.72 3.72a.75.75 0 1 0 1.06-1.06L11.06 10l3.72-3.72a.75.75 0 0 0-1.06-1.06L10 8.94 6.28 5.22Z" />
              </svg>
            </button>
          </div>

          <div className="relative flex items-center justify-center bg-black">
            <img
              // key: restart the fade when switching photos.
              key={photo.id}
              src={photo.url}
              alt={t.photos.viewerLabel(title, index + 1, photos.length)}
              className="max-h-[calc(90vh-2.75rem)] w-auto animate-fade-in object-contain"
            />
            {hasMany && (
              <>
                <NavButton label={t.photos.previous} side="left" onClick={() => go(-1)} />
                <NavButton label={t.photos.next} side="right" onClick={() => go(1)} />
              </>
            )}
          </div>
        </>
      )}
    </dialog>
  )
}

function NavButton({ label, side, onClick }: { label: string; side: 'left' | 'right'; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      className={`absolute top-1/2 -translate-y-1/2 ${side === 'left' ? 'left-2' : 'right-2'} rounded-full bg-black/50 p-2 text-white transition-colors hover:bg-black/80`}
    >
      <svg viewBox="0 0 20 20" className="h-5 w-5" fill="currentColor" aria-hidden="true">
        <path
          fillRule="evenodd"
          d={
            side === 'left'
              ? 'M11.78 5.22a.75.75 0 0 1 0 1.06L8.06 10l3.72 3.72a.75.75 0 1 1-1.06 1.06l-4.25-4.25a.75.75 0 0 1 0-1.06l4.25-4.25a.75.75 0 0 1 1.06 0Z'
              : 'M8.22 5.22a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.75.75 0 0 1-1.06-1.06L11.94 10 8.22 6.28a.75.75 0 0 1 0-1.06Z'
          }
          clipRule="evenodd"
        />
      </svg>
    </button>
  )
}
