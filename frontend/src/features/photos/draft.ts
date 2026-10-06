import { useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { deletePhoto, uploadPhoto } from '../../api/photos'
import { photoKeys } from './hooks'

/** A photo picked in the form but not uploaded yet. */
export interface StagedPhoto {
  key: string
  file: File
  /** Object URL for the preview; revoked when the photo is dropped. */
  previewUrl: string
}

/**
 * Photo changes made in the entity form. Nothing reaches the server until the
 * form is saved; Cancel simply discards the draft.
 */
export interface PhotoDraft {
  added: StagedPhoto[]
  /** Existing photos marked for deletion. */
  removedIds: string[]
}

export const EMPTY_PHOTO_DRAFT: PhotoDraft = { added: [], removedIds: [] }

/**
 * Applies a draft after the entity itself was saved: deletions first (so they
 * free room under the 5-photo limit), then uploads, one at a time. It returns
 * how many changes failed instead of throwing, because the entity is already
 * saved at this point.
 */
export function useSavePhotoDraft() {
  const queryClient = useQueryClient()
  return useCallback(
    async (entityId: string, draft: PhotoDraft): Promise<{ failed: number; firstError?: string }> => {
      let failed = 0
      let firstError: string | undefined
      const record = (error: unknown) => {
        failed++
        firstError ??= error instanceof Error ? error.message : 'unknown error'
      }

      for (const id of draft.removedIds) {
        await deletePhoto(id).catch(record)
      }
      for (const photo of draft.added) {
        await uploadPhoto(entityId, photo.file).catch(record)
      }
      if (draft.removedIds.length > 0 || draft.added.length > 0) {
        await queryClient.invalidateQueries({ queryKey: photoKeys.list(entityId) })
      }
      return { failed, firstError }
    },
    [queryClient],
  )
}
