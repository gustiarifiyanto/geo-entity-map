import { useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { deleteInstallation, putInstallation } from '../../api/installations'
import type { InstallationInput } from '../../types/installation'
import { installationKeys } from './hooks'

/**
 * Saves an entity's installation schedule after the entity itself was saved:
 * PUT when tracked, DELETE when tracking was switched off. It returns an error
 * message instead of throwing, because the entity is already saved.
 */
export function useSaveInstallation() {
  const queryClient = useQueryClient()
  return useCallback(
    async (
      entityId: string,
      installation: InstallationInput | null,
      hadSchedule: boolean,
    ): Promise<string | null> => {
      if (!installation && !hadSchedule) return null
      try {
        if (installation) await putInstallation(entityId, installation)
        else await deleteInstallation(entityId)
        return null
      } catch (error) {
        return error instanceof Error ? error.message : 'unknown error'
      } finally {
        await queryClient.invalidateQueries({ queryKey: installationKeys.all })
      }
    },
    [queryClient],
  )
}
